package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Br0ce/containerctl/pkg/ui"
)

var (
	host         string
	username     string
	identityFile string
	dockerSocket string
	askPassword  bool

	rootCmd = &cobra.Command{
		Use:   "containerctl",
		Short: "A TUI for monitoring and managing local or remote containers.",
		Long: `A TUI for monitoring and managing containers. containerctl connects to the Docker engine API
either locally or remotely via SSH.

Behavior:

  - If --host is not provided, containerctl connects to the local Docker API socket in the following order of precedence:
	1. If DOCKER_HOST environment variable is set, it is used as the host.
	2. If --docker-socket is provided, it is used as the host.
	3. Otherwise, the default unix socket (unix:///var/run/docker.sock) is used.
  - To determine the Docker API-compatible host
  - If --host is provided, the connection is done securely over SSH.
	- Host key verification is enforced using your known_hosts file.
	- The username for SSH authentication is determined in the following order of precedence:
		1. --username argument
		2. username embedded in host as user@hostname
		3. If no username is provided the current system user is used.
	- If --ask-password is set to true, the user will be prompted for their SSH password instead of using key-based 
	  authentication. This option takes precedence.
	- If --ask-password is not provided, a private SSH key is used for authentication in the following 
	  order of precedence:
		1. --identity-file Path to the private SSH key to use for authentication. Must be inside the user's .ssh directory.
		2. If not set, the a default key (~/.ssh/id_ed25519, id_rsa or id_ecdsa) is used.
    - To use a Docker engine API-compatible service, provide a --docker-socket URL (e.g. unix:///run/user/<UID>/podman/podman.sock ).   
	  If --docker-socket is not provided the default (unix:///var/run/docker.sock) is used.
	- The Docker Engine API socket is determined in the following order of precedence 
	  (DOCKER_HOST environment variable is ignored when --host is provided):
		1. If --docker-socket is provided, it is used as the docker unix socket.
		2. Otherwise, the default unix socket (//var/run/docker.sock) is used.

Examples:

  # Connect to local Docker
  containerctl

  # Connect to remote host using default SSH key from config
  containerctl --host my-host

  # Connect to remote host using with SSH key
  containerctl --host my-host:23 --identity-file ~/.ssh/id_rsa

  # Connect to remote host with username embedded in host and password prompted
  containerctl --host username@my-host --ask-password true
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := ui.Config{
				Host:         host,
				DockerSocket: dockerSocket,
				Username:     username,
				IdentityFile: identityFile,
				AskPassword:  askPassword,
			}
			ui, err := ui.New(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: create ui %s\n", err.Error())
				os.Exit(1)
			}
			defer func() {
				err := ui.Close()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: close ui %s\n", err.Error())
				}
			}()

			err = ui.Run(cmd.Context())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: run ui %s\n", err.Error())
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "localhost", "Docker host to connect to, e.g. hostname, username@hostname, hostname:port.")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username for SSH authentication. Overrides username in host if both are provided.")
	rootCmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Path to the private SSH key to use for authentication")
	rootCmd.PersistentFlags().StringVar(&dockerSocket, "docker-socket", "/var/run/docker.sock", "The Docker API socket to connect to. Ignored if --host is provided.")
	rootCmd.PersistentFlags().BoolVar(&askPassword, "ask-password", false, "Prompt for SSH password authentication")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

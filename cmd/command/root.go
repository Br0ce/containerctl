package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Br0ce/containerctl/tui"
	"github.com/Br0ce/containerctl/version"
)

var (
	host         string
	dockerhost   string
	username     string
	identityFile string
	askPassword  bool

	rootCmd = &cobra.Command{
		Use:     "cctl",
		Version: version.Version,
		Short:   "A TUI for monitoring and managing local or remote containers.",
		Long: `A TUI for monitoring and managing containers. cctl connects to the Docker Engine API
either locally or remotely via SSH.

Behavior:

  - If --host is not provided, cctl connects to the local Docker API-compatible socket in the following
    order of precedence:
	1. --docker-host, e.g. unix:///run/user/<UID>/podman/podman.sock or tcp://127.0.0.1:2375.
	2. The DOCKER_HOST environment variable. DOCKER_API_VERSION, DOCKER_CERT_PATH and DOCKER_TLS_VERIFY
	   are used as well.
	3. The default unix socket (unix:///var/run/docker.sock).
  - If --host is provided, the connection is done securely over SSH.
	- Host key verification is enforced using your known_hosts file.
	  If a ~/.ssh/config entry for host, username or port is found, it is used and overrides the
	  corresponding values provided by the command line arguments.
	- The username for SSH authentication is determined in the following order of precedence:
		1. --username argument
		2. username embedded in host as user@hostname
		3. If no username is provided the current system user is used.
	- Authentication is attempted in the following order:
		1. If --ask-password is set, the user is prompted for their SSH password. This option takes precedence.
		2. The SSH agent, if SSH_AUTH_SOCK is set.
		3. A private SSH key, prompting for its passphrase if it is encrypted:
			a. --identity-file, the name of a private SSH key inside the ~/.ssh directory.
			b. The IdentityFile from the ~/.ssh/config entry for the host.
			c. A default key (~/.ssh/id_ed25519, id_rsa or id_ecdsa).
	- The Docker socket on the remote host is --docker-host (a path or unix:// URL), or
	  /var/run/docker.sock if it is not provided. Of the Docker environment variables only
	  DOCKER_API_VERSION is used.

Examples:

  # Connect to local Docker
  cctl

  # Connect to local Podman
  cctl --docker-host unix:///run/user/1000/podman/podman.sock

  # Connect to remote host using default SSH settings
  cctl --host my-host

  # Connect to remote host using an SSH key
  cctl --host my-host:23 --identity-file ~/.ssh/id_rsa

  # Connect to remote host with username embedded in host and password prompted
  cctl --host username@my-host --ask-password
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := tui.Config{
				Host:         host,
				DockerHost:   dockerhost,
				Username:     username,
				IdentityFile: identityFile,
				AskPassword:  askPassword,
			}
			ui, err := tui.New(cfg)
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
	rootCmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Private SSH key to use for authentication. Must be inside the ~/.ssh directory.")
	rootCmd.PersistentFlags().StringVar(&dockerhost, "docker-host", "", "Docker API socket to connect to, on the remote host if --host is provided. Defaults to DOCKER_HOST locally and /var/run/docker.sock remotely.")
	rootCmd.PersistentFlags().BoolVar(&askPassword, "ask-password", false, "Prompt for SSH password authentication")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

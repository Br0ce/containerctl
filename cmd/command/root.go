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
	rootCmd      = &cobra.Command{
		Use:   "containerctl",
		Short: "A TUI for monitoring and managing local or remote containers.",
		Long: `A TUI for monitoring and managing containers. containerctl connects to the Docker engine
either locally or remotely via SSH.

Behavior:

  - If --host is not provided, containerctl connects to the local Docker socket.
  - If --host is provided:
	- Connection is done securely over SSH.
	- Host key verification is enforced using your known_hosts file.
	- If no username is provided, the private SSH key from your SSH config is used for authentication.
		1. --identity-file 
			Path to the private SSH key to use for authentication. Must be inside the user's .ssh directory.
		2. If not set, the default key (~/.ssh/id_ed25519, id_rsa, etc.) is used.
	- If a username is provided, you will be prompted to enter your password securely (input hidden).
		Username must only be provided in one place:
		1. --user 
		2. username embedded in host as user@hostname

Examples:

  # Connect to local Docker
  containerctl

  # Connect to remote host using default SSH key from config
  containerctl --host my-host

  # Connect to remote host using with SSH key
  containerctl --host my-host --identity-file ~/.ssh/id_rsa

  # Connect to remote host with username (password prompted)
  containerctl --host my-host --user username

  # Connect to remote host with username embedded in host (password prompted)
  containerctl --host username@my-host`,
		Run: func(cmd *cobra.Command, args []string) {
			ui, err := ui.New(host, username, identityFile)
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
	rootCmd.PersistentFlags().StringVar(&username, "user", "", "Username for SSH authentication. Overrides username in host if both are provided.")
	rootCmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Path to the private SSH key to use for authentication")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Br0ce/containerctl/pkg/ui"
)

var rootCmd = &cobra.Command{
	Use:   "containerctl",
	Short: "A TUI for monitoring and managing containers.",
	Long:  "A TUI for monitoring and managing local or remote containers.",
	Run: func(cmd *cobra.Command, args []string) {
		ui, err := ui.New()
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

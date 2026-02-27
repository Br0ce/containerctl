package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Br0ce/cctl/pkg/api/console"
)

var rootCmd = &cobra.Command{
	Use:   "cctl",
	Short: "A UI for monitoring and managing containers.",
	Long:  "A UI for monitoring and managing local or remote containers.",
	Run: func(cmd *cobra.Command, args []string) {
		err := console.Run(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
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

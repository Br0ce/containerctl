package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cctl",
	Short: "A terminal UI for monitoring and managing containers.",
	Long:  "A terminal UI for monitoring and managing local or remote containers.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello cctl. Use --help for more information.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

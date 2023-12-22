package premia

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "premia",
	Short: "premia - a CLI to setup financial infrastructure",
	Long:  `premia is a CLI to setup common infrastructure for asset management firms`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "There was an error while executing '%s'", err)
		os.Exit(1)
	}
}

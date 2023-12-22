package premia

import (
	"fmt"
	"log"

	"github.com/premia-ai/cli/internal/config"
	"github.com/premia-ai/cli/internal/migrations"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a financial database",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := config.SetupConfigDir()
		if err != nil {
			log.Fatal("SetupConfigDir:", err)
		}
		err = migrations.Initialize()
		if err != nil {
			log.Fatal("Initialize:", err)
		}
		err = migrations.Seed()
		if err != nil {
			log.Fatal("Seed:", err)
		}

		fmt.Println("Successfully initialized database!")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

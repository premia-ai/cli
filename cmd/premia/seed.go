package premia

import (
	"fmt"
	"log"

	"github.com/premia-ai/cli/internal/migrations"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "init",
	Short: "Seed your financial database with instrument data",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		err := migrations.Seed()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Successfully seeded database!")
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

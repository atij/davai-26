package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management utilities",
}

var resetDbCmd = &cobra.Command{
	Use:   "reset",
	Short: "Wipe all data and recreate the database schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("⚠️  WARNING: This will delete ALL data in the database '%s'.\n", cfg.Database.Name)
		fmt.Print("Are you sure? Type 'YES' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) != "YES" {
			fmt.Println("❌ Reset cancelled.")
			return nil
		}

		database, err := db.Connect(cfg.Database)
		if err != nil {
			return fmt.Errorf("db connect: %w", err)
		}
		defer database.Close()

		fmt.Print("Wiping and recreating schema... ")
		if err := db.Reset(database); err != nil {
			fmt.Println("FAILED")
			return err
		}
		fmt.Println("SUCCESS")

		logger.Info("database reset complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(resetDbCmd)
}

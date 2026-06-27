package cmd

import (
	"fmt"

	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage and validate config",
}

var validateConfigCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check all required fields are set and DB is reachable",
	RunE: func(cmd *cobra.Command, args []string) error {
		// PersistentPreRunE already loaded and validated basic config
		fmt.Println("✅ Config structure: VALID")

		database, err := db.Connect(cfg.Database)
		if err != nil {
			return fmt.Errorf("❌ Database connection: FAILED: %w", err)
		}
		defer database.Close()
		fmt.Println("✅ Database connection: SUCCESS")

		if err := db.Migrate(database); err != nil {
			return fmt.Errorf("❌ Database migration: FAILED: %w", err)
		}
		fmt.Println("✅ Database schema: UP TO DATE")

		logger.Info("config validation complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateConfigCmd)
}

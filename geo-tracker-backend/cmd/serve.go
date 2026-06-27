package cmd

import (
	"context"

	"github.com/adoreme/geo-tracker/internal/api"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start JSON API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		return api.StartServer(context.Background(), cfg.Serve, database, logger)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

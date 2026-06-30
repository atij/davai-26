package cmd

import (
	"context"

	"github.com/adoreme/geo-tracker/internal/adk"
	"github.com/adoreme/geo-tracker/internal/api"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start JSON API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		resultRepo := db.NewResultRepo(database)
		toolSet := adk.NewToolSet(resultRepo)
		sessionStore := adk.NewMySQLSessionStore(resultRepo)
		strategyAgent, err := adk.NewStrategyAgent(ctx, cfg.ADK, toolSet, sessionStore)
		if err != nil {
			logger.Fatal("strategy agent init failed", zap.Error(err))
		}

		return api.StartServer(ctx, cfg.Serve, database, logger, strategyAgent)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

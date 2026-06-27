package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfg    *config.Config
	logger *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:   "geo-tracker",
	Short: "GEO Tracker probes AI providers for brand visibility signals",
	Long: `GEO Tracker is a Go CLI app that probes AI providers (Claude, ChatGPT, Perplexity, Gemini)
with curated prompts, extracts brand visibility signals, stores results in MySQL, and serves
a JSON API for a dashboard.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		c, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := c.Validate(); err != nil {
			return err
		}
		cfg = c

		// Init logger
		logLevel := cfg.App.LogLevel
		if logLevel == "" {
			logLevel = "info"
		}

		var zapCfg zap.Config
		if strings.ToLower(logLevel) == "debug" {
			zapCfg = zap.NewDevelopmentConfig()
			zapCfg.EncoderConfig.TimeKey = "ts"
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		} else {
			zapCfg = zap.NewProductionConfig()
		}
		zapCfg.OutputPaths = []string{"stderr"}
		zapCfg.ErrorOutputPaths = []string{"stderr"}

		l, err := zapCfg.Build()
		if err != nil {
			return fmt.Errorf("init logger: %w", err)
		}
		logger = l

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Root flags will be added here later
}

package cmd

import (
	"fmt"

	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
)

var (
	resultsRunID    uint64
	resultsBrand    string
	resultsProvider string
	resultsType     string
	resultsLimit    int
)

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Query and display results",
}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Print latest run summary per brand",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		repo := db.NewResultRepo(database)

		brands := []string{resultsBrand}
		if resultsBrand == "" {
			brands, _ = repo.GetBrands()
		}

		for _, b := range brands {
			summary, err := repo.GetBrandSummary(b)
			if err != nil {
				continue
			}
			fmt.Printf("\n── %s Summary (Run #%d) ──────────────────\n", b, summary.RunID)
			fmt.Printf("  Mention Rate:    %.1f%%\n", summary.MentionRate)
			fmt.Printf("  Sentiment Score: %.2f\n", summary.SentimentScore)
			fmt.Printf("  Top Provider:    %s\n", summary.TopProvider)
			fmt.Println("──────────────────────────────────────────────────")
		}

		return nil
	},
}

var trendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Show mention rate trend for a brand",
	RunE: func(cmd *cobra.Command, args []string) error {
		if resultsBrand == "" {
			return fmt.Errorf("--brand is required")
		}
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		repo := db.NewResultRepo(database)
		trend, err := repo.GetBrandTrend(resultsBrand, resultsLimit)
		if err != nil {
			return err
		}

		fmt.Printf("\nTrend for %s (last %d runs)\n", resultsBrand, resultsLimit)
		fmt.Printf("%-10s | %-20s | %-12s\n", "Run ID", "Date", "Mention Rate")
		fmt.Println("----------------------------------------------------------")
		for _, t := range trend {
			fmt.Printf("%-10d | %-20s | %-11.1f%%\n", t.RunID, t.StartedAt.Format("2006-01-02 15:04"), t.MentionRate)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resultsCmd)
	resultsCmd.AddCommand(summaryCmd)
	resultsCmd.AddCommand(trendCmd)

	resultsCmd.PersistentFlags().Uint64Var(&resultsRunID, "run-id", 0, "specific run (default: latest)")
	resultsCmd.PersistentFlags().StringVar(&resultsBrand, "brand", "", "filter by brand")
	resultsCmd.PersistentFlags().StringVar(&resultsProvider, "provider", "", "filter by provider")
	resultsCmd.PersistentFlags().StringVar(&resultsType, "type", "organic", "organic|comparison|all")
	resultsCmd.PersistentFlags().IntVar(&resultsLimit, "last", 10, "number of runs for trend")
}

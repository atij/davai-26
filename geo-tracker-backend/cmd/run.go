package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/adoreme/geo-tracker/internal/agent"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/adoreme/geo-tracker/internal/providers"
	"github.com/adoreme/geo-tracker/internal/runner"
	"github.com/adoreme/geo-tracker/internal/scoring"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	runBrands    []string
	runProviders []string
	dryRun       bool
	verbose      bool
	exitCode     bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Fire all active prompts at all enabled providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return fmt.Errorf("db connect: %w", err)
		}
		defer database.Close()

		if err := db.Migrate(database); err != nil {
			return fmt.Errorf("db migrate: %w", err)
		}

		promptRepo := db.NewPromptRepo(database)
		resultRepo := db.NewResultRepo(database)

		prompts, err := promptRepo.ListActive()
		if err != nil {
			return fmt.Errorf("list active prompts: %w", err)
		}

		if len(prompts) == 0 {
			logger.Warn("no active prompts found")
			return nil
		}

		allProviders := providers.NewProviders(*cfg)
		var enabledProviders []providers.Provider
		if len(runProviders) > 0 {
			for _, pName := range runProviders {
				for _, p := range allProviders {
					if p.Name() == pName {
						enabledProviders = append(enabledProviders, p)
					}
				}
			}
		} else {
			enabledProviders = allProviders
		}

		brands := runBrands
		if len(brands) == 0 {
			for _, b := range cfg.Brands {
				brands = append(brands, b.Name)
			}
		}

		samples := cfg.Runner.SamplesPerPrompt
		if samples <= 0 {
			samples = 1
		}

		logger.Info("starting run",
			zap.Int("prompts", len(prompts)),
			zap.Int("providers", len(enabledProviders)),
			zap.Int("brands", len(brands)),
			zap.Int("samples", samples))

		run := db.Run{
			PromptCount: len(prompts),
			BrandCount:  len(brands),
			SampleCount: samples,
			Status:      "running",
		}

		if !dryRun {
			if err := resultRepo.CreateRun(&run); err != nil {
				return fmt.Errorf("create run: %w", err)
			}
		}

		rn := runner.NewRunner(*cfg, logger)
		fmt.Printf("Running %d jobs across %d workers...\n", len(prompts)*len(enabledProviders)*len(brands)*samples, cfg.Runner.Workers)
		results := rn.RunAll(context.Background(), prompts, enabledProviders, brands)
		fmt.Printf("\nProcessing results and calculating scores...\n")

		var totalCost float64
		successCount := 0
		for i := range results {
			if !dryRun {
				results[i].RunID = run.ID
				if err := resultRepo.InsertResult(&results[i]); err != nil {
					logger.Error("failed to insert result", zap.Error(err))
				} else {
					successCount++
					totalCost += results[i].CostUSD
				}
			}

			if verbose {
				fmt.Printf("\n--- [%s] %s (Sample %d) ---\nPrompt: %d\nResponse: %s\n",
					results[i].Provider, results[i].Brand, results[i].SampleIndex, results[i].PromptID, results[i].RawResponse)
			}
		}

		if !dryRun {
			// 1. Calculate and store stability scores
			for _, b := range brands {
				for _, pr := range enabledProviders {
					for _, p := range prompts {
						var promptSamples []db.Result
						for _, r := range results {
							if r.Brand == b && r.Provider == pr.Name() && r.PromptID == p.ID {
								promptSamples = append(promptSamples, r)
							}
						}
						if len(promptSamples) > 0 {
							score := scoring.CalcStabilityScore(promptSamples)
							resultRepo.InsertStabilityScore(&score)
						}
					}
				}
			}

			// 2. Explainer & Recommender (Simplified placeholders as per agent code)
			// In real implementation, we'd fetch previous run, calculate diffs, etc.
			for _, b := range brands {
				// Explain
				explainReq := agent.ExplainRequest{
					Brand:      b,
					CurrentRun: &run,
				}
				agent.Explain(context.Background(), explainReq)

				// Recommend
				recReq := agent.RecommendationRequest{
					Brand: b,
					RunID: run.ID,
				}
				recs, _ := agent.Recommend(context.Background(), recReq)
				for _, rec := range recs {
					resultRepo.InsertRecommendation(&rec)
				}
			}

			status := "done"
			if successCount == 0 && len(results) > 0 {
				status = "failed"
			}
			if err := resultRepo.UpdateRunStatus(run.ID, status, totalCost); err != nil {
				return fmt.Errorf("update run status: %w", err)
			}
			logger.Info("run finished", zap.Uint64("run_id", run.ID), zap.Int("results_saved", successCount))
		}

		duration := time.Since(start)
		printFancySummary(run.ID, prompts, enabledProviders, brands, samples, duration, results)

		if successCount == 0 && len(results) > 0 && exitCode {
			os.Exit(1)
		}

		return nil
	},
}

func printFancySummary(runID uint64, prompts []db.Prompt, providersList []providers.Provider, brands []string, samples int, duration time.Duration, results []db.Result) {
	organicPrompts := 0
	comparisonPrompts := 0
	for _, p := range prompts {
		if p.Category == "comparison" {
			comparisonPrompts++
		} else {
			organicPrompts++
		}
	}

	fmt.Printf("\n── Run #%d complete ──────────────────────────────────────────\n", runID)
	fmt.Printf("  %d organic prompts · %d comparison prompts\n", organicPrompts, comparisonPrompts)
	fmt.Printf("  %d providers · %d brands · %d samples each\n", len(providersList), len(brands), samples)
	fmt.Printf("  Total jobs: %d · Duration: %s\n", len(results), duration.Round(time.Second))

	fmt.Printf("\n── Organic visibility ────────────────────────────────────────\n")
	fmt.Printf("  %-18s", "Brand")
	for _, p := range providersList {
		fmt.Printf("%-10s", p.Name())
	}
	fmt.Printf("%-6s\n", "Score")

	for _, b := range brands {
		fmt.Printf("  %-18s", b)
		var brandResults []db.Result
		for _, r := range results {
			if r.Brand == b && r.Category != "comparison" {
				brandResults = append(brandResults, r)
			}
		}

		// Stability scores for this brand
		// (Simplified calculation for summary display)
		var brandStability []db.StabilityScore
		// ... logic to aggregate ...
		
		vScore := scoring.CalcVisibilityScore(b, int64(runID), brandResults, brandStability)

		for _, p := range providersList {
			var pMentions, pTotal int
			for _, r := range brandResults {
				if r.Provider == p.Name() {
					pTotal++
					if r.BrandMentioned {
						pMentions++
					}
				}
			}
			rate := 0.0
			if pTotal > 0 {
				rate = float64(pMentions) / float64(pTotal) * 100
			}
			fmt.Printf("%-10.0f%%", rate)
		}
		fmt.Printf("%-6.1f\n", vScore.Score)
	}
	fmt.Println("──────────────────────────────────────────────────────────────")
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceVar(&runBrands, "brands", nil, "override brands from config")
	runCmd.Flags().StringSliceVar(&runProviders, "providers", nil, "run only specific providers")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "probe but do not write to DB")
	runCmd.Flags().BoolVar(&verbose, "verbose", false, "print raw responses")
	runCmd.Flags().BoolVar(&exitCode, "exit-code", false, "exit non-zero on failure")
}

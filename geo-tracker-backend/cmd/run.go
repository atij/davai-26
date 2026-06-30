package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adoreme/geo-tracker/internal/adk"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/adoreme/geo-tracker/internal/providers"
	"github.com/adoreme/geo-tracker/internal/scoring"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	runBrands    []string
	runProviders []string
	dryRun       bool
	resumeRun    bool
	verbose      bool
	exitCode     bool
	runID        uint64
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Pipeline execution commands",
}

var runAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Execute full pipeline end-to-end",
	RunE:  runAllHandler,
}

var runIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Fire probe calls and store results (Phase 1)",
	RunE:  runIngestHandler,
}

var runIntelligenceCmd = &cobra.Command{
	Use:   "intelligence",
	Short: "Process signals and calculate scores (Phase 2)",
	RunE:  runIntelligenceHandler,
}

var runInsightCmd = &cobra.Command{
	Use:   "insight",
	Short: "Run Explainer and Recommender agents (Phase 3)",
	RunE:  runInsightHandler,
}

var runListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent pipeline runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		resultRepo := db.NewResultRepo(database)
		runs, err := resultRepo.ListRuns(20)
		if err != nil {
			return err
		}

		fmt.Printf("\n%-4s %-20s %-10s %-8s %-10s %-8s\n", "ID", "Started At", "Status", "Prompts", "Cost", "Duration")
		fmt.Println(strings.Repeat("-", 65))
		for _, r := range runs {
			duration := "n/a"
			if r.DurationSeconds != nil {
				duration = fmt.Sprintf("%ds", *r.DurationSeconds)
			}
			cost := 0.0
			if r.TotalCostUSD != nil {
				cost = *r.TotalCostUSD
			}
			fmt.Printf("%-4d %-20s %-10s %-8d $%-9.2f %-8s\n",
				r.ID,
				r.StartedAt.Format("2006-01-02 15:04:05"),
				r.Status,
				r.PromptCount,
				cost,
				duration)
		}
		fmt.Println()
		return nil
	},
}

func runAllHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipe, run, prompts, resultRepo, err := setupPipeline(ctx, cmd)
	if err != nil {
		return err
	}

	start := time.Now()
	pipeResult, err := pipe.Run(ctx, *run, prompts)
	if err != nil && exitCode {
		os.Exit(1)
	}

	if !dryRun {
		resultRepo.UpdateRunStatus(run.ID, "done", pipeResult.TotalCostUSD)
	}

	duration := time.Since(start)
	printFancySummary(run.ID, prompts, pipe.GetProviders(), runBrands, run.SampleCount, duration, pipeResult.Results)

	return nil
}

func runIngestHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipe, run, prompts, _, err := setupPipeline(ctx, cmd)
	if err != nil {
		return err
	}

	_, err = pipe.Ingest(ctx, *run, prompts)
	return err
}

func runIntelligenceHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipe, run, _, resultRepo, err := setupPipeline(ctx, cmd)
	if err != nil {
		return err
	}

	results, err := resultRepo.GetRunResults(run.ID)
	if err != nil {
		return err
	}

	_, err = pipe.Intelligence(ctx, *run, results)
	return err
}

func runInsightHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipe, run, _, _, err := setupPipeline(ctx, cmd)
	if err != nil {
		return err
	}

	_, err = pipe.Insight(ctx, *run)
	return err
}

func setupPipeline(ctx context.Context, cmd *cobra.Command) (*adk.Pipeline, *db.Run, []db.Prompt, *db.ResultRepo, error) {
	database, err := db.Connect(cfg.Database)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("db connect: %w", err)
	}

	if err := db.Migrate(database); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("db migrate: %w", err)
	}

	promptRepo := db.NewPromptRepo(database)
	resultRepo := db.NewResultRepo(database)

	prompts, err := promptRepo.ListActive()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("list active prompts: %w", err)
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
	runBrands = brands

	samples := cfg.Runner.SamplesPerPrompt
	if samples <= 0 {
		samples = 1
	}

	run := &db.Run{
		PromptCount: len(prompts),
		BrandCount:  len(brands),
		SampleCount: samples,
		Status:      "running",
		StartedAt:   time.Now(),
	}

	if !dryRun {
		if runID > 0 {
			run.ID = runID
			// Load existing run data to prevent overwriting metadata if needed
			var existing db.Run
			err := database.Get(&existing, "SELECT * FROM runs WHERE id = ?", runID)
			if err == nil {
				run.PromptCount = existing.PromptCount
				run.BrandCount = existing.BrandCount
				run.SampleCount = existing.SampleCount
			}
		} else if resumeRun {
			latestID, err := resultRepo.GetLatestRunID()
			if err == nil && latestID > 0 {
				run.ID = latestID
			} else {
				if err := resultRepo.CreateRun(run); err != nil {
					return nil, nil, nil, nil, err
				}
			}
		} else {
			// Only create a NEW run entry if we are running the 'all' command or 'ingest'
			// Intelligence and Insight should generally target existing runs.
			// We'll default to the latest run for them if no runID/resume is provided.
			isPhaseCommand := cmd.Name() == "intelligence" || cmd.Name() == "insight"
			if isPhaseCommand {
				latestID, err := resultRepo.GetLatestRunID()
				if err == nil && latestID > 0 {
					run.ID = latestID
					logger.Info("defaulting to latest run", zap.Uint64("run_id", run.ID))
				} else {
					return nil, nil, nil, nil, fmt.Errorf("no existing run found for phase execution, please provide --run-id")
				}
			} else {
				if err := resultRepo.CreateRun(run); err != nil {
					return nil, nil, nil, nil, err
				}
			}
		}
	}

	explainerAgent, err := adk.NewExplainerAgent(ctx, cfg.ADK)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	recommenderAgent, err := adk.NewRecommenderAgent(ctx, cfg.ADK)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	pipe := adk.NewPipeline(*cfg, resultRepo, enabledProviders, explainerAgent, recommenderAgent, logger)
	return pipe, run, prompts, resultRepo, nil
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
		for _, pr := range providersList {
			for _, p := range prompts {
				var promptSamples []db.Result
				for _, r := range brandResults {
					if r.Provider == pr.Name() && r.PromptID == p.ID {
						promptSamples = append(promptSamples, r)
					}
				}
				if len(promptSamples) > 0 {
					score := scoring.CalcStabilityScore(promptSamples)
					brandStability = append(brandStability, score)
				}
			}
		}

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
	runCmd.AddCommand(runAllCmd)
	runCmd.AddCommand(runIngestCmd)
	runCmd.AddCommand(runIntelligenceCmd)
	runCmd.AddCommand(runInsightCmd)
	runCmd.AddCommand(runListCmd)

	for _, c := range []*cobra.Command{runAllCmd, runIngestCmd, runIntelligenceCmd, runInsightCmd} {
		c.Flags().StringSliceVar(&runBrands, "brands", nil, "override brands from config")
		c.Flags().StringSliceVar(&runProviders, "providers", nil, "run only specific providers")
		c.Flags().BoolVar(&dryRun, "dry-run", false, "probe but do not write to DB")
		c.Flags().BoolVar(&resumeRun, "resume", false, "resume the latest incomplete run")
		c.Flags().Uint64Var(&runID, "run-id", 0, "target a specific run ID")
		c.Flags().BoolVar(&verbose, "verbose", false, "print raw responses")
		c.Flags().BoolVar(&exitCode, "exit-code", false, "exit non-zero on failure")
	}
}

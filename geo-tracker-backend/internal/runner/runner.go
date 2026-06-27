package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/adoreme/geo-tracker/internal/agent"
	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/adoreme/geo-tracker/internal/providers"
	"go.uber.org/zap"
)

type Job struct {
	Prompt      db.Prompt
	Provider    providers.Provider
	Brand       string
	SampleIndex int
}

type Runner struct {
	cfg    config.Config
	logger *zap.Logger
}

func NewRunner(cfg config.Config, logger *zap.Logger) *Runner {
	return &Runner{
		cfg:    cfg,
		logger: logger,
	}
}

func (r *Runner) RunAll(ctx context.Context, prompts []db.Prompt, providersList []providers.Provider, brands []string) []db.Result {
	samples := r.cfg.Runner.SamplesPerPrompt
	if samples <= 0 {
		samples = 1
	}

	totalJobs := len(prompts) * len(providersList) * len(brands) * samples
	jobs := make(chan Job, totalJobs)
	resultsChan := make(chan db.Result, totalJobs)

	// Fill jobs
	for _, p := range prompts {
		for _, pr := range providersList {
			for _, b := range brands {
				for s := 0; s < samples; s++ {
					jobs <- Job{Prompt: p, Provider: pr, Brand: b, SampleIndex: s}
				}
			}
		}
	}
	close(jobs)

	var wg sync.WaitGroup
	workers := r.cfg.Runner.Workers
	if workers <= 0 {
		workers = 1
	}

	rateLimit := r.cfg.Runner.RateLimitPerMinute
	var ticker *time.Ticker
	if rateLimit > 0 {
		ticker = time.NewTicker(time.Minute / time.Duration(rateLimit))
		defer ticker.Stop()
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ticker != nil {
					<-ticker.C
				}
				res := r.executeJob(ctx, job)
				resultsChan <- res
				r.logger.Info("job finished", 
					zap.String("provider", job.Provider.Name()), 
					zap.String("brand", job.Brand), 
					zap.Uint64("prompt_id", job.Prompt.ID),
					zap.Int("sample", job.SampleIndex))
			}
		}()
	}

	wg.Wait()
	close(resultsChan)

	var results []db.Result
	for res := range resultsChan {
		results = append(results, res)
	}

	return results
}

func (r *Runner) executeJob(ctx context.Context, job Job) db.Result {
	result := db.Result{
		PromptID:    job.Prompt.ID,
		Provider:    job.Provider.Name(),
		Brand:       job.Brand,
		SampleIndex: job.SampleIndex,
	}

	// 1. Probe
	start := time.Now()
	probeRes, err := job.Provider.Probe(ctx, job.Prompt.Text)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		r.logger.Error("probe failed",
			zap.String("provider", job.Provider.Name()),
			zap.String("brand", job.Brand),
			zap.Uint64("prompt_id", job.Prompt.ID),
			zap.Int("sample", job.SampleIndex),
			zap.Error(err))
		result.ExtractionError = fmt.Sprintf("probe: %v", err)
		return result
	}
	result.RawResponse = probeRes.RawText
	result.ModelVersion = probeRes.ModelVersion
	result.TokensInput = probeRes.TokensInput
	result.TokensOutput = probeRes.TokensOutput
	result.LatencyMS = latency

	// Calculate cost
	result.CostUSD = r.calculateCost(job.Provider.Name(), probeRes.ModelVersion, probeRes.TokensInput, probeRes.TokensOutput)

	// 2. Extract
	// Fixed to claude-haiku per tasks.md
	extractCfg := r.cfg.Providers.Claude
	extractCfg.ExtractModel = "claude-haiku-4-5-20251001"

	signal, err := agent.Extract(ctx, extractCfg, "claude", probeRes.RawText, job.Brand)
	if err != nil {
		r.logger.Warn("extraction failed",
			zap.String("provider", job.Provider.Name()),
			zap.String("brand", job.Brand),
			zap.Uint64("prompt_id", job.Prompt.ID),
			zap.Error(err))
		result.ExtractionError = fmt.Sprintf("extract (claude): %v", err)
		return result
	}

	result.BrandMentioned = signal.BrandMentioned
	result.Sentiment = signal.Sentiment
	result.MentionCount = signal.MentionCount
	result.RecommendationRank = signal.RecommendationRank
	result.CompetitorsMentioned = signal.CompetitorsMentioned
	result.CitedURLs = signal.CitedURLs

	// Perplexity URLs take priority
	if len(probeRes.CitedURLs) > 0 {
		result.CitedURLs = probeRes.CitedURLs
	}

	return result
}

func (r *Runner) calculateCost(provider, model string, input, output int) float64 {
	rate := r.cfg.CostRates.ClaudeSonnet // Default
	
	// Rough mapping of model versions/names to cost rates
	switch {
	case provider == "claude" && strings.Contains(model, "haiku"):
		rate = r.cfg.CostRates.ClaudeHaiku
	case provider == "claude" && strings.Contains(model, "sonnet"):
		rate = r.cfg.CostRates.ClaudeSonnet
	case provider == "chatgpt" && strings.Contains(model, "mini"):
		rate = r.cfg.CostRates.GPT4oMini
	case provider == "chatgpt":
		rate = r.cfg.CostRates.GPT4o
	case provider == "perplexity":
		rate = r.cfg.CostRates.Perplexity
	case provider == "gemini":
		rate = r.cfg.CostRates.GeminiFlash
	}

	return (float64(input)/1_000_000.0)*rate.Input + (float64(output)/1_000_000.0)*rate.Output
}

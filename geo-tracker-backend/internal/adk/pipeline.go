package adk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adoreme/geo-tracker/internal/agent"
	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/adoreme/geo-tracker/internal/providers"
	"github.com/adoreme/geo-tracker/internal/scoring"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// PipelineResult is the full output of one pipeline execution.
type PipelineResult struct {
	RunID            uint64
	Results          []db.Result
	StabilityScores  []db.StabilityScore
	VisibilityScores map[string]float64
	Explanations     map[string]agent.Explanation
	Recommendations  []db.Recommendation
	Traces           []db.RunTrace
	TotalCostUSD     float64
}

// Pipeline orchestrates the three phases of a Lighthouse run.
type Pipeline struct {
	cfg         config.Config
	repo        *db.ResultRepo
	providers   []providers.Provider
	explainer   *ExplainerAgent
	recommender *RecommenderAgent
	logger      *zap.Logger
}

func NewPipeline(
	cfg config.Config,
	repo *db.ResultRepo,
	providers []providers.Provider,
	explainer *ExplainerAgent,
	recommender *RecommenderAgent,
	logger *zap.Logger,
) *Pipeline {
	return &Pipeline{
		cfg:         cfg,
		repo:        repo,
		providers:   providers,
		explainer:   explainer,
		recommender: recommender,
		logger:      logger,
	}
}

func (p *Pipeline) Run(ctx context.Context, run db.Run, prompts []db.Prompt) (PipelineResult, error) {
	res := PipelineResult{
		RunID:            run.ID,
		VisibilityScores: make(map[string]float64),
		Explanations:     make(map[string]agent.Explanation),
	}

	// ── Phase 1: PROBE ───────────────────────────────────────────────────────
	results, err := p.Ingest(ctx, run, prompts)
	if err != nil {
		return res, err
	}
	res.Results = results

	// ── Phase 2: INTELLIGENCE ────────────────────────────────────────────────
	scores, err := p.Intelligence(ctx, run, results)
	if err != nil {
		return res, err
	}
	res.StabilityScores = scores

	// ── Phase 3: INSIGHT ─────────────────────────────────────────────────────
	insightRes, err := p.Insight(ctx, run)
	if err != nil {
		return res, err
	}
	res.Explanations = insightRes.Explanations
	res.Recommendations = insightRes.Recommendations

	// Sum costs
	for _, r := range results {
		res.TotalCostUSD += r.CostUSD
	}

	return res, err
}

func (p *Pipeline) Ingest(ctx context.Context, run db.Run, prompts []db.Prompt) ([]db.Result, error) {
	p.logger.Info("Phase 1: PROBE (INGEST) starting")
	trace1 := p.traceStart(ctx, run.ID, "probe", "prober_pool")
	results, err := p.phase1Probe(ctx, run, prompts)
	p.traceEnd(ctx, trace1, err)
	return results, err
}

func (p *Pipeline) Intelligence(ctx context.Context, run db.Run, results []db.Result) ([]db.StabilityScore, error) {
	p.logger.Info("Phase 2: INTELLIGENCE starting")
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		trace2a := p.traceStart(ctx, run.ID, "intelligence", "extractor")
		p.logger.Info("Intelligence: Extraction phase (signals already extracted during probe)")
		p.traceEnd(ctx, trace2a, nil)
		return nil
	})

	var allScores []db.StabilityScore
	eg.Go(func() error {
		trace2b := p.traceStart(ctx, run.ID, "intelligence", "stability_scorer")
		p.logger.Info("Intelligence: Calculating stability scores")
		for _, b := range p.cfg.Brands {
			groups := make(map[string][]db.Result)
			for _, r := range results {
				if r.Brand == b.Name {
					key := fmt.Sprintf("%d_%s", r.PromptID, r.Provider)
					groups[key] = append(groups[key], r)
				}
			}
			for _, samples := range groups {
				score := scoring.CalcStabilityScore(samples)
				score.RunID = run.ID
				p.repo.InsertStabilityScore(&score)
				allScores = append(allScores, score)
			}
		}
		p.traceEnd(ctx, trace2b, nil)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Calculate visibility scores
	for _, b := range p.cfg.Brands {
		scoringRes := scoring.CalcVisibilityScore(b.Name, int64(run.ID), results, allScores)
		row := &db.VisibilityScoreRow{
			RunID:            run.ID,
			Brand:            b.Name,
			Score:            scoringRes.Score,
			MentionRate:      scoringRes.MentionRate,
			FirstRecRate:     scoringRes.FirstRecRate,
			SentimentScore:   scoringRes.SentimentScore,
			CitationScore:    scoringRes.CitationScore,
			StabilityScore:   scoringRes.StabilityScore,
			ProviderCoverage: scoringRes.ProviderCoverage,
		}
		p.repo.InsertVisibilityScore(row)
	}

	return allScores, nil
}

type InsightResult struct {
	Explanations    map[string]agent.Explanation
	Recommendations []db.Recommendation
}

func (p *Pipeline) Insight(ctx context.Context, run db.Run) (InsightResult, error) {
	p.logger.Info("Phase 3: INSIGHT starting")
	res := InsightResult{
		Explanations: make(map[string]agent.Explanation),
	}
	eg, ctx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for _, b := range p.cfg.Brands {
		brand := b.Name
		eg.Go(func() error {
			trace3a := p.traceStart(ctx, run.ID, "insight", "explainer_"+brand)
			p.logger.Info("Insight: Running explainer agent", zap.String("brand", brand))
			req := agent.ExplainRequest{Brand: brand, CurrentRun: &run}
			exp, err := p.explainer.Explain(ctx, req)
			if err == nil {
				mu.Lock()
				res.Explanations[brand] = exp
				mu.Unlock()
				p.logger.Info("Insight: Explainer finished", zap.String("brand", brand))
			}
			p.traceEnd(ctx, trace3a, err)
			return nil
		})

		eg.Go(func() error {
			trace3b := p.traceStart(ctx, run.ID, "insight", "recommender_"+brand)
			p.logger.Info("Insight: Running recommender agent", zap.String("brand", brand))
			req := agent.RecommendationRequest{Brand: brand, RunID: run.ID}
			recs, err := p.recommender.Recommend(ctx, req)
			if err == nil {
				for i := range recs {
					recs[i].RunID = run.ID
					recs[i].Brand = brand
					recs[i].Status = "pending"
					p.repo.InsertRecommendation(&recs[i])
				}
				mu.Lock()
				res.Recommendations = append(res.Recommendations, recs...)
				mu.Unlock()
				p.logger.Info("Insight: Recommender finished", zap.String("brand", brand), zap.Int("recs_count", len(recs)))
			}
			p.traceEnd(ctx, trace3b, err)
			return nil
		})
	}

	err := eg.Wait()
	return res, err
}

func (p *Pipeline) phase1Probe(ctx context.Context, run db.Run, prompts []db.Prompt) ([]db.Result, error) {
	samples := p.cfg.Runner.SamplesPerPrompt
	if samples <= 0 {
		samples = 1
	}

	brands := []string{}
	for _, b := range p.cfg.Brands {
		brands = append(brands, b.Name)
	}

	totalJobs := len(prompts) * len(p.providers) * len(brands) * samples
	jobs := make(chan probeJob, totalJobs)
	resultsChan := make(chan db.Result, totalJobs)

	// Nesting changed to fire prompts across all providers/brands simultaneously
	for _, prompt := range prompts {
		for _, pr := range p.providers {
			for _, b := range brands {
				for s := 0; s < samples; s++ {
					jobs <- probeJob{Prompt: prompt, Provider: pr, Brand: b, SampleIndex: s}
				}
			}
		}
	}
	close(jobs)

	var wg sync.WaitGroup
	workers := p.cfg.Runner.Workers
	if workers <= 0 {
		workers = 1
	}

	p.logger.Info("starting probe worker pool", zap.Int("workers", workers), zap.Int("total_jobs", totalJobs))

	var completed int32
	rateLimit := p.cfg.Runner.RateLimitPerMinute
	var ticker *time.Ticker
	if rateLimit > 0 {
		ticker = time.NewTicker(time.Minute / time.Duration(rateLimit))
		defer ticker.Stop()
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				// Resume logic: check if result already exists
				exists, _ := p.repo.ResultExists(run.ID, job.Prompt.ID, job.Provider.Name(), job.Brand, job.SampleIndex)
				if exists {
					// Count it as completed and skip
					curr := atomic.AddInt32(&completed, 1)
					if curr%10 == 0 || curr == int32(totalJobs) {
						p.logger.Info("probe progress (resumed)", zap.Int32("completed", curr), zap.Int("total", totalJobs))
					}
					continue
				}

				if ticker != nil {
					<-ticker.C
				}
				res := p.executeProbeJob(ctx, job)
				res.RunID = run.ID // Ensure RunID is set correctly
				p.repo.InsertResult(&res)
				resultsChan <- res

				curr := atomic.AddInt32(&completed, 1)
				if curr%10 == 0 || curr == int32(totalJobs) {
					p.logger.Info("probe progress",
						zap.Int32("completed", curr),
						zap.Int("total", totalJobs),
						zap.String("last_provider", job.Provider.Name()),
						zap.Uint64("last_prompt_id", job.Prompt.ID),
					)
				}
			}
		}(i)
	}

	wg.Wait()
	close(resultsChan)

	var results []db.Result
	for r := range resultsChan {
		results = append(results, r)
	}
	return results, nil
}

type probeJob struct {
	Prompt      db.Prompt
	Provider    providers.Provider
	Brand       string
	SampleIndex int
}

func (p *Pipeline) executeProbeJob(ctx context.Context, job probeJob) db.Result {
	// Re-using logic from runner.go but adapted for Pipeline
	result := db.Result{
		PromptID:    job.Prompt.ID,
		Provider:    job.Provider.Name(),
		Brand:       job.Brand,
		SampleIndex: job.SampleIndex,
	}

	start := time.Now()
	probeRes, err := job.Provider.Probe(ctx, job.Prompt.Text)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		result.ExtractionError = fmt.Sprintf("probe: %v", err)
		return result
	}
	result.RawResponse = probeRes.RawText
	result.ModelVersion = probeRes.ModelVersion
	result.TokensInput = probeRes.TokensInput
	result.TokensOutput = probeRes.TokensOutput
	result.LatencyMS = latency
	result.CostUSD = p.calculateCost(job.Provider.Name(), probeRes.ModelVersion, probeRes.TokensInput, probeRes.TokensOutput)

	extractCfg := p.cfg.Providers.Claude
	extractCfg.ExtractModel = "claude-haiku-4-5-20251001"

	signal, err := agent.Extract(ctx, extractCfg, "claude", probeRes.RawText, job.Brand)
	if err == nil {
		result.BrandMentioned = signal.BrandMentioned
		result.Sentiment = signal.Sentiment
		result.MentionCount = signal.MentionCount
		result.RecommendationRank = signal.RecommendationRank
		result.CompetitorsMentioned = signal.CompetitorsMentioned
		result.CitedURLs = signal.CitedURLs
	} else {
		result.ExtractionError = fmt.Sprintf("extract: %v", err)
	}

	if len(probeRes.CitedURLs) > 0 {
		result.CitedURLs = probeRes.CitedURLs
	}

	return result
}

func (p *Pipeline) calculateCost(provider, model string, input, output int) float64 {
	rate := p.cfg.CostRates.ClaudeSonnet
	switch {
	case provider == "claude" && strings.Contains(model, "haiku"):
		rate = p.cfg.CostRates.ClaudeHaiku
	case provider == "claude" && strings.Contains(model, "sonnet"):
		rate = p.cfg.CostRates.ClaudeSonnet
	case provider == "chatgpt" && strings.Contains(model, "mini"):
		rate = p.cfg.CostRates.GPT4oMini
	case provider == "chatgpt":
		rate = p.cfg.CostRates.GPT4o
	case provider == "perplexity":
		rate = p.cfg.CostRates.Perplexity
	case provider == "gemini":
		rate = p.cfg.CostRates.GeminiFlash
	}
	return (float64(input)/1_000_000.0)*rate.Input + (float64(output)/1_000_000.0)*rate.Output
}

func (p *Pipeline) traceStart(ctx context.Context, runID uint64, phase, agentName string) *db.RunTrace {
	trace := &db.RunTrace{
		RunID:     runID,
		Phase:     phase,
		AgentName: agentName,
		StartedAt: time.Now().UTC(),
		Status:    "running",
	}
	p.repo.InsertRunTrace(trace)
	return trace
}

func (p *Pipeline) traceEnd(ctx context.Context, trace *db.RunTrace, err error) {
	finishedAt := time.Now().UTC()
	duration := int(finishedAt.Sub(trace.StartedAt).Milliseconds())
	status := "success"
	var errText string
	if err != nil {
		status = "error"
		errText = err.Error()
	}
	p.repo.UpdateRunTrace(trace.ID, finishedAt, duration, status, errText)
}

func (p *Pipeline) GetProviders() []providers.Provider {
	return p.providers
}

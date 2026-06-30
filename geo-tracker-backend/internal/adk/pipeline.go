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
	if len(prompts) == 0 {
		p.logger.Warn("Phase 1: No prompts to ingest, skipping")
		return []db.Result{}, nil
	}
	p.logger.Info("Phase 1: PROBE (INGEST) starting")
	trace1 := p.traceStart(ctx, run.ID, "probe", "prober_pool")
	results, err := p.phase1Probe(ctx, run, prompts)
	p.traceEnd(ctx, trace1, err)
	return results, err
}

func (p *Pipeline) Intelligence(ctx context.Context, run db.Run, results []db.Result) ([]db.StabilityScore, error) {
	if len(results) == 0 {
		p.logger.Warn("Phase 2: No results to analyze, skipping")
		return []db.StabilityScore{}, nil
	}
	p.logger.Info("Phase 2: INTELLIGENCE starting")
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		trace2a := p.traceStart(ctx, run.ID, "intelligence", "extractor")
		p.logger.Info("Intelligence: Verifying/Correcting mentions (heuristic override check)")
		
		for i := range results {
			lowerText := strings.ToLower(results[i].RawResponse)
			lowerBrand := strings.ToLower(results[i].Brand)
			brandAlt := strings.ReplaceAll(lowerBrand, " ", "")
			
			if !results[i].BrandMentioned {
				if strings.Contains(lowerText, lowerBrand) || strings.Contains(lowerText, brandAlt) {
					results[i].BrandMentioned = true
					if results[i].Sentiment == "" || results[i].Sentiment == "not_mentioned" {
						results[i].Sentiment = "neutral"
					}
					if results[i].MentionCount == 0 {
						results[i].MentionCount = 1
					}
					// Update DB
					p.repo.InsertResult(&results[i])
					
					p.logger.Info("heuristic override applied to existing result", 
						zap.Uint64("result_id", results[i].ID),
						zap.String("brand", results[i].Brand))
				}
			}
		}
		
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
	// Check if we have results for this run before starting insights
	existingResults, err := p.repo.GetRunResults(run.ID)
	if err != nil || len(existingResults) == 0 {
		p.logger.Warn("Phase 3: No results found for this run, skipping insights", zap.Uint64("run_id", run.ID))
		return InsightResult{Explanations: make(map[string]agent.Explanation)}, nil
	}

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

	err = eg.Wait()
	return res, err
}

type comparisonJob struct {
	Prompt      db.Prompt
	Provider    providers.Provider
	Brand       string
	SampleIndex int
}

func (p *Pipeline) phase1Probe(ctx context.Context, run db.Run, prompts []db.Prompt) ([]db.Result, error) {
	// Re-fetch the run from DB to ensure it exists and we have the correct ID
	// This fixes intermittent 1452 FK errors when the struct was partially initialized
	var r db.Run
	err := p.repo.GetDB().Get(&r, "SELECT * FROM runs WHERE id = ?", run.ID)
	if err != nil {
		return nil, fmt.Errorf("verify run %d: %w", run.ID, err)
	}
	run = r

	samples := p.cfg.Runner.SamplesPerPrompt
	if samples <= 0 {
		samples = 1
	}

	brands := []string{}
	for _, b := range p.cfg.Brands {
		brands = append(brands, b.Name)
	}

	var oJobs []organicJob
	var cJobs []comparisonJob

	for _, prompt := range prompts {
		for _, pr := range p.providers {
			for s := 0; s < samples; s++ {
				if prompt.Category == "comparison" {
					for _, b := range brands {
						cJobs = append(cJobs, comparisonJob{Prompt: prompt, Provider: pr, Brand: b, SampleIndex: s})
					}
				} else {
					oJobs = append(oJobs, organicJob{Prompt: prompt, Provider: pr, SampleIndex: s})
				}
			}
		}
	}

	totalJobs := len(oJobs) + len(cJobs)
	totalResultRows := (len(oJobs) * len(brands)) + len(cJobs)
	resultsChan := make(chan db.Result, totalResultRows)

	var wg sync.WaitGroup
	workers := p.cfg.Runner.Workers
	if workers <= 0 {
		workers = 1
	}

	p.logger.Info("starting probe worker pool", zap.Int("workers", workers), zap.Int("total_api_calls", totalJobs))

	var completed int32
	rateLimit := p.cfg.Runner.RateLimitPerMinute
	var ticker *time.Ticker
	if rateLimit > 0 {
		ticker = time.NewTicker(time.Minute / time.Duration(rateLimit))
		defer ticker.Stop()
	}

	// Dispatch organic jobs
	oChan := make(chan organicJob, len(oJobs))
	for _, j := range oJobs {
		oChan <- j
	}
	close(oChan)

	// Dispatch comparison jobs
	cChan := make(chan comparisonJob, len(cJobs))
	for _, j := range cJobs {
		cChan <- j
	}
	close(cChan)

	// Process organic jobs
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range oChan {
				// Check if ALL brands for this job already exist
				allExist := true
				for _, b := range brands {
					exists, _ := p.repo.ResultExists(run.ID, job.Prompt.ID, job.Provider.Name(), b, job.SampleIndex)
					if !exists {
						allExist = false
						break
					}
				}

				if allExist {
					atomic.AddInt32(&completed, 1)
					continue
				}

				if ticker != nil {
					<-ticker.C
				}

				p.executeOrganicJob(ctx, job, run, brands, resultsChan)
				atomic.AddInt32(&completed, 1)
			}
		}()
	}

	// Process comparison jobs
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range cChan {
				exists, _ := p.repo.ResultExists(run.ID, job.Prompt.ID, job.Provider.Name(), job.Brand, job.SampleIndex)
				if exists {
					atomic.AddInt32(&completed, 1)
					continue
				}

				if ticker != nil {
					<-ticker.C
				}

				res := p.executeProbeJob(ctx, probeJob{Prompt: job.Prompt, Provider: job.Provider, Brand: job.Brand, SampleIndex: job.SampleIndex})
				res.RunID = run.ID
				if !p.cfg.App.DryRun {
					p.repo.InsertResult(&res)
				}
				resultsChan <- res
				atomic.AddInt32(&completed, 1)
			}
		}()
	}

	go func() {
		for {
			curr := atomic.LoadInt32(&completed)
			p.logger.Info("probe progress", zap.Int32("completed_api_calls", curr), zap.Int("total_api_calls", totalJobs))
			if curr >= int32(totalJobs) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
		}
	}()

	wg.Wait()
	close(resultsChan)

	var results []db.Result
	for r := range resultsChan {
		results = append(results, r)
	}
	return results, nil
}

type organicJob struct {
	Prompt      db.Prompt
	Provider    providers.Provider
	SampleIndex int
}

func (p *Pipeline) executeOrganicJob(ctx context.Context, j organicJob, run db.Run, brands []string, resultsChan chan<- db.Result) {
	start := time.Now()
	probeRes, err := j.Provider.Probe(ctx, j.Prompt.Text)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		for _, brand := range brands {
			res := db.Result{
				RunID:           run.ID,
				PromptID:        j.Prompt.ID,
				Provider:        j.Provider.Name(),
				Brand:           brand,
				SampleIndex:     j.SampleIndex,
				ExtractionError: fmt.Sprintf("probe: %v", err),
			}
			if !p.cfg.App.DryRun {
				p.repo.InsertResult(&res)
			}
			resultsChan <- res
		}
		return
	}

	extractCfg := p.cfg.Providers.Claude
	extractCfg.ExtractModel = "claude-haiku-4-5-20251001"

	multiSignal, err := agent.ExtractMultiBrand(ctx, extractCfg, "claude", probeRes.RawText, brands)
	
	cost := p.calculateCost(j.Provider.Name(), probeRes.ModelVersion, probeRes.TokensInput, probeRes.TokensOutput)

	for _, brand := range brands {
		res := db.Result{
			RunID:        run.ID,
			PromptID:     j.Prompt.ID,
			Provider:     j.Provider.Name(),
			Brand:        brand,
			SampleIndex:  j.SampleIndex,
			RawResponse:  probeRes.RawText,
			ModelVersion: probeRes.ModelVersion,
			TokensInput:  probeRes.TokensInput,
			TokensOutput: probeRes.TokensOutput,
			LatencyMS:    latency,
			CostUSD:      cost,
		}

		if err == nil {
			signal, ok := multiSignal[brand]
			if ok {
				res.BrandMentioned = signal.BrandMentioned
				res.Sentiment = signal.Sentiment
				res.MentionCount = signal.MentionCount
				res.RecommendationRank = signal.RecommendationRank
				res.CompetitorsMentioned = signal.CompetitorsMentioned
				res.CitedURLs = signal.CitedURLs
				
				// Heuristic override
				lowerText := strings.ToLower(probeRes.RawText)
				lowerBrand := strings.ToLower(brand)
				brandAlt := strings.ReplaceAll(lowerBrand, " ", "")
				if !res.BrandMentioned {
					if strings.Contains(lowerText, lowerBrand) || strings.Contains(lowerText, brandAlt) {
						res.BrandMentioned = true
						if res.Sentiment == "" || res.Sentiment == "not_mentioned" {
							res.Sentiment = "neutral"
						}
						if res.MentionCount == 0 {
							res.MentionCount = 1
						}
					}
				}
			}
		} else {
			res.ExtractionError = fmt.Sprintf("multi-extract: %v", err)
		}

		if len(probeRes.CitedURLs) > 0 {
			res.CitedURLs = probeRes.CitedURLs
		}

	if !p.cfg.App.DryRun {
		err := p.repo.InsertResult(&res)
		if err != nil {
			p.logger.Error("failed to insert result", 
				zap.Uint64("run_id", res.RunID),
				zap.Error(err))
		}
	}
		resultsChan <- res
	}
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
		
		// Priority fix: if LLM missed it, but text contains it, override
		lowerText := strings.ToLower(probeRes.RawText)
		lowerBrand := strings.ToLower(job.Brand)
		brandAlt := strings.ReplaceAll(lowerBrand, " ", "")
		
		if !result.BrandMentioned {
			if strings.Contains(lowerText, lowerBrand) || strings.Contains(lowerText, brandAlt) {
				result.BrandMentioned = true
				if result.Sentiment == "" || result.Sentiment == "not_mentioned" {
					result.Sentiment = "neutral"
				}
				if result.MentionCount == 0 {
					result.MentionCount = 1
				}
				p.logger.Warn("llm extractor missed mention - applied heuristic override", 
					zap.String("brand", job.Brand),
					zap.Uint64("prompt_id", job.Prompt.ID))
			}
		}
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

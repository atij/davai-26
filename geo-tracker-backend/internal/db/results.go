package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Result struct {
	ID                   uint64    `db:"id" json:"id"`
	RunID                uint64    `db:"run_id" json:"run_id"`
	PromptID             uint64    `db:"prompt_id" json:"prompt_id"`
	SampleIndex          int       `db:"sample_index" json:"sample_index"`
	Provider             string    `db:"provider" json:"provider"`
	ModelVersion         string    `db:"model_version" json:"model_version"`
	Brand                string    `db:"brand" json:"brand"`
	RawResponse          string    `db:"raw_response" json:"raw_response"`
	BrandMentioned       bool      `db:"brand_mentioned" json:"brand_mentioned"`
	Sentiment            string    `db:"sentiment" json:"sentiment"`
	MentionCount         int       `db:"mention_count" json:"mention_count"`
	RecommendationRank   *int      `db:"recommendation_rank" json:"recommendation_rank"`
	CompetitorsMentioned []string  `db:"competitors_mentioned" json:"competitors_mentioned"`
	CitedURLs            []string  `db:"cited_urls" json:"cited_urls"`
	TokensInput          int       `db:"tokens_input" json:"tokens_input"`
	TokensOutput         int       `db:"tokens_output" json:"tokens_output"`
	LatencyMS            int       `db:"latency_ms" json:"latency_ms"`
	CostUSD              float64   `db:"cost_usd" json:"cost_usd"`
	ExtractionError      string    `db:"extraction_error" json:"extraction_error"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	PromptText           string    `db:"prompt_text" json:"prompt_text"`
	Category             string    `db:"category" json:"category"`
}

// Internal db model with JSON fields for scanning
type resultDB struct {
	Result
	CompetitorsMentionedJSON json.RawMessage `db:"competitors_mentioned"`
	CitedURLsJSON            json.RawMessage `db:"cited_urls"`
}

func (r *ResultRepo) GetRunResults(runID interface{}) ([]Result, error) {
	var rows []resultDB
	query := `
		SELECT r.*, p.text as prompt_text, p.category as category 
		FROM results r
		JOIN prompts p ON r.prompt_id = p.id
		WHERE r.run_id = ?
	`
	err := r.db.Select(&rows, query, runID)
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(rows))
	for i, row := range rows {
		res := row.Result
		if len(row.CompetitorsMentionedJSON) > 0 {
			json.Unmarshal(row.CompetitorsMentionedJSON, &res.CompetitorsMentioned)
		}
		if len(row.CitedURLsJSON) > 0 {
			json.Unmarshal(row.CitedURLsJSON, &res.CitedURLs)
		}
		results[i] = res
	}
	return results, nil
}

type ResultRepo struct {
	db *sqlx.DB
}

func NewResultRepo(db *sqlx.DB) *ResultRepo {
	return &ResultRepo{db: db}
}

func (r *ResultRepo) InsertResult(res *Result) error {
	compJSON, _ := json.Marshal(res.CompetitorsMentioned)
	citedJSON, _ := json.Marshal(res.CitedURLs)

	query := `INSERT INTO results (
		run_id, prompt_id, sample_index, provider, model_version, brand, raw_response, brand_mentioned, 
		sentiment, mention_count, recommendation_rank, competitors_mentioned, 
		cited_urls, tokens_input, tokens_output, latency_ms, cost_usd, extraction_error
	) VALUES (
		:run_id, :prompt_id, :sample_index, :provider, :model_version, :brand, :raw_response, :brand_mentioned,
		:sentiment, :mention_count, :recommendation_rank, :competitors_mentioned,
		:cited_urls, :tokens_input, :tokens_output, :latency_ms, :cost_usd, :extraction_error
	)`

	_, err := r.db.NamedExec(query, map[string]interface{}{
		"run_id":                res.RunID,
		"prompt_id":             res.PromptID,
		"sample_index":          res.SampleIndex,
		"provider":              res.Provider,
		"model_version":         res.ModelVersion,
		"brand":                 res.Brand,
		"raw_response":          res.RawResponse,
		"brand_mentioned":       res.BrandMentioned,
		"sentiment":             res.Sentiment,
		"mention_count":         res.MentionCount,
		"recommendation_rank":   res.RecommendationRank,
		"competitors_mentioned": compJSON,
		"cited_urls":            citedJSON,
		"tokens_input":          res.TokensInput,
		"tokens_output":         res.TokensOutput,
		"latency_ms":            res.LatencyMS,
		"cost_usd":              res.CostUSD,
		"extraction_error":      res.ExtractionError,
	})
	return err
}

type Run struct {
	ID            uint64     `db:"id" json:"id"`
	PromptSetID   *uint64    `db:"prompt_set_id" json:"prompt_set_id"`
	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	FinishedAt    *time.Time `db:"finished_at" json:"finished_at"`
	PromptCount   int        `db:"prompt_count" json:"prompt_count"`
	BrandCount    int        `db:"brand_count" json:"brand_count"`
	SampleCount   int        `db:"sample_count" json:"sample_count"`
	Status        string     `db:"status" json:"status"`
	TotalCostUSD  float64    `db:"total_cost_usd" json:"total_cost_usd"`
}

func (r *ResultRepo) CreateRun(run *Run) error {
	query := `INSERT INTO runs (prompt_set_id, prompt_count, brand_count, sample_count, status) 
	          VALUES (:prompt_set_id, :prompt_count, :brand_count, :sample_count, :status)`
	res, err := r.db.NamedExec(query, run)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		run.ID = uint64(id)
	}
	return err
}

func (r *ResultRepo) UpdateRunStatus(id uint64, status string, totalCost float64) error {
	_, err := r.db.Exec("UPDATE runs SET status = ?, finished_at = NOW(), total_cost_usd = ? WHERE id = ?", status, totalCost, id)
	return err
}

func (r *ResultRepo) GetLatestRunID() (uint64, error) {
	var id uint64
	err := r.db.Get(&id, "SELECT id FROM runs WHERE status = 'done' ORDER BY started_at DESC LIMIT 1")
	if err != nil {
		// Fallback to latest run regardless of status
		err = r.db.Get(&id, "SELECT id FROM runs ORDER BY started_at DESC LIMIT 1")
	}
	return id, err
}

type RunSummary struct {
	Provider      string  `db:"provider"`
	Brand         string  `db:"brand"`
	MentionRate   float64 `db:"mention_rate"`
	SentimentMode string  `db:"sentiment_mode"`
}

func (r *ResultRepo) GetRunSummary(runID uint64, brand, provider string) ([]RunSummary, error) {
	query := `
		SELECT 
			provider, 
			brand, 
			AVG(brand_mentioned) * 100 as mention_rate
		FROM results 
		WHERE run_id = ?
	`
	args := []interface{}{runID}
	if brand != "" {
		query += " AND brand = ?"
		args = append(args, brand)
	}
	if provider != "" {
		query += " AND provider = ?"
		args = append(args, provider)
	}
	query += " GROUP BY provider, brand"

	var summary []RunSummary
	err := r.db.Select(&summary, query, args...)
	return summary, err
}

func (r *ResultRepo) GetPromptResults(promptID uint64) ([]Result, error) {
	var rows []resultDB
	err := r.db.Select(&rows, "SELECT * FROM results WHERE prompt_id = ? ORDER BY created_at DESC", promptID)
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(rows))
	for i, row := range rows {
		res := row.Result
		if len(row.CompetitorsMentionedJSON) > 0 {
			json.Unmarshal(row.CompetitorsMentionedJSON, &res.CompetitorsMentioned)
		}
		if len(row.CitedURLsJSON) > 0 {
			json.Unmarshal(row.CitedURLsJSON, &res.CitedURLs)
		}
		results[i] = res
	}
	return results, nil
}

type BrandSummary struct {
	Brand              string  `db:"brand" json:"brand"`
	MentionRate        float64 `db:"mention_rate" json:"mention_rate"`
	AvgRecommendation  float64 `db:"avg_recommendation" json:"avg_recommendation"`
	SentimentPositive  float64 `db:"sentiment_positive" json:"sentiment_positive"`
	SentimentNeutral   float64 `db:"sentiment_neutral" json:"sentiment_neutral"`
	SentimentNegative  float64 `db:"sentiment_negative" json:"sentiment_negative"`
	// Additional fields for frontend BrandSummary
	SentimentScore     float64            `json:"sentiment_score"`
	SentimentBreakdown SentimentBreakdown `json:"sentiment_breakdown"`
	ProviderRates      map[string]float64 `json:"provider_rates"`
	TopProvider        string             `json:"top_provider"`
	WeakestProvider    string             `json:"weakest_provider"`
	RunID              uint64             `json:"run_id"`
	RunAt              time.Time          `json:"run_at"`
}

type SentimentBreakdown struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

func (r *ResultRepo) GetBrandSummary(brand string) (*BrandSummary, error) {
	// Get summary from the latest successful run
	var latestRun Run
	err := r.db.Get(&latestRun, "SELECT id, started_at FROM runs WHERE status = 'done' ORDER BY started_at DESC LIMIT 1")
	if err != nil {
		// Fallback: if no successful run, check for ANY latest run (e.g. still running or failed but has partial results)
		err = r.db.Get(&latestRun, "SELECT id, started_at FROM runs ORDER BY started_at DESC LIMIT 1")
		if err != nil {
			return nil, fmt.Errorf("no runs found: %w", err)
		}
	}

	query := `
		SELECT 
			brand,
			COALESCE(AVG(brand_mentioned) * 100, 0) as mention_rate,
			COALESCE(AVG(recommendation_rank), 0) as avg_recommendation,
			COALESCE(SUM(CASE WHEN sentiment = 'positive' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0) * 100, 0) as sentiment_positive,
			COALESCE(SUM(CASE WHEN sentiment = 'neutral' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0) * 100, 0) as sentiment_neutral,
			COALESCE(SUM(CASE WHEN sentiment = 'negative' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0) * 100, 0) as sentiment_negative
		FROM results 
		WHERE run_id = ? AND brand = ?
		GROUP BY brand
	`
	var summary BrandSummary
	err = r.db.Get(&summary, query, latestRun.ID, brand)
	if err != nil {
		// If no results for this specific brand/run yet, return a skeleton summary instead of error
		return &BrandSummary{
			Brand: brand,
			RunID: latestRun.ID,
			RunAt: latestRun.StartedAt,
		}, nil
	}

	summary.RunID = latestRun.ID
	summary.RunAt = latestRun.StartedAt
	summary.SentimentBreakdown = SentimentBreakdown{
		Positive: summary.SentimentPositive,
		Neutral:  summary.SentimentNeutral,
		Negative: summary.SentimentNegative,
	}
	// Simplified score: positive - negative
	summary.SentimentScore = (summary.SentimentPositive - summary.SentimentNegative) / 100

	// Provider rates
	var pRates []struct {
		Provider    string  `db:"provider"`
		MentionRate float64 `db:"mention_rate"`
	}
	err = r.db.Select(&pRates, "SELECT provider, AVG(brand_mentioned) * 100 as mention_rate FROM results WHERE run_id = ? AND brand = ? GROUP BY provider", latestRun.ID, brand)
	if err == nil {
		summary.ProviderRates = make(map[string]float64)
		var topP, weakP string
		var maxR float64 = -1
		var minR float64 = 101

		for _, pr := range pRates {
			summary.ProviderRates[pr.Provider] = pr.MentionRate
			if pr.MentionRate > maxR {
				maxR = pr.MentionRate
				topP = pr.Provider
			}
			if pr.MentionRate < minR {
				minR = pr.MentionRate
				weakP = pr.Provider
			}
		}
		summary.TopProvider = topP
		summary.WeakestProvider = weakP
	}

	return &summary, nil
}

type TrendPoint struct {
	RunID       uint64    `db:"run_id" json:"run_id"`
	StartedAt   time.Time `db:"started_at" json:"run_at"` // Match frontend TrendPoint.run_at
	MentionRate float64   `db:"mention_rate" json:"mention_rate"`
}

func (r *ResultRepo) GetBrandTrend(brand string, limit int) ([]TrendPoint, error) {
	query := `
		SELECT 
			runs.id as run_id, 
			runs.started_at, 
			AVG(results.brand_mentioned) * 100 as mention_rate
		FROM results
		JOIN runs ON results.run_id = runs.id
		WHERE results.brand = ?
		GROUP BY runs.id, runs.started_at
		ORDER BY runs.started_at DESC
		LIMIT ?
	`
	var trend []TrendPoint
	err := r.db.Select(&trend, query, brand, limit)
	return trend, err
}

type CompetitorCount struct {
	Name      string `db:"competitor_name" json:"name"`
	Frequency int    `db:"frequency" json:"frequency"`
}

func (r *ResultRepo) GetTopCompetitors(brand string, limit int) ([]CompetitorCount, error) {
	// Total prompts in the latest run for percentage calculation
	var totalPrompts int
	err := r.db.Get(&totalPrompts, "SELECT prompt_count FROM runs WHERE status = 'done' ORDER BY started_at DESC LIMIT 1")
	if err != nil || totalPrompts == 0 {
		totalPrompts = 1 // Prevent division by zero
	}

	query := `
		SELECT 
			jt.competitor_name,
			ROUND(COUNT(*) / ? * 100) as frequency
		FROM results,
		JSON_TABLE(competitors_mentioned, '$[*]' COLUMNS (competitor_name VARCHAR(128) PATH '$')) AS jt
		WHERE brand = ? AND competitor_name != ?
		GROUP BY jt.competitor_name
		ORDER BY frequency DESC
		LIMIT ?
	`
	var competitors []CompetitorCount
	err = r.db.Select(&competitors, query, totalPrompts, brand, brand, limit)
	return competitors, err
}

type CitationGapEntry struct {
	CitedURL      string `db:"cited_url" json:"cited_url"`
	Domain        string `db:"domain" json:"domain"`
	CitationCount int    `db:"citation_count" json:"citation_count"`
	Category      string `db:"category" json:"category"`
}

func (r *ResultRepo) GetCitationGap(brand string, runID uint64) ([]CitationGapEntry, error) {
	query := `
		SELECT
			ANY_VALUE(JSON_UNQUOTE(url_item.value))           AS cited_url,
			SUBSTRING_INDEX(
				REPLACE(REPLACE(JSON_UNQUOTE(url_item.value),
					'https://',''),'http://',''), '/', 1)    AS domain,
			COUNT(*)                                          AS citation_count,
			MIN(p.category)                                   AS category
		FROM results r
		JOIN prompts p ON r.prompt_id = p.id,
		JSON_TABLE(r.cited_urls, '$[*]' COLUMNS (value JSON PATH '$')) url_item
		WHERE r.run_id          = ?
		  AND r.brand           = ?
		  AND r.brand_mentioned = FALSE
		  AND p.category        != 'comparison'
		  AND r.cited_urls      IS NOT NULL
		GROUP BY domain
		ORDER BY citation_count DESC
		LIMIT 20
	`
	var gaps []CitationGapEntry
	err := r.db.Select(&gaps, query, runID, brand)
	return gaps, err
}

type StabilityScore struct {
	ID             uint64  `db:"id" json:"id"`
	RunID          uint64  `db:"run_id" json:"run_id"`
	PromptID       uint64  `db:"prompt_id" json:"prompt_id"`
	Provider       string  `db:"provider" json:"provider"`
	Brand          string  `db:"brand" json:"brand"`
	SampleCount    int     `db:"sample_count" json:"sample_count"`
	MentionRate    float64 `db:"mention_rate" json:"mention_rate"`
	RankVariance   float64 `db:"rank_variance" json:"rank_variance"`
	StabilityScore float64 `db:"stability_score" json:"stability_score"`
}

func (r *ResultRepo) InsertStabilityScore(s *StabilityScore) error {
	query := `INSERT INTO stability_scores (
		run_id, prompt_id, provider, brand, sample_count, mention_rate, rank_variance, stability_score
	) VALUES (
		:run_id, :prompt_id, :provider, :brand, :sample_count, :mention_rate, :rank_variance, :stability_score
	)`
	_, err := r.db.NamedExec(query, s)
	return err
}

func (r *ResultRepo) GetStabilityScores(runID uint64, brand string) ([]StabilityScore, error) {
	var scores []StabilityScore
	err := r.db.Select(&scores, "SELECT * FROM stability_scores WHERE run_id = ? AND brand = ?", runID, brand)
	return scores, err
}

type Recommendation struct {
	ID             uint64     `db:"id" json:"id"`
	RunID          uint64     `db:"run_id" json:"run_id"`
	Brand          string     `db:"brand" json:"brand"`
	Category       string     `db:"category" json:"category"`
	Action         string     `db:"action" json:"action"`
	ExpectedImpact string     `db:"expected_impact" json:"expected_impact"`
	Rationale      string     `db:"rationale" json:"rationale"`
	Status         string     `db:"status" json:"status"`
	ImplementedAt  *time.Time `db:"implemented_at" json:"implemented_at"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

func (r *ResultRepo) InsertRecommendation(rec *Recommendation) error {
	query := `INSERT INTO recommendations (
		run_id, brand, category, action, expected_impact, rationale, status
	) VALUES (
		:run_id, :brand, :category, :action, :expected_impact, :rationale, :status
	)`
	_, err := r.db.NamedExec(query, rec)
	return err
}

func (r *ResultRepo) GetRecommendations(brand string, status string) ([]Recommendation, error) {
	var recs []Recommendation
	query := "SELECT * FROM recommendations WHERE brand = ?"
	args := []interface{}{brand}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&recs, query, args...)
	return recs, err
}

func (r *ResultRepo) MarkRecommendationImplemented(id uint64) error {
	_, err := r.db.Exec("UPDATE recommendations SET status = 'implemented', implemented_at = NOW() WHERE id = ?", id)
	return err
}

func (r *ResultRepo) GetBrands() ([]string, error) {

	var brands []string
	err := r.db.Select(&brands, "SELECT DISTINCT brand FROM results ORDER BY brand ASC")
	return brands, err
}


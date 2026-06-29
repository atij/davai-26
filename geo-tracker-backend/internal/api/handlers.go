package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Handlers struct {
	db *sqlx.DB
}

func NewHandlers(database *sqlx.DB) *Handlers {
	return &Handlers{db: database}
}

func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	dbStatus := "ok"
	if err != nil {
		dbStatus = "error"
	}
	sendJSON(w, http.StatusOK, HealthResponse{Status: "ok", DB: dbStatus})
}

func (h *Handlers) GetPrompts(w http.ResponseWriter, r *http.Request) {
	repo := db.NewPromptRepo(h.db)
	prompts, err := repo.ListActive()
	if err != nil {
		fmt.Printf("GetPrompts Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, prompts)
}

func (h *Handlers) GetRuns(w http.ResponseWriter, r *http.Request) {
	type runResponse struct {
		ID              uint64     `db:"id" json:"id"`
		StartedAt       time.Time  `db:"started_at" json:"started_at"`
		FinishedAt      *time.Time `db:"finished_at" json:"finished_at"`
		DurationSeconds *int       `db:"duration_seconds" json:"duration_seconds"`
		PromptCount     int        `db:"prompt_count" json:"prompt_count"`
		BrandCount      int        `db:"brand_count" json:"brand_count"`
		SampleCount     int        `db:"sample_count" json:"sample_count"`
		Status          string     `db:"status" json:"status"`
		TotalCostUSD    *float64   `db:"total_cost_usd" json:"total_cost_usd"`
	}
	var runs []runResponse
	err := h.db.Select(&runs, `
		SELECT 
			id, started_at, finished_at, duration_seconds, prompt_count, brand_count, sample_count, status, total_cost_usd
		FROM runs 
		ORDER BY started_at DESC`)
	if err != nil {
		fmt.Printf("GetRuns Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, runs)
}

func (h *Handlers) GetRunResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	repo := db.NewResultRepo(h.db)
	results, err := repo.GetRunResults(id)
	if err != nil {
		fmt.Printf("GetRunResults Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, results)
}

func (h *Handlers) GetBrands(w http.ResponseWriter, r *http.Request) {
	repo := db.NewResultRepo(h.db)
	brands, err := repo.GetBrands()
	if err != nil {
		fmt.Printf("GetBrands Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, brands)
}

func (h *Handlers) GetBrandSummary(w http.ResponseWriter, r *http.Request) {
	brandRaw := chi.URLParam(r, "brand")
	brand := resolveBrand(brandRaw)

	repo := db.NewResultRepo(h.db)
	summary, err := repo.GetBrandSummary(brand)
	if err != nil {
		fmt.Printf("GetBrandSummary Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}

	// Fetch stored visibility score (computed by run command, not recomputed here)
	vscore, err := repo.GetLatestVisibilityScore(brand)
	if err == nil && vscore != nil {
		summary.VisibilityScore = vscore.Score
		summary.FirstRecRate = vscore.FirstRecRate
		summary.CitationScore = vscore.CitationScore
		summary.StabilityScore = vscore.StabilityScore
		summary.ProviderCoverage = vscore.ProviderCoverage
	}

	summary.PromptType = "organic"
	sendJSON(w, http.StatusOK, summary)
}

func (h *Handlers) GetExplain(w http.ResponseWriter, r *http.Request) {
	runIDStr := chi.URLParam(r, "run_id")
	runID, _ := strconv.ParseUint(runIDStr, 10, 64)
	brand := resolveBrand(r.URL.Query().Get("brand"))

	repo := db.NewResultRepo(h.db)
	explanation, err := repo.GetExplanation(runID, brand)
	if err != nil {
		sendError(w, http.StatusNotFound, "no explanation found", "NOT_FOUND")
		return
	}
	sendJSON(w, http.StatusOK, explanation)
}

func (h *Handlers) GetBrandTrend(w http.ResponseWriter, r *http.Request) {
	brandRaw := chi.URLParam(r, "brand")
	brand := resolveBrand(brandRaw)

	runsParam := r.URL.Query().Get("runs")
	limit := 10
	if runsParam != "" {
		if l, err := strconv.Atoi(runsParam); err == nil {
			limit = l
		}
	}

	repo := db.NewResultRepo(h.db)
	trend, err := repo.GetBrandTrend(brand, limit)
	if err != nil {
		fmt.Printf("GetBrandTrend Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, trend)
}

func (h *Handlers) GetBrandStability(w http.ResponseWriter, r *http.Request) {
	brandRaw := chi.URLParam(r, "brand")
	brand := resolveBrand(brandRaw)
	repo := db.NewResultRepo(h.db)
	runID, _ := repo.GetLatestRunID()
	
	scores, err := repo.GetStabilityScores(runID, brand)
	if err != nil {
		fmt.Printf("GetBrandStability Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, scores)
}

func (h *Handlers) GetCitationGap(w http.ResponseWriter, r *http.Request) {
	brandRaw := chi.URLParam(r, "brand")
	brand := resolveBrand(brandRaw)
	repo := db.NewResultRepo(h.db)
	runID, _ := repo.GetLatestRunID()

	gaps, err := repo.GetCitationGap(brand, runID)
	if err != nil {
		fmt.Printf("GetCitationGap Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"brand":  brand,
		"run_id": runID,
		"gaps":   gaps,
	})
}

func (h *Handlers) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	brandRaw := r.URL.Query().Get("brand")
	brand := resolveBrand(brandRaw)
	status := r.URL.Query().Get("status")
	repo := db.NewResultRepo(h.db)
	
	recs, err := repo.GetRecommendations(brand, status)
	if err != nil {
		fmt.Printf("GetRecommendations Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, recs)
}

func (h *Handlers) PostImplementRecommendation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	repo := db.NewResultRepo(h.db)
	
	err := repo.MarkRecommendationImplemented(id)
	if err != nil {
		fmt.Printf("PostImplementRecommendation Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) GetPromptResults(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid prompt id", "BAD_REQUEST")
		return
	}
	repo := db.NewResultRepo(h.db)
	results, err := repo.GetPromptResults(id)
	if err != nil {
		fmt.Printf("GetPromptResults Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, results)
}

func (h *Handlers) GetCompare(w http.ResponseWriter, r *http.Request) {
	brandsParam := r.URL.Query().Get("brands")
	if brandsParam == "" {
		sendError(w, http.StatusBadRequest, "brands parameter is required", "BAD_REQUEST")
		return
	}
	brandsRaw := strings.Split(brandsParam, ",")
	repo := db.NewResultRepo(h.db)

	type compareResponse struct {
		Brands map[string]*db.BrandSummary `json:"brands"`
		Trend  map[string][]db.TrendPoint  `json:"trend"`
	}

	res := compareResponse{
		Brands: make(map[string]*db.BrandSummary),
		Trend:  make(map[string][]db.TrendPoint),
	}

	for _, bRaw := range brandsRaw {
		brand := resolveBrand(bRaw)

		summary, err := repo.GetBrandSummary(brand)
		if err == nil {
			res.Brands[bRaw] = summary
		} else {
			fmt.Printf("GetCompare (Summary) Error for %s: %v\n", brand, err)
		}
		trend, err := repo.GetBrandTrend(brand, 10)
		if err == nil {
			res.Trend[bRaw] = trend
		} else {
			fmt.Printf("GetCompare (Trend) Error for %s: %v\n", brand, err)
		}
	}
	sendJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetCompetitors(w http.ResponseWriter, r *http.Request) {
	brandRaw := r.URL.Query().Get("brand")
	brand := resolveBrand(brandRaw)

	repo := db.NewResultRepo(h.db)
	competitors, err := repo.GetTopCompetitors(brand, 10)
	if err != nil {
		fmt.Printf("GetCompetitors Error: %v\n", err)
		sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}
	sendJSON(w, http.StatusOK, competitors)
}

func resolveBrand(brandRaw string) string {
	lower := strings.ToLower(brandRaw)
	if lower == "adoreme" || lower == "adore me" {
		return "Adore Me"
	}
	if lower == "victorias secret" || lower == "victoria's secret" || lower == "victoria secret" || lower == "vs" || strings.Contains(lower, "victoria") {
		return "Victoria's Secret"
	}
	return brandRaw
}

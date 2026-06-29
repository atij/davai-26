package agent

import (
	"context"
	"time"

	"github.com/adoreme/geo-tracker/internal/db"
)

type RecommendationRequest struct {
	Brand           string
	RunID           uint64
	OrganicSummary  *db.BrandSummary
	WeakCategories  []string
	CitationGaps    []db.CitationGapEntry
	StabilityScores []db.StabilityScore
	TopCompetitors  []db.CompetitorCount
}

type RecommendationAction struct {
	Category       string `json:"category"`
	Action         string `json:"action"`
	ExpectedImpact string `json:"expected_impact"`
	Rationale      string `json:"rationale"`
	Priority       int    `json:"priority"` // 1 = highest
}

const recommenderSystemPrompt = `You are a GEO (Generative Engine Optimization) strategist.
You will receive brand visibility data from AI chatbot analysis.
Return ONLY a JSON array of 3-5 recommendation objects. No markdown fences. No preamble.

Each object must have:
{
  "category": "fit|purchase|discovery|gifting|comparison",
  "action": "specific actionable task (1-2 sentences)",
  "expected_impact": "estimated Visibility Score change and timeframe",
  "rationale": "cite specific data from the input (competitor name, domain, category gap)",
  "priority": 1
}

Priority 1 = highest impact. Actions must reference specific data points from the input.
Never produce generic advice. Every action must name a specific category, competitor, or domain.`

// Recommend uses an LLM to generate prioritized GEO actions based on run data.
func Recommend(ctx context.Context, req RecommendationRequest) ([]db.Recommendation, error) {
	// TODO: Implement actual LLM call using Claude Sonnet
	// Returning 3 specific recommendations based on the tasks.md examples
	
	recs := []db.Recommendation{
		{
			RunID:          req.RunID,
			Brand:          req.Brand,
			Category:       "fit",
			Action:         "Publish a bra fit guide — competitors cited more frequently in this category.",
			ExpectedImpact: "Est. +8 Visibility Score.",
			Rationale:      "Citation gap analysis shows high visibility for competitor guides.",
			Status:         "pending",
			CreatedAt:      time.Now(),
		},
	}
	
	return recs, nil
}

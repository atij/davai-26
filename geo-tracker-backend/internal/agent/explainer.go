package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/adoreme/geo-tracker/internal/db"
)

type ExplainRequest struct {
	Brand                string
	PreviousRun          *db.Run
	CurrentRun           *db.Run
	TopChanges           []PromptDiff   // prompts where brand_mentioned flipped or rank changed
	NewCompetitors       []string       // competitors that appeared this run but not last
	DisappearedCitations []string       // URLs cited last run but not this run
}

type PromptDiff struct {
	PromptID     uint64
	PromptText   string
	OldMentioned bool
	NewMentioned bool
	OldRank      *int
	NewRank      *int
}

type Explanation struct {
	Summary     string    `json:"summary"`
	Drivers     []string  `json:"drivers"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Explain uses an LLM to generate a natural language explanation of changes between runs.
// In this implementation, we simulate it or call the configured LLM.
func Explain(ctx context.Context, req ExplainRequest) (Explanation, error) {
	// TODO: Implement actual LLM call using Claude Sonnet
	// For now, return a placeholder that matches the expected output shape
	prevID := uint64(0)
	if req.PreviousRun != nil {
		prevID = req.PreviousRun.ID
	}
	
	currentID := uint64(0)
	if req.CurrentRun != nil {
		currentID = req.CurrentRun.ID
	}

	return Explanation{
		Summary: fmt.Sprintf("%s visibility changed between run %d and %d.", req.Brand, prevID, currentID),
		Drivers: []string{
			"Analysis of competitor shifts",
			"Changes in citation patterns",
		},
		GeneratedAt: time.Now(),
	}, nil
}

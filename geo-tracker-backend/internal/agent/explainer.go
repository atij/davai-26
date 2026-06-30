package agent

import (
	"context"
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

// This is a type alias to avoid circular import if needed, but since adk package
// will import internal/agent, we can't have internal/agent import adk.
// Instead, we pass the agent as an interface or a concrete type if we don't mind the dependency.
// Given the tasks.md, we should use a package alias.

// Explain is now a thin delegate.
func Explain(ctx context.Context, req ExplainRequest, a explainerInterface) (Explanation, error) {
	return a.Explain(ctx, req)
}

type explainerInterface interface {
	Explain(context.Context, ExplainRequest) (Explanation, error)
}

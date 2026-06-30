package adk

import (
	"github.com/adoreme/geo-tracker/internal/db"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// ToolSet holds tool functions bound to a DB repo instance.
type ToolSet struct {
	repo *db.ResultRepo
}

func NewToolSet(repo *db.ResultRepo) *ToolSet {
	return &ToolSet{repo: repo}
}

// Tools returns the slice of ADK tools to pass to the Strategy Agent.
func (t *ToolSet) Tools() []tool.Tool {
	tools := []tool.Tool{}

	// get_visibility_trend
	tool1, _ := functiontool.New(functiontool.Config{
		Name:        "get_visibility_trend",
		Description: "Returns the last N visibility scores for a brand across runs.",
	}, t.getVisibilityTrend)
	tools = append(tools, tool1)

	// get_citation_gaps
	tool2, _ := functiontool.New(functiontool.Config{
		Name:        "get_citation_gaps",
		Description: "Returns domains that cite competitors but not the brand.",
	}, t.getCitationGaps)
	tools = append(tools, tool2)

	// get_stability_scores
	tool3, _ := functiontool.New(functiontool.Config{
		Name:        "get_stability_scores",
		Description: "Returns stability scores (variance across samples) for a brand and run.",
	}, t.getStabilityScores)
	tools = append(tools, tool3)

	// get_competitor_share
	tool4, _ := functiontool.New(functiontool.Config{
		Name:        "get_competitor_share",
		Description: "Returns the top competitors by mention count for a run.",
	}, t.getCompetitorShare)
	tools = append(tools, tool4)

	// search_recommendations
	tool5, _ := functiontool.New(functiontool.Config{
		Name:        "search_recommendations",
		Description: "Returns recommendations for a brand filtered by status ('pending', 'implemented', or '').",
	}, t.searchRecommendations)
	tools = append(tools, tool5)

	// mark_recommendation_done
	tool6, _ := functiontool.New(functiontool.Config{
		Name:        "mark_recommendation_done",
		Description: "Marks a specific recommendation as implemented.",
	}, t.markRecommendationDone)
	tools = append(tools, tool6)

	return tools
}

type VisibilityTrendArgs struct {
	Brand string `json:"brand" description:"Brand name, e.g. 'Adore Me'"`
	Limit int    `json:"limit" description:"Number of past runs to return, default 10"`
}
type VisibilityTrendResult struct {
	Points []db.TrendPoint `json:"points"`
}
func (t *ToolSet) getVisibilityTrend(ctx tool.Context, args VisibilityTrendArgs) (VisibilityTrendResult, error) {
	if args.Limit <= 0 {
		args.Limit = 10
	}
	points, err := t.repo.GetVisibilityTrend(args.Brand, args.Limit)
	return VisibilityTrendResult{Points: points}, err
}

type CitationGapArgs struct {
	Brand string `json:"brand"`
	RunID uint64 `json:"run_id" description:"Run ID to analyse. Use 0 for latest run."`
}
type CitationGapResult struct {
	Gaps []db.CitationGapEntry `json:"gaps"`
}
func (t *ToolSet) getCitationGaps(ctx tool.Context, args CitationGapArgs) (CitationGapResult, error) {
	if args.RunID == 0 {
		var err error
		args.RunID, err = t.repo.GetLatestRunID()
		if err != nil {
			return CitationGapResult{}, err
		}
	}
	gaps, err := t.repo.GetCitationGap(args.Brand, args.RunID)
	return CitationGapResult{Gaps: gaps}, err
}

type StabilityArgs struct {
	Brand string `json:"brand"`
	RunID uint64 `json:"run_id"`
}
type StabilityResult struct {
	Scores []db.StabilityScore `json:"scores"`
}
func (t *ToolSet) getStabilityScores(ctx tool.Context, args StabilityArgs) (StabilityResult, error) {
	if args.RunID == 0 {
		var err error
		args.RunID, err = t.repo.GetLatestRunID()
		if err != nil {
			return StabilityResult{}, err
		}
	}
	scores, err := t.repo.GetStabilityScores(args.RunID, args.Brand)
	return StabilityResult{Scores: scores}, err
}

type CompetitorShareArgs struct {
	Brand string `json:"brand"`
	RunID uint64 `json:"run_id"`
}
type CompetitorShareResult struct {
	Competitors []db.CompetitorCount `json:"competitors"`
}
func (t *ToolSet) getCompetitorShare(ctx tool.Context, args CompetitorShareArgs) (CompetitorShareResult, error) {
	if args.RunID == 0 {
		var err error
		args.RunID, err = t.repo.GetLatestRunID()
		if err != nil {
			return CompetitorShareResult{}, err
		}
	}
	competitors, err := t.repo.GetCompetitorShare(args.Brand, args.RunID)
	return CompetitorShareResult{Competitors: competitors}, err
}

type SearchRecsArgs struct {
	Brand  string `json:"brand"`
	Status string `json:"status" description:"'pending', 'implemented', or '' for all"`
}
type SearchRecsResult struct {
	Recommendations []db.Recommendation `json:"recommendations"`
}
func (t *ToolSet) searchRecommendations(ctx tool.Context, args SearchRecsArgs) (SearchRecsResult, error) {
	recs, err := t.repo.SearchRecommendations(args.Brand, args.Status)
	return SearchRecsResult{Recommendations: recs}, err
}

type MarkDoneArgs struct {
	RecommendationID uint64 `json:"recommendation_id"`
}
type MarkDoneResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
func (t *ToolSet) markRecommendationDone(ctx tool.Context, args MarkDoneArgs) (MarkDoneResult, error) {
	err := t.repo.MarkRecommendationImplemented(args.RecommendationID)
	if err != nil {
		return MarkDoneResult{Success: false, Message: err.Error()}, nil
	}
	return MarkDoneResult{Success: true, Message: "Recommendation marked as implemented"}, nil
}

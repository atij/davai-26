package adk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/adoreme/geo-tracker/internal/agent"
	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/adoreme/geo-tracker/internal/db"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	adkrunner "google.golang.org/adk/runner"
	adksession "google.golang.org/adk/session"
	"google.golang.org/genai"
)

// ─── Explainer Agent ────────────────────────────────────────────────────────

// ExplainerAgent generates a plain-English diff explanation between two runs.
type ExplainerAgent struct {
	runner *adkrunner.Runner
}

func NewExplainerAgent(ctx context.Context, cfg config.ADKConfig) (*ExplainerAgent, error) {
	model, err := NewADKModel(ctx, cfg, cfg.ExplainerModel)
	if err != nil {
		return nil, fmt.Errorf("explainer model: %w", err)
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "lighthouse_explainer",
		Model:       model,
		Instruction: explainerSystemPrompt,
	})
	if err != nil {
		return nil, err
	}
	r, err := adkrunner.New(adkrunner.Config{
		AppName:        "lighthouse",
		Agent:          a,
		SessionService: adksession.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return &ExplainerAgent{runner: r}, nil
}

func (e *ExplainerAgent) Explain(ctx context.Context, req agent.ExplainRequest) (agent.Explanation, error) {
	prompt := fmt.Sprintf("Explain the visibility changes for brand %s.\nData: %v", req.Brand, req)
	
	iter := e.runner.Run(ctx, "system", "explain-run", &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}, adkagent.RunConfig{})

	var lastText string
	for ev, err := range iter {
		if err != nil {
			return agent.Explanation{}, err
		}
		if ev.Content != nil {
			for _, p := range ev.Content.Parts {
				if p.Text != "" {
					lastText = p.Text
				}
			}
		}
	}

	var explanation agent.Explanation
	err := json.Unmarshal([]byte(lastText), &explanation)
	if err != nil {
		// Fallback for non-JSON response
		explanation.Summary = lastText
	}
	return explanation, nil
}

const explainerSystemPrompt = `You are a GEO (Generative Engine Optimization) analyst for the Victoria's Secret brand family.
You receive structured data showing how brand visibility changed between two AI tracking runs.
Respond ONLY with a valid JSON object — no markdown fences, no explanation outside the JSON:
{"summary": "2-3 sentence plain-English explanation of what changed and why", "drivers": ["specific factor 1", "specific factor 2"]}
Reference concrete numbers, specific prompt categories, and named competitors. Never be vague.`

// ─── Recommender Agent ──────────────────────────────────────────────────────

type RecommenderAgent struct {
	runner *adkrunner.Runner
}

func NewRecommenderAgent(ctx context.Context, cfg config.ADKConfig) (*RecommenderAgent, error) {
	model, err := NewADKModel(ctx, cfg, cfg.RecommenderModel)
	if err != nil {
		return nil, fmt.Errorf("recommender model: %w", err)
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "lighthouse_recommender",
		Model:       model,
		Instruction: recommenderSystemPrompt,
	})
	if err != nil {
		return nil, err
	}
	r, err := adkrunner.New(adkrunner.Config{
		AppName:        "lighthouse",
		Agent:          a,
		SessionService: adksession.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return &RecommenderAgent{runner: r}, nil
}

func (r *RecommenderAgent) Recommend(ctx context.Context, req agent.RecommendationRequest) ([]db.Recommendation, error) {
	prompt := fmt.Sprintf("Generate recommendations for brand %s.\nData: %v", req.Brand, req)
	
	iter := r.runner.Run(ctx, "system", "recommend-run", &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}, adkagent.RunConfig{})

	var lastText string
	for ev, err := range iter {
		if err != nil {
			return nil, err
		}
		if ev.Content != nil {
			for _, p := range ev.Content.Parts {
				if p.Text != "" {
					lastText = p.Text
				}
			}
		}
	}

	var recs []db.Recommendation
	err := json.Unmarshal([]byte(lastText), &recs)
	return recs, err
}

const recommenderSystemPrompt = `You are a GEO strategist for the Victoria's Secret brand family (Adore Me + Victoria's Secret).
You receive structured visibility data: mention rates, citation gaps, stability scores, competitor share.
Return ONLY a JSON array of 3-5 prioritised actions. Each action must reference specific data from the input.
No markdown. No preamble. Only the JSON array.
Shape: [{"priority":1,"category":"fit","action":"...","expected_impact":"...","rationale":"..."}]`

// Strategy Agent ─────────────────────────────────────────────────────────

type StrategyAgent struct {
	agent        adkagent.Agent
	sessionStore adksession.Service
	runner       *adkrunner.Runner
}

func NewStrategyAgent(
	ctx context.Context,
	cfg config.ADKConfig,
	tools *ToolSet,
	store adksession.Service,
) (*StrategyAgent, error) {
	model, err := NewADKModel(ctx, cfg, cfg.StrategyModel)
	if err != nil {
		return nil, fmt.Errorf("strategy model: %w", err)
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "lighthouse_strategy",
		Model:       model,
		Instruction: strategySystemPrompt,
		Tools:       tools.Tools(),
	})
	if err != nil {
		return nil, err
	}
	r, err := adkrunner.New(adkrunner.Config{
		AppName:        "lighthouse",
		Agent:          a,
		SessionService: store,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return &StrategyAgent{agent: a, sessionStore: store, runner: r}, nil
}

type ChatEvent struct {
	Type    string `json:"type"` // "chunk" | "tool_call" | "tool_result" | "done" | "error"
	Text    string `json:"text,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Args    any    `json:"args,omitempty"`
	Preview string `json:"preview,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *StrategyAgent) Chat(ctx context.Context, sessionID, brand, message string) (<-chan ChatEvent, error) {
	out := make(chan ChatEvent, 10)
	
	iter := s.runner.Run(ctx, brand, sessionID, &genai.Content{
		Parts: []*genai.Part{{Text: message}},
	}, adkagent.RunConfig{
		StreamingMode: adkagent.StreamingModeSSE,
	})

	go func() {
		defer close(out)
		for ev, err := range iter {
			if err != nil {
				out <- ChatEvent{Type: "error", Error: err.Error()}
				return
			}

			if ev.Content != nil {
				for _, p := range ev.Content.Parts {
					if p.Text != "" {
						out <- ChatEvent{Type: "chunk", Text: p.Text}
					}
					if p.FunctionCall != nil {
						out <- ChatEvent{Type: "tool_call", Tool: p.FunctionCall.Name, Args: p.FunctionCall.Args}
					}
					if p.FunctionResponse != nil {
						out <- ChatEvent{Type: "tool_result", Tool: p.FunctionResponse.Name, Preview: "Data retrieved"}
					}
				}
			}
		}
		out <- ChatEvent{Type: "done"}
	}()

	return out, nil
}

const strategySystemPrompt = `You are the Lighthouse Strategy Agent — a GEO intelligence assistant for the Adore Me and Victoria's Secret brand team.
You have access to real visibility data through your tools. Always call the relevant tool before answering data questions — never guess.
Be specific: cite actual scores, actual domains, actual category names from data you retrieve.
When asked what to prioritise, call get_citation_gaps and get_visibility_trend first, then reason over both.
When asked about a past recommendation, call search_recommendations before responding.
You remember the conversation history in this session — refer back to decisions made earlier when relevant.
Keep responses concise and actionable. The team reading this is technical and time-pressured.`

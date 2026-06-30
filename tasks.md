# Project Lighthouse — ADK Refactoring Task List
### geo-tracker-backend · Agent-Native Pipeline

> Hand this file to Claude Code. Work tasks in order. Each task has a
> verification step — do not proceed to the next task until it passes.

---

## Context: what we're doing and why

The existing backend has a working goroutine runner (`runner/runner.go`) and two
LLM agent stubs (`agent/explainer.go`, `agent/recommender.go`) with TODO placeholders.

We are replacing the orchestration layer with **Google ADK Go**
(`google.golang.org/adk`) to get:
- A real multi-agent pipeline with declared parallel phases
- Real explainer and recommender agents (replacing the stubs)
- A conversational Strategy Agent with DB-backed tools and session memory
- A run trace table that makes the agent graph visible in the dashboard

**Nothing in `providers/` or `scoring/` changes.**

---

## Files: what happens to each one

| File | Action | Notes |
|---|---|---|
| `internal/providers/*.go` | **KEEP, zero changes** | Pure HTTP calls, no orchestration |
| `internal/scoring/visibility.go` | **KEEP, zero changes** | Pure math |
| `internal/db/schema.sql` | **ADDITIVE** | 2 new tables appended at bottom |
| `internal/db/results.go` | **ADDITIVE** | New query functions for ADK tools only |
| `internal/config/config.go` | **ADDITIVE** | New `ADKConfig` struct + field |
| `config.yaml` | **ADDITIVE** | New `adk:` section |
| `internal/runner/runner.go` | **REPLACE** | ADK pipeline replaces goroutine pool |
| `internal/agent/explainer.go` | **REPLACE** | Real `LlmAgent` replaces stub |
| `internal/agent/recommender.go` | **REPLACE** | Real `LlmAgent` replaces stub |
| `internal/agent/extractor.go` | **KEEP** | Called by pipeline Phase 2 unchanged |
| `internal/adk/` | **NEW PACKAGE** | Pipeline, model factory, tools, agents, memory |
| `internal/api/handlers.go` | **ADDITIVE** | New `/api/strategy/chat` + `/api/runs/{id}/trace` |
| `internal/api/server.go` | **ADDITIVE** | Register 2 new routes |
| `cmd/run.go` | **UPDATE** | Call `pipeline.Run()` instead of `runner.RunAll()` |
| `cmd/serve.go` | **UPDATE** | Construct and inject `StrategyAgent` into handler |
| `CLAUDE.md` | **ADDITIVE** | ADK rules section appended |

---

## Task 0 — Add ADK Go dependency

```bash
go get google.golang.org/adk@latest
go mod tidy
```

**Verify:** `go build ./...` compiles with zero errors.

**Hard gate:** Do not proceed to Task 1 until this passes.

---

## Task 1 — ADK config: provider-agnostic design

The ADK agents (explainer, recommender, strategy) can run on any LLM backend —
Gemini, Claude, or any model ADK supports. The config captures the provider name,
the API key for that provider, and the model string for each agent role.
This keeps the agent layer completely decoupled from the probe provider config.

### 1a — `internal/config/config.go`

Add to the file (do not modify any existing structs):

```go
// ADKConfig controls which LLM backend powers the agent layer.
// This is separate from the providers config, which controls probe calls.
// Set Provider to "gemini" or "anthropic". APIKey is the key for that provider.
// Model strings must match the chosen provider's model naming convention.
type ADKConfig struct {
    Provider         string `mapstructure:"provider"`          // "gemini" | "anthropic"
    APIKey           string `mapstructure:"api_key"`           // provider-agnostic key field
    StrategyModel    string `mapstructure:"strategy_model"`
    ExplainerModel   string `mapstructure:"explainer_model"`
    RecommenderModel string `mapstructure:"recommender_model"`
    SessionTTLDays   int    `mapstructure:"session_ttl_days"`
}
```

Add the field to the root `Config` struct:
```go
ADK ADKConfig `mapstructure:"adk"`
```

Add validation in `Validate()`:
```go
// In Validate():
validADKProviders := map[string]bool{"gemini": true, "anthropic": true}
if cfg.ADK.Provider != "" && !validADKProviders[cfg.ADK.Provider] {
    errs = append(errs, fmt.Errorf("adk.provider must be 'gemini' or 'anthropic', got %q", cfg.ADK.Provider))
}
if cfg.ADK.Provider != "" && cfg.ADK.APIKey == "" {
    errs = append(errs, fmt.Errorf("adk.api_key is required when adk.provider is set"))
}
```

### 1b — `config.yaml`

Append at the bottom. Do not modify existing sections.

```yaml
# ADK agent layer — controls LLM backend for explainer, recommender, strategy agents.
# This is separate from the providers section (probe calls).
#
# provider options: "gemini" | "anthropic"
# api_key:          your Gemini API key (if provider = gemini)
#                   your Anthropic API key (if provider = anthropic)
#
# Gemini model examples:  "gemini-2.0-flash", "gemini-1.5-pro"
# Anthropic model examples: "claude-sonnet-4-6", "claude-haiku-4-5-20251001"
adk:
  provider: "gemini"
  api_key: ""                            # GEOTRACKER_ADK_API_KEY env var
  strategy_model: "gemini-2.0-flash"
  explainer_model: "gemini-2.0-flash"
  recommender_model: "gemini-2.0-flash"
  session_ttl_days: 30
```

> **To switch to Anthropic:** change `provider` to `"anthropic"`, set `api_key` to your
> Anthropic key, and update model strings to `"claude-sonnet-4-6"` etc.
> No code changes required — only config.

### 1c — `config.local.yaml` (gitignored)

This is where the actual key lives locally:
```yaml
adk:
  provider: "gemini"
  api_key: "YOUR_GEMINI_API_KEY_HERE"
```

In Kubernetes, inject via:
```
GEOTRACKER_ADK_PROVIDER=gemini
GEOTRACKER_ADK_API_KEY=<secret>
```

**Verify:** `go test ./internal/config/...` passes.

---

## Task 2 — Add new DB tables to schema

**File:** `internal/db/schema.sql`

Append at the very bottom. Do not modify any existing table definitions.

```sql
-- ─────────────────────────────────────────────────────────────────────────────
-- ADK agent layer tables (added for ADK refactor)
-- ─────────────────────────────────────────────────────────────────────────────

-- Agent session memory — persists Strategy Agent conversation state across requests.
-- One row per brand per user session. `data` is a JSON blob of ADK session state.
CREATE TABLE IF NOT EXISTS agent_sessions (
    id          VARCHAR(64) PRIMARY KEY,
    brand       VARCHAR(128) NOT NULL,
    data        JSON NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_sessions_brand (brand)
);

-- Run traces — one row per agent per pipeline phase per run.
-- Used by GET /api/runs/:id/trace to render the agent timeline in the dashboard.
CREATE TABLE IF NOT EXISTS run_traces (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id      BIGINT UNSIGNED NOT NULL,
    phase       VARCHAR(64)  NOT NULL,   -- probe | intelligence | insight
    agent_name  VARCHAR(128) NOT NULL,   -- e.g. "claude_prober", "extractor", "explainer"
    started_at  DATETIME(3)  NOT NULL,
    finished_at DATETIME(3),
    duration_ms INT,
    status      VARCHAR(32)  NOT NULL,   -- running | success | error | retried
    error_text  TEXT,
    FOREIGN KEY (run_id) REFERENCES runs(id),
    INDEX idx_traces_run (run_id)
);
```

Apply to local DB:
```bash
mysql -u root geo_tracker < internal/db/schema.sql
```

**Verify:** `SHOW TABLES;` in MySQL shows `agent_sessions` and `run_traces`.
No existing tables were dropped or altered.

---

## Task 3 — New DB structs and query functions

**File:** `internal/db/results.go`

Add the following structs and functions. Do not modify any existing code.

### 3a — New structs

```go
// RunTrace maps to the run_traces table.
type RunTrace struct {
    ID         uint64     `db:"id"`
    RunID      uint64     `db:"run_id"`
    Phase      string     `db:"phase"`
    AgentName  string     `db:"agent_name"`
    StartedAt  time.Time  `db:"started_at"`
    FinishedAt *time.Time `db:"finished_at"`
    DurationMS *int       `db:"duration_ms"`
    Status     string     `db:"status"`
    ErrorText  *string    `db:"error_text"`
}

// AgentSession maps to the agent_sessions table.
type AgentSession struct {
    ID        string    `db:"id"`
    Brand     string    `db:"brand"`
    Data      string    `db:"data"` // serialized JSON blob
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

### 3b — New query functions

```go
// --- Strategy Agent tools (read-only) ---

// GetVisibilityTrend returns the last N visibility scores for a brand across runs.
// Used by Strategy Agent tool: get_visibility_trend
func (r *ResultsRepo) GetVisibilityTrend(brand string, limit int) ([]TrendPoint, error)

// GetCompetitorShare returns the top competitors by mention count for a run.
// Used by Strategy Agent tool: get_competitor_share
func (r *ResultsRepo) GetCompetitorShare(brand string, runID uint64) ([]CompetitorCount, error)

// SearchRecommendations returns recommendations for a brand filtered by status.
// status: "pending" | "implemented" | "" (all)
// Used by Strategy Agent tool: search_recommendations
func (r *ResultsRepo) SearchRecommendations(brand string, status string) ([]Recommendation, error)

// --- Run trace ---

// GetRunTrace returns all trace rows for a run, ordered by started_at.
// Used by GET /api/runs/:id/trace handler.
func (r *ResultsRepo) GetRunTrace(runID uint64) ([]RunTrace, error)

// InsertRunTrace writes a new trace row. Called by pipeline at phase start.
func (r *ResultsRepo) InsertRunTrace(trace *RunTrace) error

// UpdateRunTrace sets finished_at, duration_ms, status, error_text on an existing row.
// Called by pipeline at phase end.
func (r *ResultsRepo) UpdateRunTrace(id uint64, finishedAt time.Time, durationMS int, status, errText string) error

// --- Session store ---

// GetAgentSession loads a session by ID. Returns nil, nil if not found.
func (r *ResultsRepo) GetAgentSession(id string) (*AgentSession, error)

// UpsertAgentSession inserts or updates a session row (INSERT ... ON DUPLICATE KEY UPDATE).
func (r *ResultsRepo) UpsertAgentSession(session *AgentSession) error

// DeleteAgentSession removes a session row. Called during session cleanup.
func (r *ResultsRepo) DeleteAgentSession(id string) error
```

**Verify:** `go build ./internal/db/...` compiles cleanly.

---

## Task 4 — Create `internal/adk/` package

Create the directory. This package is the heart of the refactor.
Files within it must not import from `internal/runner/` — that package is being replaced.

---

### Task 4a — `internal/adk/model.go` — provider-agnostic model factory

This is the key file that makes the agent provider swappable via config.
It reads `cfg.ADK.Provider` and returns the correct ADK model instance.

```go
package adk

import (
    "context"
    "fmt"

    "github.com/adoreme/geo-tracker/internal/config"
    adkmodel "google.golang.org/adk/model"
    "google.golang.org/adk/model/anthropic"
    "google.golang.org/adk/model/gemini"
    "google.golang.org/genai"
)

// NewADKModel returns an ADK model instance for the given model string,
// using the provider and API key from ADKConfig.
//
// provider = "gemini"    → uses google.golang.org/adk/model/gemini
// provider = "anthropic" → uses google.golang.org/adk/model/anthropic
//
// The model string must match the chosen provider's naming convention:
//   gemini:    "gemini-2.0-flash", "gemini-1.5-pro", etc.
//   anthropic: "claude-sonnet-4-6", "claude-haiku-4-5-20251001", etc.
func NewADKModel(ctx context.Context, cfg config.ADKConfig, modelStr string) (adkmodel.Model, error) {
    switch cfg.Provider {
    case "gemini":
        return gemini.NewModel(ctx, modelStr, &genai.ClientConfig{
            APIKey: cfg.APIKey,
        })
    case "anthropic":
        return anthropic.NewModel(ctx, modelStr, cfg.APIKey)
    default:
        return nil, fmt.Errorf("unsupported adk.provider %q: must be 'gemini' or 'anthropic'", cfg.Provider)
    }
}
```

> **Note on the `anthropic` ADK adapter:** As of ADK Go v1.4, the `anthropic` model
> package in `google.golang.org/adk/model/anthropic` may not be available in Go
> (it's confirmed for Java/Python). If it is absent, use the `litellm` adapter
> or route Anthropic calls through the `gemini` package using Vertex AI's
> Anthropic endpoint. Check `pkg.go.dev/google.golang.org/adk` after Task 0
> and adjust the import path accordingly. The config contract (`provider` +
> `api_key` + model string) does not change regardless.

**Verify:** `go build ./internal/adk/...` compiles. Fix import paths if the
`anthropic` sub-package doesn't exist — use LiteLLM adapter as fallback.

---

### Task 4b — `internal/adk/memory.go` — MySQL session store

Implements the ADK `session.Store` interface so the Strategy Agent remembers
conversations across HTTP requests.

```go
package adk

import (
    "context"
    "encoding/json"
    "time"

    "github.com/adoreme/geo-tracker/internal/db"
    adksession "google.golang.org/adk/session"
)

// MySQLSessionStore implements adksession.Store using the agent_sessions table.
type MySQLSessionStore struct {
    repo *db.ResultsRepo
}

func NewMySQLSessionStore(repo *db.ResultsRepo) *MySQLSessionStore {
    return &MySQLSessionStore{repo: repo}
}

// Implement the adksession.Store interface.
// Check google.golang.org/adk/session for the exact method signatures —
// they may be Get/Save/Delete or Load/Save/Delete depending on ADK version.
//
// Serialization contract:
//   - Marshal adksession.Session to JSON for the `data` column.
//   - Brand is extracted from session.State["brand"] and stored in the brand column
//     to allow brand-scoped queries.
//
// Error handling:
//   - Not-found: return nil, nil (not an error).
//   - DB errors: wrap and return.
```

**Verify:** `go build ./internal/adk/...` compiles.

---

### Task 4c — `internal/adk/tools.go` — Strategy Agent tool functions

Wraps existing DB query functions as ADK callable tools.
All tool functions are read-only except `mark_recommendation_done`.

```go
package adk

import (
    "context"

    "github.com/adoreme/geo-tracker/internal/db"
    "google.golang.org/adk/tool"
)

// ToolSet holds tool functions bound to a DB repo instance.
type ToolSet struct {
    repo *db.ResultsRepo
}

func NewToolSet(repo *db.ResultsRepo) *ToolSet {
    return &ToolSet{repo: repo}
}

// Tools returns the slice of ADK tools to pass to the Strategy Agent.
func (t *ToolSet) Tools() []tool.Tool {
    return []tool.Tool{
        tool.FromFunc("get_visibility_trend",     t.getVisibilityTrend),
        tool.FromFunc("get_citation_gaps",        t.getCitationGaps),
        tool.FromFunc("get_stability_scores",     t.getStabilityScores),
        tool.FromFunc("get_competitor_share",     t.getCompetitorShare),
        tool.FromFunc("search_recommendations",   t.searchRecommendations),
        tool.FromFunc("mark_recommendation_done", t.markRecommendationDone),
    }
}

// Each tool function uses a typed args struct and a typed result struct.
// ADK serializes these automatically — no manual JSON handling.

type VisibilityTrendArgs struct {
    Brand string `json:"brand" description:"Brand name, e.g. 'Adore Me'"`
    Limit int    `json:"limit" description:"Number of past runs to return, default 10"`
}
type VisibilityTrendResult struct {
    Points []db.TrendPoint `json:"points"`
}
func (t *ToolSet) getVisibilityTrend(ctx context.Context, args VisibilityTrendArgs) (VisibilityTrendResult, error)

type CitationGapArgs struct {
    Brand string `json:"brand"`
    RunID uint64 `json:"run_id" description:"Run ID to analyse. Use 0 for latest run."`
}
type CitationGapResult struct {
    Gaps []db.CitationGapEntry `json:"gaps"`
}
func (t *ToolSet) getCitationGaps(ctx context.Context, args CitationGapArgs) (CitationGapResult, error)

type StabilityArgs struct {
    Brand string `json:"brand"`
    RunID uint64 `json:"run_id"`
}
type StabilityResult struct {
    Scores []db.StabilityScore `json:"scores"`
}
func (t *ToolSet) getStabilityScores(ctx context.Context, args StabilityArgs) (StabilityResult, error)

type CompetitorShareArgs struct {
    Brand string `json:"brand"`
    RunID uint64 `json:"run_id"`
}
type CompetitorShareResult struct {
    Competitors []db.CompetitorCount `json:"competitors"`
}
func (t *ToolSet) getCompetitorShare(ctx context.Context, args CompetitorShareArgs) (CompetitorShareResult, error)

type SearchRecsArgs struct {
    Brand  string `json:"brand"`
    Status string `json:"status" description:"'pending', 'implemented', or '' for all"`
}
type SearchRecsResult struct {
    Recommendations []db.Recommendation `json:"recommendations"`
}
func (t *ToolSet) searchRecommendations(ctx context.Context, args SearchRecsArgs) (SearchRecsResult, error)

type MarkDoneArgs struct {
    RecommendationID int64 `json:"recommendation_id"`
}
type MarkDoneResult struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}
func (t *ToolSet) markRecommendationDone(ctx context.Context, args MarkDoneArgs) (MarkDoneResult, error)
```

**Verify:** `go build ./internal/adk/...` compiles.

---

### Task 4d — `internal/adk/agents.go` — LlmAgent definitions

Three agents. All use `NewADKModel()` from `model.go` — switching provider is
one config line, zero code changes.

```go
package adk

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/adoreme/geo-tracker/internal/config"
    "github.com/adoreme/geo-tracker/internal/db"
    "google.golang.org/adk/agent/llmagent"
    adkrunner "google.golang.org/adk/runner"
    adksession "google.golang.org/adk/session"
)

// ─── Explainer Agent ────────────────────────────────────────────────────────

// ExplainerAgent generates a plain-English diff explanation between two runs.
// Uses the model from cfg.ADK.ExplainerModel.
// No tools — all context is passed in the prompt.
// Returns structured JSON parsed into an Explanation.
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
    r := adkrunner.New(a, adksession.NewInMemoryStore(), nil)
    return &ExplainerAgent{runner: r}, nil
}

// Explain builds a structured prompt from req, calls the LlmAgent,
// and parses the JSON response into an Explanation.
func (e *ExplainerAgent) Explain(ctx context.Context, req ExplainRequest) (Explanation, error)

// ExplainRequest and Explanation types stay in internal/agent/explainer.go
// (imported here) so cmd/ code referencing them doesn't need to change.

const explainerSystemPrompt = `You are a GEO (Generative Engine Optimization) analyst for the Victoria's Secret brand family.
You receive structured data showing how brand visibility changed between two AI tracking runs.
Respond ONLY with a valid JSON object — no markdown fences, no explanation outside the JSON:
{"summary": "2-3 sentence plain-English explanation of what changed and why", "drivers": ["specific factor 1", "specific factor 2"]}
Reference concrete numbers, specific prompt categories, and named competitors. Never be vague.`

// ─── Recommender Agent ──────────────────────────────────────────────────────

// RecommenderAgent generates 3-5 prioritised GEO actions from run data.
// Uses the model from cfg.ADK.RecommenderModel.
// No tools — all context is passed in the prompt.
// Returns a JSON array parsed into []db.Recommendation.
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
    r := adkrunner.New(a, adksession.NewInMemoryStore(), nil)
    return &RecommenderAgent{runner: r}, nil
}

func (r *RecommenderAgent) Recommend(ctx context.Context, req RecommendationRequest) ([]db.Recommendation, error)

const recommenderSystemPrompt = `You are a GEO strategist for the Victoria's Secret brand family (Adore Me + Victoria's Secret).
You receive structured visibility data: mention rates, citation gaps, stability scores, competitor share.
Return ONLY a JSON array of 3-5 prioritised actions. Each action must reference specific data from the input.
No markdown. No preamble. Only the JSON array.
Shape: [{"priority":1,"category":"fit","action":"...","expected_impact":"...","rationale":"..."}]`

// ─── Strategy Agent ─────────────────────────────────────────────────────────

// StrategyAgent is a conversational agent with 6 DB-backed tools and persistent
// session memory. Powers the /api/strategy/chat SSE endpoint.
// Uses the model from cfg.ADK.StrategyModel.
type StrategyAgent struct {
    agent        *llmagent.Agent
    sessionStore *MySQLSessionStore
    runner       *adkrunner.Runner
}

func NewStrategyAgent(
    ctx context.Context,
    cfg config.ADKConfig,
    tools *ToolSet,
    store *MySQLSessionStore,
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
    r := adkrunner.New(a, store, nil)
    return &StrategyAgent{agent: a, sessionStore: store, runner: r}, nil
}

// Chat sends one user message and streams the agent response as SSE chunks.
// sessionID is brand-scoped (one per brand per UI session).
// Returns a channel of ChatEvent for the SSE handler to forward.
func (s *StrategyAgent) Chat(ctx context.Context, sessionID, brand, message string) (<-chan ChatEvent, error)

// ChatEvent is one SSE payload sent to the frontend.
type ChatEvent struct {
    Type       string `json:"type"`        // "chunk" | "tool_call" | "tool_result" | "done" | "error"
    Text       string `json:"text,omitempty"`
    Tool       string `json:"tool,omitempty"`
    Args       any    `json:"args,omitempty"`
    Preview    string `json:"preview,omitempty"` // short human-readable summary of tool result
    Error      string `json:"error,omitempty"`
}

const strategySystemPrompt = `You are the Lighthouse Strategy Agent — a GEO intelligence assistant for the Adore Me and Victoria's Secret brand team.
You have access to real visibility data through your tools. Always call the relevant tool before answering data questions — never guess.
Be specific: cite actual scores, actual domains, actual category names from data you retrieve.
When asked what to prioritise, call get_citation_gaps and get_visibility_trend first, then reason over both.
When asked about a past recommendation, call search_recommendations before responding.
You remember the conversation history in this session — refer back to decisions made earlier when relevant.
Keep responses concise and actionable. The team reading this is technical and time-pressured.`
```

**Verify:** `go build ./internal/adk/...` compiles. No LLM calls happen at construction time.

---

### Task 4e — `internal/adk/pipeline.go` — the orchestration core

Replaces `internal/runner/runner.go`. The goroutine pool logic moves into
Phase 1 of this file. Phases 2 and 3 use `errgroup` for clean parallel execution.

```go
package adk

import (
    "context"
    "sync"
    "time"

    "github.com/adoreme/geo-tracker/internal/config"
    "github.com/adoreme/geo-tracker/internal/db"
    "github.com/adoreme/geo-tracker/internal/providers"
    "github.com/adoreme/geo-tracker/internal/scoring"
    "github.com/adoreme/geo-tracker/internal/agent"
    "go.uber.org/zap"
    "golang.org/x/sync/errgroup"
)

// PipelineResult is the full output of one pipeline execution.
type PipelineResult struct {
    RunID            uint64
    Results          []db.Result
    StabilityScores  []db.StabilityScore
    VisibilityScores map[string]float64           // brand → score
    Explanations     map[string]agent.Explanation  // brand → explanation
    Recommendations  []db.Recommendation
    Traces           []db.RunTrace
    TotalCostUSD     float64
}

// Pipeline orchestrates the three phases of a Lighthouse run.
type Pipeline struct {
    cfg          config.Config
    repo         *db.ResultsRepo
    providers    []providers.Provider
    explainer    *ExplainerAgent
    recommender  *RecommenderAgent
    logger       *zap.Logger
}

func NewPipeline(
    cfg config.Config,
    repo *db.ResultsRepo,
    providers []providers.Provider,
    explainer *ExplainerAgent,
    recommender *RecommenderAgent,
    logger *zap.Logger,
) *Pipeline

// Run executes all three phases and returns the full result.
// All DB writes happen inside Run — cmd/run.go only calls Run and prints the summary.
//
// ── Phase 1: PROBE (parallel) ────────────────────────────────────────────────
//   Fan out all prompt × provider × brand × sample jobs using a goroutine worker pool.
//   Worker count: cfg.Runner.Workers
//   Rate limit:   cfg.Runner.RateLimitPerMinute (time.Ticker)
//   Each job: provider.Probe() → write db.Result row immediately (not batched)
//   Trace: one RunTrace row per provider group (claude, chatgpt, perplexity, gemini)
//
// ── Phase 2: INTELLIGENCE (parallel via errgroup) ────────────────────────────
//   A. Extraction: for each result row without a GEOSignal, call agent.Extract()
//      and update the row. Uses cfg.Providers.Claude or cfg.Providers.Gemini
//      for the extract call (unchanged from current runner.go logic).
//   B. Stability scoring: call scoring.CalcStabilityScore per prompt×provider×brand group.
//   A and B run concurrently. Both write to DB on completion.
//   Trace: one RunTrace row each for "extractor" and "stability_scorer".
//
// ── Phase 3: INSIGHT (parallel via errgroup, per brand) ──────────────────────
//   A. Explainer: fetch previous run, build ExplainRequest, call ExplainerAgent.Explain().
//      Store result in runs.explanation_json (add column if not present) and log it.
//   B. Recommender: fetch citation gaps + stability + competitors, call
//      RecommenderAgent.Recommend(). Insert rows into recommendations table.
//   A and B run concurrently per brand.
//   Trace: one RunTrace row each for "explainer" and "recommender".
//
func (p *Pipeline) Run(ctx context.Context, run db.Run, prompts []db.Prompt) (PipelineResult, error)

// traceStart inserts a run_trace row with status="running" and returns it.
func (p *Pipeline) traceStart(ctx context.Context, runID uint64, phase, agentName string) *db.RunTrace

// traceEnd updates the run_trace row with finished_at, duration_ms, and status.
// Pass err=nil for success, non-nil for error.
func (p *Pipeline) traceEnd(ctx context.Context, trace *db.RunTrace, err error)
```

**Verify:** `go build ./internal/adk/...` compiles. Run `go vet ./internal/adk/...` and fix any issues.

---

## Task 5 — Update `internal/agent/explainer.go`

Keep all existing types (`ExplainRequest`, `PromptDiff`, `Explanation`) exactly as-is —
they are referenced by `cmd/run.go` and must not change signatures.

Replace only the `Explain` function body. It now delegates to `ExplainerAgent`:

```go
// Explain is now a thin delegate. The agent is injected, not constructed here.
// Types (ExplainRequest, Explanation) stay in this file unchanged.
func Explain(ctx context.Context, req ExplainRequest, a *adkpkg.ExplainerAgent) (Explanation, error) {
    return a.Explain(ctx, req)
}
```

Remove the TODO stub body and the placeholder return.

**Verify:** `go build ./internal/agent/...` compiles.

---

## Task 6 — Update `internal/agent/recommender.go`

Same pattern as Task 5.

```go
func Recommend(ctx context.Context, req RecommendationRequest, a *adkpkg.RecommenderAgent) ([]db.Recommendation, error) {
    return a.Recommend(ctx, req)
}
```

Remove the TODO stub body and the hardcoded placeholder recommendation.

**Verify:** `go build ./internal/agent/...` compiles.

---

## Task 7 — Update `cmd/run.go`

This is the cutover. Replace the manual orchestration block with a single pipeline call.

### 7a — Construct ADK components (once, before cobra RunE)

```go
// After cfg and resultRepo are initialised, before RunE is defined:
explainerAgent, err := adkpkg.NewExplainerAgent(ctx, cfg.ADK)
if err != nil {
    logger.Fatal("explainer agent init failed", zap.Error(err))
}
recommenderAgent, err := adkpkg.NewRecommenderAgent(ctx, cfg.ADK)
if err != nil {
    logger.Fatal("recommender agent init failed", zap.Error(err))
}
```

### 7b — Replace the RunAll block inside RunE

**Remove this block (current code):**
```go
rn := runner.NewRunner(*cfg, logger)
results := rn.RunAll(ctx, prompts, enabledProviders, brands)
// ... manual result insert loop
// ... manual stability score triple-nested loop
// ... manual explainer/recommender calls per brand
```

**Replace with:**
```go
pipe := adkpkg.NewPipeline(*cfg, resultRepo, enabledProviders, explainerAgent, recommenderAgent, logger)
pipeResult, err := pipe.Run(ctx, run, prompts)
if err != nil && exitCode {
    os.Exit(1)
}
// Results, stability scores, explanations, recommendations already written to DB by pipeline.
// Just update run status and print summary:
resultRepo.UpdateRunStatus(run.ID, "done", pipeResult.TotalCostUSD)
printFancySummary(run.ID, pipeResult)
```

**Verify:** `go run . run --dry-run` completes without panic. All three pipeline phases should log to stdout.

---

## Task 8 — Update `cmd/serve.go`

Construct the Strategy Agent and inject it into the API handler.

```go
// In the serve command, after resultRepo is set up:
toolSet := adkpkg.NewToolSet(resultRepo)
sessionStore := adkpkg.NewMySQLSessionStore(resultRepo)
strategyAgent, err := adkpkg.NewStrategyAgent(ctx, cfg.ADK, toolSet, sessionStore)
if err != nil {
    logger.Fatal("strategy agent init failed", zap.Error(err))
}
// Pass strategyAgent to the handler constructor (Task 9 adds the field)
handler := api.NewHandler(resultRepo, *cfg, logger, strategyAgent)
```

**Verify:** `go run . serve` starts cleanly.

---

## Task 9 — Add new API endpoints

### 9a — `internal/api/handlers.go`

Add two handlers. Do not modify any existing handlers.

```go
// StrategyChatHandler — POST /api/strategy/chat
//
// Request body:
//   {"brand": "Adore Me", "message": "Why did our score drop?", "session_id": "uuid"}
//
// Response: Server-Sent Events stream, one JSON object per line:
//   data: {"type":"chunk","text":"Based on your data..."}
//   data: {"type":"tool_call","tool":"get_visibility_trend","args":{"brand":"Adore Me","limit":5}}
//   data: {"type":"tool_result","tool":"get_visibility_trend","preview":"5 data points returned"}
//   data: {"type":"chunk","text":"Your score dropped from 42 to 34..."}
//   data: {"type":"done"}
//
// The tool_call and tool_result events are the key demo moment —
// the frontend renders them as collapsible "Checking data..." badges.
func (h *Handler) StrategyChatHandler(w http.ResponseWriter, r *http.Request)

// RunTraceHandler — GET /api/runs/{id}/trace
//
// Returns all trace rows for a run as a JSON array.
// Used by the frontend to render the agent timeline on /runs/:id.
func (h *Handler) RunTraceHandler(w http.ResponseWriter, r *http.Request)
```

Add `strategyAgent` field to the `Handler` struct:
```go
type Handler struct {
    repo          *db.ResultsRepo
    cfg           config.Config
    logger        *zap.Logger
    strategyAgent *adkpkg.StrategyAgent  // NEW
}

// Update NewHandler signature:
func NewHandler(repo *db.ResultsRepo, cfg config.Config, logger *zap.Logger, strategyAgent *adkpkg.StrategyAgent) *Handler
```

### 9b — `internal/api/server.go`

Register the two new routes. Do not change existing routes.

```go
r.Post("/api/strategy/chat",      h.StrategyChatHandler)
r.Get("/api/runs/{id}/trace",     h.RunTraceHandler)
```

**Verify:**
```bash
go run . serve &
# Should return 400 (missing body), not 500
curl -s -o /dev/null -w "%{http_code}" -X POST localhost:8080/api/strategy/chat
# Should return 200 with JSON array (after a real run has been stored)
curl localhost:8080/api/runs/1/trace
```

---

## Task 10 — Integration smoke test

Run through all gates in sequence. Fix failures before moving on.

```bash
# Gate 1: full build
go build ./...

# Gate 2: config validation
go run . config validate
# Expected: "config OK" with adk.provider and adk.api_key shown as set

# Gate 3: dry run (no DB writes, no API calls)
go run . run --dry-run --verbose
# Expected: 3 phases logged, PipelineResult printed, no panic

# Gate 4: real run (requires DB + API keys in config.local.yaml)
go run . run --verbose
# Expected: results in DB, run_traces rows written, recommendations inserted

# Gate 5: serve + strategy chat
go run . serve &
curl -X POST localhost:8080/api/strategy/chat \
  -H "Content-Type: application/json" \
  -d '{"brand":"Adore Me","message":"What is my current visibility score?","session_id":"smoke-001"}'
# Expected: SSE stream with at least one tool_call event and a done event

# Gate 6: run trace
curl localhost:8080/api/runs/1/trace
# Expected: JSON array with probe/intelligence/insight trace rows

# Gate 7: results command unchanged
go run . results summary
# Expected: organic and comparison sections printed separately
```

**Hard gate:** All 7 gates must pass before hackathon day.

---

## Task 11 — Update `CLAUDE.md`

Append to the backend repo `CLAUDE.md`. Do not modify existing content.

```markdown
---

## ADK agent layer

Pipeline orchestration lives in `internal/adk/pipeline.go`.
Do not add sequencing logic to `cmd/run.go` — all phase ordering is in `Pipeline.Run()`.

### Provider selection

ADK agents use a separate provider from the probe providers.
Configured via `adk.provider` in `config.yaml` — either `"gemini"` or `"anthropic"`.
The API key is `adk.api_key` (provider-agnostic field name).
Model strings in `adk.strategy_model`, `adk.explainer_model`, `adk.recommender_model`
must match the naming convention of the chosen provider.

To switch from Gemini to Anthropic: change `adk.provider` and `adk.api_key` in
`config.local.yaml` and update the model strings. Zero code changes required.

### Three pipeline phases

1. **Probe** (parallel goroutine pool) — fires all provider × prompt × brand × sample jobs
2. **Intelligence** (parallel errgroup) — extraction + stability scoring run concurrently
3. **Insight** (parallel errgroup, per brand) — explainer + recommender run concurrently

### Agent types

| Agent | File | Model config key | Tools | Memory |
|---|---|---|---|---|
| ExplainerAgent | `internal/adk/agents.go` | `adk.explainer_model` | none | none |
| RecommenderAgent | `internal/adk/agents.go` | `adk.recommender_model` | none | none |
| StrategyAgent | `internal/adk/agents.go` | `adk.strategy_model` | 6 DB tools | MySQL session store |

### Rules

- Never call ADK agents from `cmd/` directly — always through `Pipeline` or `Handler`
- Strategy Agent session IDs are brand-scoped: format `"{brand}-{uuid}"`, e.g. `"adore-me-abc123"`
- Tool functions in `tools.go` are read-only except `mark_recommendation_done`
- Run traces are written by `Pipeline.traceStart/traceEnd` — never write them manually
- `adk.api_key` never appears in logs — same rule as all other API keys
- Model factory `NewADKModel()` is the only place that switches on `adk.provider` —
  do not add provider-switch logic anywhere else
```

---

## Task 12 — Generate demo baseline

Once Gate 4 (real run) passes:

```bash
go run . run --format json > demo/baseline.json
git add demo/baseline.json
git commit -m "chore: add hackathon demo baseline"
```

This is the `--demo` flag fallback if live API calls fail during the presentation.

---

## Summary: pre-hackathon gate checklist

| # | Gate | Command | Pass condition |
|---|---|---|---|
| 1 | Build | `go build ./...` | Zero errors |
| 2 | Config | `go run . config validate` | ADK provider + key shown |
| 3 | Dry run | `go run . run --dry-run` | 3 phases logged, no panic |
| 4 | Real run | `go run . run --verbose` | DB has results + traces |
| 5 | Serve | `go run . serve` | :8080 responds |
| 6 | Strategy chat | `curl POST /api/strategy/chat` | SSE with tool_call events |
| 7 | Run trace | `curl GET /api/runs/1/trace` | JSON array of trace rows |
| 8 | Results | `go run . results summary` | Organic + comparison separate |
| 9 | Demo baseline | `ls demo/baseline.json` | File exists and is non-empty |

---

*Project Lighthouse · ADK Refactor · Adore Me Tech Hackathon · June 2026*

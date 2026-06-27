# Project Lighthouse — Backend Implementation
### Coding Agent Document · geo-tracker-backend

---

## What this document is

Complete implementation specification for the `geo-tracker-backend` Go application.
Work through sections in order. Each section is independently testable before moving on.
Do not skip ahead — later sections depend on earlier ones being correct.

---

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go 1.23+ |
| CLI | cobra + viper |
| Database | MySQL 8 + sqlx |
| HTTP | chi v5 |
| Logging | zap |
| Testing | standard `testing` + testify |

---

## Repository structure

```
geo-tracker-backend/
├── cmd/
│   ├── root.go
│   ├── run.go
│   ├── results.go
│   ├── prompts.go
│   ├── serve.go
│   └── config.go
├── internal/
│   ├── config/config.go
│   ├── db/
│   │   ├── db.go
│   │   ├── schema.sql
│   │   ├── prompts.go
│   │   └── results.go
│   ├── providers/
│   │   ├── provider.go
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   ├── perplexity.go
│   │   └── gemini.go
│   ├── agent/
│   │   ├── extractor.go
│   │   ├── explainer.go
│   │   └── recommender.go
│   ├── runner/
│   │   └── runner.go
│   ├── scoring/
│   │   └── visibility.go
│   └── api/
│       ├── server.go
│       ├── handlers.go
│       └── dto.go
├── prompts/
│   └── seed.yaml
├── config.yaml
├── config.local.yaml       # gitignored
├── Dockerfile
├── go.mod
├── go.sum
└── main.go
```

---

## Section 1 — Config

### 1.1 config.yaml

```yaml
app:
  name: geo-tracker
  log_level: info

database:
  host: localhost
  port: 3306
  name: geo_tracker
  user: root
  password: ""
  max_open_conns: 10
  max_idle_conns: 5

brands:
  - name: "Adore Me"
  - name: "Victoria's Secret"

providers:
  claude:
    enabled: true
    api_key: ""
    probe_model: "claude-sonnet-4-6"
    extract_model: "claude-haiku-4-5-20251001"
    timeout_seconds: 30
  chatgpt:
    enabled: true
    api_key: ""
    probe_model: "gpt-4o"
    extract_model: "gpt-4o-mini"
    timeout_seconds: 30
  perplexity:
    enabled: true
    api_key: ""
    probe_model: "llama-3.1-sonar-large-128k-online"
    timeout_seconds: 30
  gemini:
    enabled: true
    api_key: ""
    probe_model: "gemini-2.0-flash"
    timeout_seconds: 30

runner:
  workers: 8
  samples_per_prompt: 3          # stability analysis — run each prompt N times
  retry_attempts: 2
  retry_delay_seconds: 5
  rate_limit_per_minute: 60

serve:
  host: "0.0.0.0"
  port: 8080
  cors_origins:
    - "http://localhost:3000"
```

### 1.2 `internal/config/config.go`

```go
type Config struct {
    App       AppConfig
    Database  DatabaseConfig
    Brands    []BrandConfig
    Providers ProvidersConfig
    Runner    RunnerConfig
    Serve     ServeConfig
}

func Load() (*Config, error)    // reads config.yaml → config.local.yaml → env vars (GEOTRACKER_ prefix)
func (c *Config) Validate() []error
```

Viper loading order (highest priority wins):
1. Env vars prefixed `GEOTRACKER_`
2. `config.local.yaml`
3. `config.yaml`

### 1.3 `cmd/root.go`

- Persistent pre-run: load config, validate, inject into cobra context
- `--log-level` flag overrides config
- Never log API keys or DB passwords

---

## Section 2 — Database schema

File: `internal/db/schema.sql`
Apply idempotently via `Migrate()` on startup using `CREATE TABLE IF NOT EXISTS`.

```sql
CREATE TABLE prompt_sets (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE prompts (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    prompt_set_id  BIGINT UNSIGNED,
    text           TEXT NOT NULL,
    category       VARCHAR(64) NOT NULL,
    active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retired_at     DATETIME,
    notes          TEXT,
    FOREIGN KEY (prompt_set_id) REFERENCES prompt_sets(id)
);

CREATE TABLE runs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    prompt_set_id   BIGINT UNSIGNED,
    started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at     DATETIME,
    prompt_count    INT NOT NULL DEFAULT 0,
    brand_count     INT NOT NULL DEFAULT 0,
    sample_count    INT NOT NULL DEFAULT 1,
    status          VARCHAR(32) NOT NULL DEFAULT 'running',
    FOREIGN KEY (prompt_set_id) REFERENCES prompt_sets(id)
);

CREATE TABLE results (
    id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id                BIGINT UNSIGNED NOT NULL,
    prompt_id             BIGINT UNSIGNED NOT NULL,
    sample_index          TINYINT NOT NULL DEFAULT 0,    -- 0,1,2 for stability analysis
    provider              VARCHAR(32) NOT NULL,
    model_version         VARCHAR(128),
    brand                 VARCHAR(128) NOT NULL,
    raw_response          MEDIUMTEXT,
    brand_mentioned       BOOLEAN NOT NULL DEFAULT FALSE,
    sentiment             VARCHAR(32),
    mention_count         INT NOT NULL DEFAULT 0,
    recommendation_rank   INT,
    competitors_mentioned JSON,
    cited_urls            JSON,
    tokens_input          INT,
    tokens_output         INT,
    latency_ms            INT,
    cost_usd              DECIMAL(10,6),
    extraction_error      TEXT,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id)    REFERENCES runs(id),
    FOREIGN KEY (prompt_id) REFERENCES prompts(id)
);

-- Stability scores — calculated after a run, one row per prompt×provider×brand
CREATE TABLE stability_scores (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id            BIGINT UNSIGNED NOT NULL,
    prompt_id         BIGINT UNSIGNED NOT NULL,
    provider          VARCHAR(32) NOT NULL,
    brand             VARCHAR(128) NOT NULL,
    sample_count      INT NOT NULL,
    mention_rate      DECIMAL(5,2),   -- % of samples where brand_mentioned = true
    rank_variance     DECIMAL(5,2),   -- variance of recommendation_rank across samples
    stability_score   DECIMAL(5,2),   -- 0-100, higher = more consistent
    FOREIGN KEY (run_id)    REFERENCES runs(id),
    FOREIGN KEY (prompt_id) REFERENCES prompts(id)
);

-- Recommendation engine output
CREATE TABLE recommendations (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id          BIGINT UNSIGNED NOT NULL,
    brand           VARCHAR(128) NOT NULL,
    category        VARCHAR(64),
    action          TEXT NOT NULL,
    expected_impact TEXT,
    rationale       TEXT,
    implemented_at  DATETIME,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id) REFERENCES runs(id)
);

-- Indexes
CREATE INDEX idx_results_brand_provider  ON results(brand, provider);
CREATE INDEX idx_results_run_id          ON results(run_id);
CREATE INDEX idx_results_created_at      ON results(created_at);
CREATE INDEX idx_results_category        ON results(run_id, brand);
CREATE INDEX idx_stability_run           ON stability_scores(run_id, brand);
CREATE INDEX idx_recommendations_brand   ON recommendations(brand, status);
```

---

## Section 3 — Provider interface

### 3.1 `internal/providers/provider.go`

```go
type ProbeResponse struct {
    RawText      string
    CitedURLs    []string   // Perplexity populates; others return empty slice
    TokensInput  int
    TokensOutput int
    LatencyMS    int
    ModelVersion string
}

type Provider interface {
    Name()  string
    Probe(ctx context.Context, prompt string) (ProbeResponse, error)
}
```

### 3.2 Provider implementations

One file per provider:
- `anthropic.go` — Claude, Anthropic API shape, `x-api-key` header
- `openai.go` — ChatGPT, OpenAI-compatible shape, `Authorization: Bearer` header
- `perplexity.go` — same as OpenAI-compat + extract `citations` array from response into `CitedURLs`
- `gemini.go` — Gemini via OpenAI-compat endpoint

Rules for all providers:
- `context.WithTimeout` using `timeout_seconds` from config
- Retry on 5xx and timeout only — never retry 4xx
- Probe call has NO system prompt — measures organic AI behavior
- Record `tokens_input`, `tokens_output`, `latency_ms`, `model_version` from response
- Factory: `NewProviders(cfg config.Config) []Provider` returns only enabled providers

---

## Section 4 — Extraction agents

### 4.1 `internal/agent/extractor.go`

```go
type GEOSignal struct {
    BrandMentioned       bool     `json:"brand_mentioned"`
    Sentiment            string   `json:"sentiment"`           // positive|neutral|negative|not_mentioned
    MentionCount         int      `json:"mention_count"`
    RecommendationRank   *int     `json:"recommendation_rank"` // nil if not mentioned
    CompetitorsMentioned []string `json:"competitors_mentioned"`
    CitedURLs            []string `json:"cited_urls"`
    Summary              string   `json:"summary"`
    ReasoningNote        string   `json:"reasoning_note"`
}

func Extract(ctx context.Context, raw, brand string) (GEOSignal, error)
```

- Uses `claude-haiku-4-5-20251001` — cheap and fast
- System prompt: return only valid JSON, no markdown fences
- Merge `CitedURLs` from `ProbeResponse` — Perplexity URLs take priority over extracted
- If JSON parse fails: return error, caller stores `extraction_error` in DB and continues

### 4.2 `internal/agent/explainer.go`

```go
type ExplainRequest struct {
    Brand       string
    PreviousRun RunSummary
    CurrentRun  RunSummary
    TopChanges  []PromptDiff   // prompts where brand_mentioned flipped or rank changed
    NewCompetitors []string    // competitors that appeared this run but not last
    DisappearedCitations []string // URLs cited last run but not this run
}

type Explanation struct {
    Summary    string   // 2-3 sentence plain-English explanation of what changed
    Drivers    []string // bullet list of specific contributing factors
    GeneratedAt time.Time
}

func Explain(ctx context.Context, req ExplainRequest) (Explanation, error)
```

- Uses `claude-sonnet-4-6` — needs reasoning quality, not just extraction
- Called after a run completes if a previous run exists for the same brand
- Stored in DB alongside the run record
- Output example: "Adore Me visibility declined 8% because Skims became the first recommendation for comfort-related prompts (previously Adore Me ranked #1 in 4 of those). Perplexity citation sources also shifted — the ThirdLove fit guide replaced two Adore Me-adjacent Reddit threads."

### 4.3 `internal/agent/recommender.go`

```go
type RecommendationRequest struct {
    Brand           string
    RunID           int64
    OrganicSummary  BrandSummary
    WeakCategories  []CategoryGap      // categories with lowest mention rate
    CitationGaps    []CitationGapEntry // domains cited when brand NOT mentioned
    StabilityScores []PromptStability  // prompts with low stability scores
    TopCompetitors  []Competitor
}

type RecommendationAction struct {
    Category       string
    Action         string
    ExpectedImpact string
    Rationale      string
    Priority       int    // 1 = highest
}

func Recommend(ctx context.Context, req RecommendationRequest) ([]RecommendationAction, error)
```

- Uses `claude-sonnet-4-6`
- Returns exactly 3-5 prioritized, specific GEO actions
- Each action must reference concrete data from the request (specific category, specific competitor, specific citation domain)
- Results written to `recommendations` table
- Called automatically after Explain() at end of each run

---

## Section 5 — Stability scoring

### 5.1 `internal/scoring/visibility.go`

#### Stability score (per prompt × provider × brand)

After all `samples_per_prompt` samples are collected for a prompt:

```go
func CalcStabilityScore(samples []Result) StabilityScore
```

Formula:
```
mention_rate   = mentions / sample_count × 100
rank_variance  = variance of recommendation_rank values (nil = not mentioned = max penalty)
stability_score = mention_rate × (1 - normalized_rank_variance)
```

Range: 0-100. Higher = more consistent.
A prompt where brand is mentioned 3/3 times at rank 1 every time = 100.
A prompt where brand appears 1/3 times at varying ranks = low score.

#### Visibility Score (per brand × run, organic only)

```go
type VisibilityScore struct {
    Brand               string
    RunID               int64
    Score               float64   // 0-100 composite
    MentionRate         float64   // organic only
    FirstRecRate        float64   // % of organic prompts where rank = 1
    SentimentScore      float64   // -1 to 1
    CitationScore       float64   // % of prompts with at least one cited URL
    StabilityScore      float64   // avg stability across organic prompts
    ProviderCoverage    float64   // % of providers that mentioned brand
}

func CalcVisibilityScore(summary BrandSummary, stability []StabilityScore) VisibilityScore
```

Composite formula (adjust weights as needed):
```
Score = (MentionRate × 0.35)
      + (FirstRecRate × 0.25)
      + ((SentimentScore + 1) / 2 × 100 × 0.15)
      + (CitationScore × 0.10)
      + (StabilityScore × 0.10)
      + (ProviderCoverage × 0.05)
```

#### Prompt type filter — CRITICAL

All organic scoring queries MUST filter: `WHERE p.category != 'comparison'`
All comparison scoring queries MUST filter: `WHERE p.category = 'comparison'`
Never aggregate across both types. See Section 6 for split query patterns.

---

## Section 6 — Database queries

### 6.1 `internal/db/results.go` — key query functions

```go
// Organic summary — EXCLUDES comparison category
func (r *ResultsRepo) GetOrganicSummary(brand string, runID int64) (*BrandSummary, error)

// Comparison summary — comparison category ONLY
func (r *ResultsRepo) GetComparisonSummary(brand string, runID int64) (*ComparisonSummary, error)

// Organic trend — EXCLUDES comparison
func (r *ResultsRepo) GetOrganicTrend(brand string, limit int) ([]TrendPoint, error)

// Comparison trend — comparison ONLY, includes win_rate
func (r *ResultsRepo) GetComparisonTrend(brand string, limit int) ([]ComparisonTrendPoint, error)

// Citation gap — domains cited when brand NOT mentioned (organic only)
func (r *ResultsRepo) GetCitationGap(brand string, runID int64) ([]CitationGapEntry, error)

// Stability scores for a run
func (r *ResultsRepo) GetStabilityScores(runID int64, brand string) ([]StabilityScore, error)

// Recommendations for a brand
func (r *ResultsRepo) GetRecommendations(brand string, status string) ([]Recommendation, error)
func (r *ResultsRepo) MarkRecommendationImplemented(id int64) error
```

### 6.2 Citation gap query

This is the most actionable query in the system:

```sql
-- Domains cited when brand was NOT mentioned (organic prompts only)
-- These are the exact pages/domains Adore Me needs to get mentioned on
SELECT
    JSON_UNQUOTE(url_item.value)                     AS cited_url,
    SUBSTRING_INDEX(
        REPLACE(REPLACE(JSON_UNQUOTE(url_item.value),
            'https://',''),'http://',''), '/', 1)    AS domain,
    COUNT(*)                                          AS citation_count
FROM results r
JOIN prompts p ON r.prompt_id = p.id,
JSON_TABLE(r.cited_urls, '$[*]' COLUMNS (value JSON PATH '$')) url_item
WHERE r.run_id          = :run_id
  AND r.brand           = :brand
  AND r.brand_mentioned = FALSE
  AND p.category        != 'comparison'
  AND r.cited_urls      IS NOT NULL
GROUP BY domain
ORDER BY citation_count DESC
LIMIT 20;
```

---

## Section 7 — Runner

### 7.1 `internal/runner/runner.go`

```go
type Job struct {
    Prompt      db.Prompt
    Provider    providers.Provider
    Brand       string
    SampleIndex int   // 0 to (samples_per_prompt - 1)
}

func RunAll(ctx context.Context, cfg RunConfig) ([]Result, error)
```

Job generation:
```
for each prompt in active prompts:
  for each provider in enabled providers:
    for each brand in brands:
      for sample_index in 0..samples_per_prompt-1:
        enqueue Job
```

Total jobs = prompts × providers × brands × samples
Default: 50 × 4 × 2 × 3 = 1,200 jobs per run

Worker pool of `runner.workers` goroutines pulling from buffered job channel.
Rate limiter: token bucket, `rate_limit_per_minute` from config.
Progress: log each completed job at info level.

After all jobs complete:
1. Calculate stability scores per prompt × provider × brand
2. Calculate visibility scores per brand
3. Call Explain() if previous run exists
4. Call Recommend() with full context
5. Store all computed scores and outputs to DB

---

## Section 8 — Commands

### `geo-tracker run`

```
Flags:
  --brands strings      override brands from config
  --providers strings   run only specific providers
  --dry-run             probe but do not write to DB
  --verbose             print raw responses to stdout
  --exit-code           exit non-zero on failure (required for K8s)
```

Stdout on completion:
```
── Run #14 complete ──────────────────────────────────────────
  50 organic prompts · 10 comparison prompts
  4 providers · 2 brands · 3 samples each
  Total jobs: 1,440 · Duration: 4m 12s

── Organic visibility ────────────────────────────────────────
  Brand              Claude  ChatGPT  Perplexity  Gemini  Score
  Adore Me           20%     35%      30%         55%     38.2
  Victoria's Secret  60%     75%      65%         80%     71.4

── Stability (organic) ───────────────────────────────────────
  Brand              Avg stability  Unstable prompts
  Adore Me           64.2           8 prompts < 50
  Victoria's Secret  81.7           3 prompts < 50

── Head-to-head (comparison prompts only) ────────────────────
  Brand              Mentioned  Win rate  Avg rank
  Adore Me           90%        40%       1.8
  Victoria's Secret  100%       60%       1.4

  Note: comparison metrics excluded from Visibility Score.

── Explainability ────────────────────────────────────────────
  Adore Me: visibility +3.2 pts vs last run
  "Adore Me gained traction in gifting prompts (+15%) after
  Reddit r/femalefashionadvice appeared as a new Perplexity
  citation. Fit category remains weakest at 18% mention rate."

── Top recommendations ───────────────────────────────────────
  1. [fit] Publish a bra fit guide — ThirdLove's guide cited
     8× when Adore Me not mentioned. Est. +8 Visibility Score.
  2. [discovery] Engage r/ABraThatFits — cited 5× by Perplexity
     in discovery prompts without Adore Me present.
  3. [fit] Target "small band large cup" content gap — 0%
     mention rate, high-confidence stable gap.
──────────────────────────────────────────────────────────────
```

### `geo-tracker results`

Subcommands: `summary`, `trend`, `prompt`

```
--type string    organic|comparison|all (default: organic)
--brand string   filter by brand
--run-id int     specific run (default: latest)
--format string  table|json|csv (default: table)
--last int       number of runs for trend (default: 10)
```

Default is organic — operator must explicitly opt in to comparison data.

### `geo-tracker prompts`

Subcommands: `list`, `add`, `retire`, `import`

Import format (`prompts/seed.yaml`):
```yaml
prompts:
  - text: "Best lingerie for plus size women?"
    category: purchase
    notes: "Core discovery query"
```

Valid categories: `purchase`, `discovery`, `fit`, `comparison`, `gifting`

### `geo-tracker serve`

Starts chi HTTP server. All routes under `/api/`.
See Section 9 for full endpoint list.

### `geo-tracker config`

Subcommands: `show` (masks API keys), `validate` (pings DB + providers)

---

## Section 9 — API endpoints

```
GET  /api/health

GET  /api/runs
GET  /api/runs/:id/results

GET  /api/brands
GET  /api/brands/:brand/summary              → organic only, includes visibility_score
GET  /api/brands/:brand/trend                → organic only
GET  /api/brands/:brand/comparison-summary   → comparison prompts only
GET  /api/brands/:brand/comparison-trend     → comparison prompts only
GET  /api/brands/:brand/stability            → stability scores for latest run
GET  /api/brands/:brand/citation-gap         → domains cited when brand not mentioned

GET  /api/compare/organic?brands=A,B         → side-by-side organic summary
GET  /api/compare/head-to-head?brands=A,B    → side-by-side comparison summary

GET  /api/competitors?brand=X

GET  /api/recommendations?brand=X&status=pending
POST /api/recommendations/:id/implement      → mark as implemented

GET  /api/prompts
GET  /api/prompts/:id/results

GET  /api/explain/:run_id?brand=X            → explainability text for a run
```

Response conventions:
- All responses: `Content-Type: application/json`
- Error envelope: `{"error": "human message", "code": "SCREAMING_SNAKE"}`
- Organic endpoints include `"prompt_type": "organic"` in response body
- Comparison endpoints include `"prompt_type": "comparison"` in response body
- Pagination: `?page=1&per_page=20`, default 20, max 100
- No auth in v1

### `GET /api/brands/:brand/summary` response shape

```json
{
  "brand": "Adore Me",
  "prompt_type": "organic",
  "run_id": 14,
  "run_at": "2026-06-25T08:14:00Z",
  "visibility_score": 38.2,
  "mention_rate": 35.0,
  "first_rec_rate": 18.0,
  "sentiment_score": 0.42,
  "citation_score": 22.0,
  "stability_score": 64.2,
  "provider_coverage": 100.0,
  "provider_rates": {
    "claude": 20.0, "chatgpt": 35.0, "perplexity": 30.0, "gemini": 55.0
  },
  "top_provider": "gemini",
  "weakest_provider": "claude"
}
```

### `GET /api/brands/:brand/citation-gap` response shape

```json
{
  "brand": "Adore Me",
  "run_id": 14,
  "gaps": [
    { "domain": "thirddlove.com", "citation_count": 8, "category": "fit" },
    { "domain": "reddit.com",     "citation_count": 5, "category": "discovery" },
    { "domain": "allure.com",     "citation_count": 3, "category": "purchase" }
  ]
}
```

### `GET /api/recommendations?brand=X` response shape

```json
{
  "brand": "Adore Me",
  "run_id": 14,
  "recommendations": [
    {
      "id": 1,
      "priority": 1,
      "category": "fit",
      "action": "Publish a comprehensive bra fit guide targeting 'small band large cup' queries",
      "expected_impact": "+8 Visibility Score over 4-6 weeks",
      "rationale": "ThirdLove's fit guide was cited 8 times when Adore Me was not mentioned in fit prompts",
      "status": "pending",
      "implemented_at": null
    }
  ]
}
```

---

## Section 10 — Cost tracking

Record per result row:
- `tokens_input`, `tokens_output` — from provider API response
- `latency_ms` — measure with `time.Since(start)`
- `cost_usd` — calculated per provider using rates from config

Add to config.yaml:
```yaml
cost_rates:
  claude_sonnet:  { input: 3.00, output: 15.00 }   # per million tokens
  claude_haiku:   { input: 1.00, output:  5.00 }
  gpt4o:          { input: 2.50, output: 10.00 }
  gpt4o_mini:     { input: 0.15, output:  0.60 }
  perplexity:     { input: 1.00, output:  1.00 }
  gemini_flash:   { input: 1.50, output:  9.00 }
```

Add to runs table: `total_cost_usd DECIMAL(10,4)` — sum of all result costs for the run.

Expose via `GET /api/runs` — include `total_cost_usd` in run list response.

---

## Section 11 — Demo reliability

### Pre-baked dataset

```go
// cmd/run.go — --demo flag
// Loads a pre-generated result set from demo/baseline.json instead of firing APIs
// Useful when live API calls might fail during presentation
--demo    load results from demo/baseline.json instead of calling providers
```

Generate `demo/baseline.json` by running `geo-tracker run --format json > demo/baseline.json`
before the hackathon. Commit it to the repo.

### Live "run now" endpoint

```
POST /api/runs/live?prompt_id=X&providers=claude,chatgpt
```

Fires a single prompt at specified providers immediately.
Returns results as server-sent events (SSE) — one event per provider as it completes.
Used by the dashboard "Run Now" button for live demo effect.

```go
// SSE event format
data: {"provider":"claude","brand":"Adore Me","brand_mentioned":true,"sentiment":"positive"}
data: {"provider":"chatgpt","brand":"Adore Me","brand_mentioned":false,"sentiment":"not_mentioned"}
```

---

## Section 12 — Kubernetes

No internal scheduler. Scheduling is handled by Kubernetes CronJob.

The app must:
- Exit 0 on success
- Exit non-zero on failure (use `--exit-code` flag)
- Read all config from env vars — no config files in Docker image

Dockerfile: multi-stage build, `golang:1.23-alpine` → `alpine:3.19`, binary only.

K8s manifest pattern:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: lighthouse
spec:
  schedule: "0 8 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: lighthouse
              image: adoreme/geo-tracker:latest
              command: ["geo-tracker", "run", "--exit-code"]
              envFrom:
                - secretRef:
                    name: lighthouse-secrets
          restartPolicy: OnFailure
```

---

## Section 13 — Implementation order

Work tasks in this sequence:

1. Config struct + Load() + Validate()
2. DB connection + schema.sql + Migrate()
3. DB repos: prompts.go + results.go (all query functions)
4. Provider interface + all 4 provider implementations
5. Extraction agent (extractor.go)
6. Runner — worker pool, job fan-out, result collection
7. Stability scoring (scoring/visibility.go)
8. Visibility Score calculation
9. Explainability agent (explainer.go)
10. Recommendation agent (recommender.go)
11. `run` command — wire everything together, stdout summary
12. `results` command — subcommands, --type flag
13. `prompts` command — subcommands, import
14. API server — chi router, all handlers
15. `serve` command
16. `config` command
17. Live run SSE endpoint
18. Demo dataset generation
19. Update CLAUDE.md

---

## Section 14 — Testing requirements

- Unit test `config.Validate()` — missing required fields
- Unit test `scoring.CalcStabilityScore()` — various sample patterns
- Unit test `scoring.CalcVisibilityScore()` — weighted formula
- Unit test `db.GetOrganicSummary()` — assert comparison rows excluded
- Unit test `db.GetComparisonSummary()` — assert only comparison rows included
- Unit test `db.GetCitationGap()` — assert brand_mentioned = false filter
- Provider tests — mock HTTP server via `net/http/httptest`
- Runner test — mock providers, assert job count = prompts × providers × brands × samples
- API handler tests — `net/http/httptest`, assert `prompt_type` field in responses
- Agent tests — mock Anthropic API, assert JSON parsing handles malformed response

---

## Critical rules (never violate)

- Probe calls have NO system prompt — measures organic AI behavior
- `comparison` category is NEVER included in organic metrics
- API keys and passwords are NEVER logged at any level
- All DB queries use sqlx named parameters
- No global variables — inject all dependencies
- Every result row has a valid run_id before being written
- `--exit-code` flag must cause non-zero exit on any run failure

---

*Project Lighthouse · Backend · Adore Me Tech Hackathon · June 2026*

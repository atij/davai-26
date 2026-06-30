# CLAUDE.md — GEO Tracker

This file tells you everything you need to know to work in this codebase.
Read it fully before writing any code.

---

## What this app does

GEO Tracker is a Go CLI app that probes 4 AI providers (Claude, ChatGPT, Perplexity, Gemini)
with curated prompts, extracts brand visibility signals, stores results in MySQL, and serves
a JSON API for a React dashboard. It runs as a Kubernetes CronJob — no internal scheduler.

---

## Commands

```
geo-tracker run       # fire all active prompts at all enabled providers
geo-tracker results   # query and display results (subcommands: summary, trend, prompt)
geo-tracker prompts   # manage prompt library (subcommands: list, add, retire, import)
geo-tracker serve     # expose results as JSON API for the dashboard
geo-tracker config    # validate and show resolved config (subcommands: show, validate)
```

There is no `schedule` command. Scheduling is handled externally by Kubernetes CronJob.

---

## Repository structure

```
geo-tracker/
├── cmd/
│   ├── root.go           # root Cobra command, Viper bootstrap, logger init
│   ├── run.go            # run command
│   ├── results.go        # results command + subcommands
│   ├── prompts.go        # prompts command + subcommands
│   ├── serve.go          # serve command
│   └── config.go         # config command + subcommands
├── internal/
│   ├── config/
│   │   └── config.go     # Config struct, Load(), Validate()
│   ├── db/
│   │   ├── db.go         # sqlx connection, Migrate()
│   │   ├── schema.sql    # table definitions (source of truth)
│   │   ├── prompts.go    # prompt repository
│   │   └── results.go    # results repository
│   ├── providers/
│   │   ├── provider.go   # Provider interface + ProbeResponse struct
│   │   ├── anthropic.go  # Claude
│   │   ├── openai.go     # ChatGPT
│   │   ├── perplexity.go # Perplexity (+ cited URL extraction)
│   │   └── gemini.go     # Gemini via OpenAI-compat endpoint
│   ├── agent/
│   │   └── extractor.go  # Haiku extraction agent → GEOSignal
│   ├── runner/
│   │   └── runner.go     # worker pool, async fan-out, result collector
│   └── api/
│       ├── server.go     # chi router, middleware
│       ├── handlers.go   # HTTP handlers
│       └── dto.go        # request/response types
├── prompts/
│   └── seed.yaml         # 50 curated seed prompts
├── config.yaml           # default config, committed, no secrets
├── config.local.yaml     # gitignored, local overrides and API keys
├── .gitignore
├── go.mod
├── go.sum
├── Dockerfile
└── main.go
```

---

## Tech stack

| Concern | Library |
|---|---|
| CLI | `github.com/spf13/cobra` |
| Config | `github.com/spf13/viper` |
| Database | `github.com/jmoiron/sqlx` + `github.com/go-sql-driver/mysql` |
| HTTP router | `github.com/go-chi/chi/v5` |
| Logging | `go.uber.org/zap` |
| Testing | standard `testing` package + `github.com/stretchr/testify` |

Do not add dependencies without a clear reason. Prefer stdlib over new deps for simple tasks.

---

## Configuration

Config is loaded in this priority order (highest wins):

1. Environment variables prefixed `GEOTRACKER_` (e.g. `GEOTRACKER_DATABASE_PASSWORD`)
2. `config.local.yaml` (gitignored — local dev and secrets)
3. `config.yaml` (committed — defaults and structure)

In Kubernetes, secrets are injected as env vars from a K8s Secret object.
Never read API keys or DB passwords from anywhere other than the Viper-resolved config.

### config.yaml structure

```yaml
app:
  name: geo-tracker
  log_level: info           # debug | info | warn | error

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
  retry_attempts: 2
  retry_delay_seconds: 5
  rate_limit_per_minute: 60

serve:
  host: "0.0.0.0"
  port: 8080
  cors_origins:
    - "http://localhost:3000"
```

---

## Database

### Rules
- All schema changes go in `internal/db/schema.sql` first — this is the source of truth
- `Migrate()` in `db.go` applies schema idempotently using `CREATE TABLE IF NOT EXISTS`
- Never use raw `database/sql` — always use `sqlx`
- All queries use named parameters: `db.NamedExec`, `db.NamedQuery`, `sqlx.Named`
- Never use `SELECT *` — always name columns explicitly
- Soft-delete only — prompts are retired (`active=false`, `retired_at=now`), never hard-deleted
- Every `results` row must have a valid `run_id` — always create the run record first

### Schema overview

```sql
-- prompts: the library of questions fired at providers
-- runs: one record per execution of geo-tracker run
-- results: one record per prompt × provider × brand × run
```

Full definitions live in `internal/db/schema.sql`.

---

## Provider interface

Every provider must implement this interface. No concrete provider type is used
outside of `internal/providers/`.

```go
type ProbeResponse struct {
    RawText   string
    CitedURLs []string  // Perplexity populates this; others return empty slice
}

type Provider interface {
    Name() string
    Probe(ctx context.Context, prompt string) (ProbeResponse, error)
}
```

### Rules
- Each provider gets its own file — one file per provider, no shared logic except the interface
- Always use `context.WithTimeout` with the provider's configured `timeout_seconds`
- Retry on transient errors only (5xx, timeout) — do not retry 4xx
- Never log raw API responses at info level — only at debug level
- Perplexity: extract the `citations` array from the API response into `CitedURLs`
- The extraction agent (Haiku) is called by `agent/extractor.go`, not by providers

### Two-call design (important)
The probe call has **no system prompt** — this measures organic AI behavior.
The extraction call is a separate Claude Haiku call with a strict system prompt.
Never combine them into one call.

**Note on Search/Grounding:** "No system prompt" on the probe call refers to *not directing the model's search behavior* — it does NOT mean tools/search must be disabled. Native server-side web search/grounding tools are enabled for all providers (Anthropic, OpenAI Responses API, Gemini Native API, Perplexity) to mirror chat UI behavior and ensure grounded responses with citations.

---

## Extraction agent

`internal/agent/extractor.go` takes a raw provider response and returns a structured signal.

```go
type GEOSignal struct {
    BrandMentioned       bool     `json:"brand_mentioned"`
    Sentiment            string   `json:"sentiment"`         // positive|neutral|negative|not_mentioned
    MentionCount         int      `json:"mention_count"`
    RecommendationRank   *int     `json:"recommendation_rank"` // nil if not mentioned
    CompetitorsMentioned []string `json:"competitors_mentioned"`
    CitedURLs            []string `json:"cited_urls"`
}
```

### Rules
- Always use `claude-haiku-4-5-20251001` for extraction — cheap and fast
- System prompt must instruct the model to return only valid JSON, no markdown fences
- Parse response with `json.Unmarshal` — if it fails, return an error, do not guess
- Merge `CitedURLs` from `ProbeResponse` into the signal (Perplexity URLs take priority)
- Log extraction errors but do not fail the whole run — store `extraction_error` in DB

---

## Runner

`internal/runner/runner.go` fans out all jobs (prompt × provider × brand) across a
worker pool and collects results.

### Rules
- Worker count comes from `runner.workers` config — never hardcode
- Use a buffered job channel and a fixed pool of goroutines — no unbounded goroutine spawning
- Respect `rate_limit_per_minute` using a token bucket or `time.Ticker`
- Each job: `provider.Probe()` → `agent.Extract()` → return result
- Collect results via a results channel — use a `sync.WaitGroup` to know when all jobs are done
- Progress: log each completed job at info level with provider, brand, prompt_id, brand_mentioned

---

## Runner (run command)

### Behavior
1. Load active prompts from DB
2. Create a `runs` record with `status = running`
3. Call `runner.RunAll()`
4. Write all results to DB
5. Update `runs` record to `status = done` (or `failed` on error)
6. Print summary table to stdout

### Flags
```
--brands strings      override brands from config
--providers strings   run only specific providers
--dry-run             probe but do not write to DB
--verbose             print raw responses to stdout
--exit-code           exit non-zero on any probe or extraction error (for K8s job detection)
```

### Kubernetes notes
- The app must exit 0 on success and non-zero on failure
- `--exit-code` flag ensures K8s CronJob can detect and retry failed runs
- All config comes from env vars injected by K8s Secret — no config files in the image

---

## API server (serve command)

Router: `chi`. All routes under `/api/`.

### Endpoints

```
GET /api/health                          → {"status":"ok","db":"ok"}
GET /api/runs                            → paginated list of runs
GET /api/runs/:id/results                → all results for a run
GET /api/brands                          → list of tracked brands
GET /api/brands/:brand/summary           → latest mention rate + sentiment
GET /api/brands/:brand/trend             → mention rate over time (?runs=N)
GET /api/compare?brands=A,B             → side-by-side brand comparison
GET /api/competitors?brand=X            → top competitors for a brand
GET /api/prompts                         → active prompt list
GET /api/prompts/:id/results             → results per prompt across providers
```

### Rules
- All handlers return `application/json`
- Error envelope: `{"error": "human message", "code": "SCREAMING_SNAKE_CASE"}`
- 404 for unknown routes, 405 for wrong method — chi handles these automatically
- CORS origins come from `serve.cors_origins` config — do not hardcode
- No auth in v1 — this is an internal tool on a private network
- Pagination: `?page=1&per_page=20` — default 20, max 100

---

## Logging

- Use `zap.Logger` everywhere — no `fmt.Println` in business logic, no `log.Print`
- Inject logger via struct fields and function parameters — no global logger variable
- Standard fields on every log line: `run_id`, `provider`, `brand`, `prompt_id` (where applicable)
- API keys and DB passwords must never appear in log output at any level
- Slow provider calls (> 10s) log at warn level with duration field
- `--log-level` flag on root command overrides `app.log_level` from config

---

## Error handling

- Return errors up the call stack — do not swallow them silently
- Wrap errors with context: `fmt.Errorf("probe claude: %w", err)`
- Extraction errors are non-fatal — log + store `extraction_error` in DB, continue run
- Provider probe errors are non-fatal per job — log + mark result as errored, continue run
- A run fails (exit non-zero) only if no results were successfully stored at all
- DB connection failure on startup is fatal — log and exit 1

---

## Testing

- Table-driven tests for: `internal/config`, `internal/agent`, `internal/db` repositories
- Provider implementations: test with a mock HTTP server (`net/http/httptest`)
- Runner: test with mock providers that return fixed responses
- API handlers: test with `net/http/httptest` and real chi router
- Do not test `cmd/` directly — test the internal packages they call
- Test files live next to the code they test: `extractor_test.go` beside `extractor.go`
- No integration tests that require a real DB or real API keys in CI

---

## Code style

- `gofmt` always — no exceptions
- `golint` and `go vet` must pass before commit
- No exported types in `internal/` packages except those that cross package boundaries
- Interfaces defined where they are used, not where they are implemented
- No init() functions
- No global variables — inject everything
- Short variable names for short scopes, descriptive names for package-level identifiers
- Comment all exported types and functions — one sentence minimum

---

## Prompt library

Seed prompts live in `prompts/seed.yaml`. Import with:

```bash
geo-tracker prompts import prompts/seed.yaml
```

### YAML format

```yaml
prompts:
  - text: "Best lingerie for plus size women?"
    category: purchase
    notes: "Core discovery query"
  - text: "Compare Adore Me vs Victoria's Secret"
    category: comparison
    notes: "Direct brand comparison"
```

### Categories
- `purchase` — buy intent queries (15 prompts)
- `discovery` — brand discovery queries (10 prompts)
- `fit` — fit and sizing advice (10 prompts)
- `comparison` — brand comparison queries (10 prompts)
- `gifting` — gift recommendation queries (5 prompts)

### Rules for prompts
- Write prompts exactly as a real customer would type them to an AI chatbot
- No brand names in the prompt text unless the category is `comparison`
- Cover a range of customer intents, body types, price sensitivities
- Retiring a prompt does not delete historical results — `run_id` links are preserved

---

## Running locally

```bash
# 1. Copy and fill in secrets
cp config.yaml config.local.yaml
# edit config.local.yaml with API keys and DB credentials

# 2. Start MySQL (Docker)
docker run -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=geo_tracker mysql:8

# 3. Validate config and apply schema
go run . config validate

# 4. Seed prompts
go run . prompts import prompts/seed.yaml

# 5. Run once
go run . run --verbose

# 6. Check results
go run . results summary

# 7. Start API server
go run . serve
```

---

## Building and deploying

```bash
# Build binary
go build -o geo-tracker .

# Build Docker image
docker build -t adoreme/geo-tracker:latest .

# Kubernetes CronJob fires:
geo-tracker run --exit-code
```

The Dockerfile should produce a minimal image — use a multi-stage build with
`golang:1.23-alpine` to build and `alpine:3.19` as the final stage.
Copy only the binary into the final image. No config files — all config via env vars in K8s.

---

## What not to build

- No internal scheduler or cron — Kubernetes handles this
- No authentication — internal tool on private network
- No frontend — the React dashboard is a separate repo that calls `geo-tracker serve`
- No migrations framework — `Migrate()` with `CREATE TABLE IF NOT EXISTS` is enough for v1
- No message queue — the runner worker pool is sufficient for 50 prompts × 4 providers

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

---

*GEO Tracker · Adore Me Tech · June 2026*

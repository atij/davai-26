# CLAUDE.md — GEO Tracker (geo-tracker-backend)

This file tells you everything you need to know to work in this codebase.
Read it fully before writing any code.

> **This file was rewritten on 2026-06-30 to match the actual repository
> state after the ADK migration and the provider web-search fix.** The
> original hackathon-era CLAUDE.md described a hand-rolled goroutine runner
> that no longer exists. If you find code that contradicts this file, trust
> the code and flag the doc as stale again — but as of this rewrite, this
> file has been verified line-by-line against the live repo.

---

## What this app does

GEO Tracker is a Go CLI app that probes AI providers (Claude, ChatGPT,
Perplexity, Gemini — currently only ChatGPT and Gemini are enabled, see
"Current provider state" below) with curated prompts, extracts brand
visibility signals via an LLM agent layer, stores results in MySQL, and
serves a JSON API for a React dashboard. It runs as a Kubernetes CronJob —
no internal scheduler.

---

## Commands

```
geo-tracker run all            # full pipeline: probe → intelligence → insight
geo-tracker run ingest         # Phase 1 only — probe + store raw results
geo-tracker run intelligence   # Phase 2 only — extraction + stability scoring
geo-tracker run insight        # Phase 3 only — explainer + recommender
geo-tracker run list           # list recent pipeline runs
geo-tracker results            # query and display results (subcommands: summary, trend, prompt)
geo-tracker prompts            # manage prompt library (subcommands: list, add, retire, import)
geo-tracker serve              # expose results as JSON API for the dashboard
geo-tracker config             # validate and show resolved config (subcommands: show, validate)
```

There is no `schedule` command. Scheduling is handled externally by
Kubernetes CronJob (`geo-tracker run all --exit-code`).

---

## Repository structure (current)

```
geo-tracker-backend/
├── cmd/
│   ├── root.go
│   ├── run.go             # run all/ingest/intelligence/insight/list — wires up adk.Pipeline
│   ├── results.go
│   ├── prompts.go
│   ├── serve.go            # constructs + injects StrategyAgent (not yet exposed via API — see below)
│   └── config.go
├── internal/
│   ├── config/
│   │   └── config.go       # Config struct incl. ADKConfig, CostRatesConfig
│   ├── db/
│   │   ├── db.go
│   │   ├── schema.sql       # source of truth — includes agent_sessions, run_traces
│   │   ├── prompts.go
│   │   └── results.go       # results repo + ADK tool query functions
│   ├── providers/
│   │   ├── provider.go      # Provider interface + ProbeResponse + ResolveRedirects()
│   │   ├── anthropic.go     # Claude — native web_search_20250305 tool
│   │   ├── openai.go        # ChatGPT (Responses API, native web_search tool)
│   │   │                    #   + Perplexity + Gemini implementations live in this
│   │   │                    #   same file via openAIProvider{name: ...} branching —
│   │   │                    #   NOT one-file-per-provider despite the name. See note below.
│   │   └── factory.go       # NewProviders(), Extract() helper used by agent layer
│   ├── agent/
│   │   └── extractor.go     # GEOSignal, MultiBrandSignal, Extract(), ExtractMultiBrand()
│   │                        # NOTE: extraction LLM call now goes through internal/adk
│   │                        # model factory, not a hardcoded Claude Haiku call — see
│   │                        # "Extraction" section below.
│   ├── adk/
│   │   ├── pipeline.go      # Pipeline.Run() — the 3-phase orchestrator (THE runner)
│   │   ├── agents.go        # ExplainerAgent, RecommenderAgent, StrategyAgent
│   │   ├── model.go         # NewADKModel() — provider-agnostic model factory
│   │   ├── memory.go        # MySQLSessionStore (Strategy Agent session persistence)
│   │   └── tools.go         # ToolSet — DB-query tools for Strategy Agent (not yet wired to an API route)
│   ├── scoring/
│   │   └── visibility.go    # pure math — stability score, visibility score formula
│   └── api/
│       ├── server.go
│       ├── handlers.go
│       └── dto.go
├── prompts/
│   └── seed.yaml
├── config.yaml               # committed, no secrets
├── config.local.yaml          # gitignored — local overrides and API keys
├── .gitignore
├── go.mod
├── go.sum
├── Dockerfile
└── main.go
```

**Note on `internal/providers/openai.go`:** despite the filename, this file
contains the OpenAI, Perplexity, AND Gemini probe implementations, branching
internally on `p.name`. This deviates from the original "one file per
provider, no shared logic" rule. Whether to split this back into separate
files is an open cleanup item — not done as part of this rewrite, flagged
here so nobody assumes `gemini.go`/`perplexity.go` exist when they don't.

`internal/runner/runner.go` has been **deleted**. It was the pre-ADK,
single-brand-per-job implementation, superseded by `internal/adk/pipeline.go`.
If you see references to it anywhere (old branches, stale docs), that code
no longer exists in `main`.

---

## Current provider state (as of 2026-06-30, testing phase)

| Provider | Enabled | Reason |
|---|---|---|
| ChatGPT | yes | |
| Gemini | yes | |
| Claude | no | waiting on a valid API key |
| Perplexity | no | blocked by Zscaler network policy |

This is a temporary testing configuration, not a permanent architecture
decision. Code must continue to support all 4 providers being enabled —
do not hardcode assumptions that only 2 providers are active.

---

## Probe models (config.yaml `providers.*.probe_model`)

| Provider | Current value | Notes |
|---|---|---|
| `claude` | `claude-sonnet-4-6` | matches Claude.ai default |
| `chatgpt` | `gpt-4o` | stale — ChatGPT's chat-UI default moved to GPT-5.5 Instant; not yet updated here, not urgent (gpt-4o still functions) |
| `perplexity` | `llama-3.1-sonar-large-128k-online` | not yet re-verified against current perplexity.ai default |
| `gemini` | `gemini-flash-latest` | current — auto-tracking alias, fixed after `gemini-2.0-flash` was shut down by Google on 2026-06-01 |

---

## Provider interface

```go
type ProbeResponse struct {
    RawText      string
    CitedURLs    []string
    TokensInput  int
    TokensOutput int
    LatencyMS    int
    ModelVersion string
}

type Provider interface {
    Name() string
    Probe(ctx context.Context, prompt string) (ProbeResponse, error)
}
```

### Rules
- `context.WithTimeout` using the provider's configured `timeout_seconds`
- Retry on transient errors only (5xx, timeout) — do not retry 4xx
- Never log raw API responses at info level — only at debug level
- The extraction agent is called by `agent/extractor.go` via the ADK model
  factory, not by providers directly

### Two-call design (important)
The probe call has **no system prompt** — this measures organic AI behavior.
The extraction call is a separate LLM call with a strict system prompt.
Never combine them into one call.

**Note on search/grounding:** "no system prompt" refers to *not directing
the model's search behavior* — it does NOT mean tools/search must be
disabled. Native server-side web search/grounding tools are enabled for all
4 providers to mirror real chat-UI behavior and ensure grounded responses
with citations:

| Provider | Search mechanism |
|---|---|
| Claude | native `web_search_20250305` tool, server-executed, single turn |
| ChatGPT | Responses API (`/v1/responses`) native `web_search` tool — not `/v1/chat/completions`, which does not support native search |
| Gemini | native `generateContent` endpoint with `google_search` grounding tool; grounding redirect URLs are resolved via `providers.ResolveRedirects()` |
| Perplexity | searches by default (Sonar models) — no tool config needed |

Do not reintroduce custom client-side function-tool definitions for search
(e.g. a hand-declared `google_search` function tool) — these require a
second round-trip to execute that the pipeline does not perform, and will
silently produce ungrounded, uncited responses.

---

## Extraction (multi-brand)

`internal/agent/extractor.go` takes a raw provider response and returns a
multi-brand structured signal in a single LLM call, not one call per
brand:

```go
type GEOSignal struct {
    BrandMentioned       bool
    Sentiment            string   // positive|neutral|negative|not_mentioned
    MentionCount         int
    RecommendationRank   *int
    CompetitorsMentioned []string
    CitedURLs            []string
    Summary              string
    ReasoningNote        string
}

type MultiBrandSignal map[string]GEOSignal // key: brand name

func ExtractMultiBrand(ctx, cfg, providerType, rawText string, brands []string) (MultiBrandSignal, error)
func Extract(ctx, cfg, providerType, rawText, brand string) (GEOSignal, error) // still used for comparison-category prompts
```

### Why multi-brand
Organic prompts (purchase/discovery/fit/gifting) never name a brand by
design — the prompt text is identical for every brand. Probing the same
prompt once per brand wasted API calls and produced non-comparable samples
(brand A's mention rate measured against a different raw response than
brand B's). The pipeline now probes once per (prompt, provider, sample)
for organic prompts, and extracts signals for all configured brands
from that single response.

Comparison-category prompts are unaffected — they name a brand explicitly
in the prompt text, so they still require one probe per (prompt, provider,
brand, sample), and use the single-brand `Extract()` function.

### Extraction model — now lives under `adk`, not per-provider config
The `extract_model` field that used to live on each `providers.*` config
block is gone. Extraction is an ADK agent operation now:

- Config key: `adk.extractor_model`
- Default: `gemini-2.5-flash` (current `adk.provider: gemini` default)
- The extraction call goes through the same `NewADKModel()` factory as
  Explainer/Recommender/Strategy — there is no hardcoded model string
  for extraction anywhere in `internal/adk/pipeline.go`. If you find one,
  it's a regression — file an issue.
- `gemini-2.0-flash` must never be referenced anywhere as a valid model —
  Google shut it down 2026-06-01.

### Known issue — substring heuristic override (pending decision)
`internal/adk/pipeline.go`'s organic extraction path currently includes a
post-extraction heuristic: if the LLM extractor says `brand_mentioned:
false` but the brand name appears as a literal case-insensitive substring
anywhere in the raw response text, the code force-flips
`brand_mentioned: true`. This is unreviewed and undocumented in its
current form — it can produce false positives (e.g. "I have no
information about Adore Me" would count as a mention). A decision is
pending on whether to keep it (and make it visible via a
`mention_source: llm | heuristic_override` field) or remove it in favor of
improving the extraction prompt directly. Do not assume this is settled
behavior — check with the team before relying on it or removing it.

---

## ADK agent layer

Pipeline orchestration lives in `internal/adk/pipeline.go`.
Do not add sequencing logic to `cmd/run.go` — all phase ordering is in
`Pipeline.Run()`.

### Provider selection

```yaml
adk:
  provider: "gemini"            # "gemini" | "anthropic"
  api_key: ""                   # GEOTRACKER_ADK_API_KEY env var — single key for all ADK agents
  strategy_model: "gemini-2.5-flash"
  explainer_model: "gemini-2.5-flash"
  recommender_model: "gemini-2.5-flash"
  extractor_model: "gemini-2.5-flash"   # see "Extraction" section above
  session_ttl_days: 30
```

All four ADK agent models currently share one `adk.provider`/`adk.api_key`
— there is no per-agent provider override yet. When the Claude API key
becomes available, mixed-provider experiments (e.g. Gemini for extraction,
Claude for explainer) are planned future work, not current behavior.

`NewADKModel()` in `model.go` is the single switch point for provider
selection — do not add provider-switch logic anywhere else.

### Three pipeline phases

```
Phase 1 — PROBE (parallel goroutine pool)
  Organic prompts:    (prompt, provider, sample) -> provider.Probe() -> ExtractMultiBrand() -> N brand result rows
  Comparison prompts: (prompt, provider, brand, sample) -> provider.Probe() -> Extract() -> 1 result row

Phase 2 — INTELLIGENCE (parallel via errgroup)
  A: stability scoring (per prompt x provider x brand, across samples)
  B: visibility score calculation

Phase 3 — INSIGHT (parallel via errgroup, per brand)
  A: ExplainerAgent.Explain()
  B: RecommenderAgent.Recommend()
```

Each phase writes a row to `run_traces` on start/finish — surfaced (or
intended to be surfaced) in the dashboard as a live agent execution
timeline (`/runs/:id`).

### Agent types

| Agent | File | Model config key | Status |
|---|---|---|---|
| ExplainerAgent | `internal/adk/agents.go` | `adk.explainer_model` | shipped, part of Phase 3 |
| RecommenderAgent | `internal/adk/agents.go` | `adk.recommender_model` | shipped, part of Phase 3 |
| StrategyAgent | `internal/adk/agents.go` | `adk.strategy_model` | built but not wired to a live API route yet — `cmd/serve.go` constructs it, but `POST /api/strategy/chat` (SSE) is not yet implemented in `internal/api/handlers.go`. Treat as in-progress, not shipped. |

### Strategy Agent (in progress, not yet exposed)
Conversational agent with 6 read-mostly DB tools (`internal/adk/tools.go`)
and persistent session memory via `MySQLSessionStore` (`agent_sessions`
table). Tools: `get_visibility_trend`, `get_citation_gaps`,
`get_stability_scores`, `get_competitor_share`, `search_recommendations`
(read-only), `mark_recommendation_done` (the one write). This is the
planned demo centerpiece but is not yet reachable from the frontend or API
— do not document or demo it as a working feature until
`POST /api/strategy/chat` exists.

### Rules
- Never call ADK agents from `cmd/` directly — always through `Pipeline` or
  a constructed agent passed in
- Strategy Agent session IDs are brand-scoped: format `"{brand}-{uuid}"`
- Tool functions in `tools.go` are read-only except `mark_recommendation_done`
- Run traces are written by `Pipeline.traceStart/traceEnd` — never write
  them manually
- `adk.api_key` never appears in logs — same rule as all other API keys
  and must never appear in chat, config samples shared outside the
  secrets manager, or committed config — if it ever does, rotate it
  immediately

---

## Known frontend/backend serialization mismatch (open issue)

`geo-tracker-frontend/components/runs/RunTraceTimeline.tsx` defensively
reads both snake_case and PascalCase field names for trace data
(`trace.duration_ms ?? trace.DurationMS`, etc.), even though the backend
`RunTrace` struct has correct `json:"duration_ms"`-style tags. This means
some code path producing trace data for the frontend (likely an SSE stream
or a different in-memory representation than the tagged `RunTrace` struct)
is NOT going through those JSON tags. The frontend workaround masks the
symptom but the root cause — wherever trace data is serialized outside the
`RunTrace` struct's tags — has not been found/fixed yet. Any new component
consuming `run_traces` data should be aware it may need the same dual-key
workaround until the root cause is fixed.

---

## Known "no data" ambiguity (open issue)

`ResultRepo.GetBrandSummary()` returns a zeroed/skeleton `BrandSummary`
(all rates 0.0, no error) when a brand has no results yet for the latest
run, rather than a distinct "no data" signal (404, null, or an explicit
flag). The frontend currently cannot distinguish "this brand genuinely
scored 0% mention rate" from "we have no data for this brand in this run
yet" — both render identically. Not yet fixed; flagged for whoever picks
up dashboard "no data" complaints next.

---

## Cost tracking — model-keyed rate map (pending implementation)

`cost_rates` in config currently keys rates by hand-picked labels
(`claude_sonnet`, `gpt4o`, `gemini_flash`, etc.) and `calculateCost()` does
fuzzy provider-name/substring matching to pick a rate. This breaks silently
whenever a `probe_model` changes (e.g. swapping `gemini-2.0-flash` ->
`gemini-flash-latest` kept applying the old `gemini_flash` rate with no
signal that pricing might now be wrong).

Decided direction (not yet implemented): `cost_rates` should be keyed
by the literal model version string (`result.ModelVersion`, as recorded
from the actual API response) instead of a hand-picked label, looked up via
a flat map in `calculateCost()`. If a model string has no entry, log a
warning (not an error — the run continues) and the row's cost should
not silently default to another model's rate. Prices are maintained
manually based on provider pricing pages — no auto-fetching.

Also pending: the same cost-double-counting bug affecting organic rows
(each probe's full cost currently gets attributed to every brand's result
row derived from it, inflating `TotalCostUSD` by roughly brand-count times
the real value). Needs a fix where probe cost is attributed once per probe,
not once per derived result row.

---

## Known config drift to watch for
- `config.ProviderConfig` struct still declares `ExtractModel` — this field
  is dead, no longer read by `pipeline.go`, and absent from
  `config.yaml`. Remove it from the struct when convenient; until then,
  do not add new code that reads `cfg.Providers.*.ExtractModel`.

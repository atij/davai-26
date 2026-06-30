# Task: Extraction must use ADK config exclusively

## Problem (confirmed root cause candidate)
`internal/adk/pipeline.go` hardcodes extraction to Claude in two places:

```go
extractCfg := p.cfg.Providers.Claude
extractCfg.ExtractModel = "claude-haiku-4-5-20251001"
multiSignal, err := agent.ExtractMultiBrand(ctx, extractCfg, "claude", probeRes.RawText, brands)
```

— in `executeOrganicJob` (organic path) and again in `executeProbeJob`
(comparison path, calling `agent.Extract` instead of `ExtractMultiBrand`).

This contradicts the agreed design: extraction should run entirely through
the ADK layer (`adk.provider` / `adk.api_key` / `adk.extractor_model`), not
through `p.cfg.Providers.Claude`. Since Claude is currently disabled
(`enabled: false`, no valid API key yet), every extraction call is likely
failing silently against the Anthropic API — independent of which provider
actually did the probing — which would explain near-zero mention rates
despite confirmed-working web search and real brand mentions in raw
responses.

## Fix

### 1. `internal/agent/extractor.go`
- [ ] Change `ExtractMultiBrand` and `Extract` signatures to accept an ADK
  model config instead of `config.ProviderConfig` — i.e. whatever shape
  `internal/adk/model.go`'s `NewADKModel()` / the ADK agent constructors
  already use for Explainer/Recommender/Strategy. Reuse that exact
  mechanism rather than inventing a parallel one.
- [ ] Route the actual LLM call through `NewADKModel()` (or call through an
  `ExtractorAgent` type in `internal/adk/agents.go`, consistent with how
  `ExplainerAgent`/`RecommenderAgent` are structured) so extraction behaves
  identically to the other three ADK agents — same provider switch, same
  api key, same logging/error conventions.
- [ ] Remove any remaining references to `config.ProviderConfig.ExtractModel`
  in this file.

### 2. `internal/adk/pipeline.go`
- [ ] `executeOrganicJob`: replace
  ```go
  extractCfg := p.cfg.Providers.Claude
  extractCfg.ExtractModel = "claude-haiku-4-5-20251001"
  multiSignal, err := agent.ExtractMultiBrand(ctx, extractCfg, "claude", probeRes.RawText, brands)
  ```
  with a call using `p.cfg.ADK` (provider, api_key, extractor_model) — no
  reference to `p.cfg.Providers.Claude` anywhere in this function.
- [ ] `executeProbeJob` (comparison path): same fix for the
  `agent.Extract(ctx, extractCfg, "claude", probeRes.RawText, job.Brand)`
  call.
- [ ] Grep `internal/adk/pipeline.go` for any other occurrence of
  `p.cfg.Providers.Claude` used for extraction purposes and fix those too —
  there were two call sites found in this review, confirm there isn't a
  third.

### 3. `internal/config/config.go`
- [ ] Add `ExtractorModel string` field to `ADKConfig` (mapstructure key
  `extractor_model`), matching `StrategyModel`/`ExplainerModel`/
  `RecommenderModel`.
- [ ] Remove `ExtractModel` from `ProviderConfig` struct — it's fully dead
  once this task lands (per earlier decision: "all other will be removed").

### 4. `config.yaml`
- [ ] Add `adk.extractor_model: "gemini-2.5-flash"` alongside the other
  three ADK model keys.
- [ ] Remove `extract_model` if it still appears anywhere under
  `providers.*` (already absent from the config pasted earlier in this
  conversation, but confirm `config.yaml` — the committed default file —
  doesn't still have it even if `config.local.yaml` doesn't).

### 5. Verification
- [ ] Run `geo-tracker run all` (or `run ingest` + `run intelligence`) with
  Claude still disabled and confirm extraction succeeds — check that
  `ExtractionError` is empty/null on result rows instead of containing
  `"multi-extract: ..."` or `"extract (claude): ..."` errors.
- [ ] Confirm `brand_mentioned` rates now reflect what's actually in
  `raw_response` — spot check a few result rows where the raw text clearly
  contains "Adore Me" or "Victoria's Secret" and confirm `brand_mentioned`
  is now `true` without needing the substring heuristic override to fire
  (i.e. the LLM extractor itself is now succeeding, not just the override
  compensating for a broken call).
- [ ] Re-run the small 3-prompt manual test you used before as a quick
  before/after comparison on the same prompt set.
- [ ] Once confirmed fixed, this is a strong signal the substring heuristic
  override (separate open item) may have been compensating for this very
  bug — worth re-evaluating whether it's still needed at all once real
  extraction is working, rather than deciding its fate in isolation.


## Task 8 — Multi-brand extraction for organic prompts (extractor-scoped)

### Problem
Organic prompts (purchase/discovery/fit/gifting) never mention a brand name
by design (golden rule #2). Today the job grid is prompt × provider × brand ×
sample, so the **same prompt string is probed twice** (once "for" Adore Me,
once "for" Victoria's Secret) even though nothing about the prompt differs
per brand. This wastes probe + extraction calls and produces non-comparable
data, since Adore Me's mention rate ends up measured against a different raw
sample than VS's mention rate for the "same" prompt.

Comparison prompts are unaffected — they name the brand explicitly in the
prompt text, so they genuinely need a separate probe per brand. **Do not
change comparison prompt handling.**

### Scope constraint
This fix is **extractor-only**. Do not change the provider `Probe()`
interface, provider implementations, or the runner's worker-pool mechanics.
The runner may still loop brand-by-brand for organic prompts as it does
today — the goal is to avoid re-probing and re-extracting from scratch per
brand by having the extractor produce a multi-brand signal from one raw
response, and to skip the duplicate probe call when one is already
available for that (prompt, provider, sample).

### 8.1 — `internal/agent/extractor.go`: multi-brand signal shape
- [ ] Replace single-brand `GEOSignal` with a result keyed by brand for
  organic extraction:
  ```go
  type GEOSignal struct {
      BrandMentioned       bool     `json:"brand_mentioned"`
      Sentiment            string   `json:"sentiment"`
      MentionCount         int      `json:"mention_count"`
      RecommendationRank   *int     `json:"recommendation_rank"`
      CompetitorsMentioned []string `json:"competitors_mentioned"`
      CitedURLs            []string `json:"cited_urls"`
  }

  type MultiBrandSignal map[string]GEOSignal // key: brand name
  ```
- [ ] Add `ExtractMultiBrand(ctx, rawResponse ProbeResponse, brands []string) (MultiBrandSignal, error)`
  for organic prompts — single Haiku call, single strict-JSON system prompt
  that asks the model to return a per-brand object in one pass (one extra
  small brand list in the prompt, not one call per brand).
- [ ] Keep the existing single-brand `Extract()` function as-is for
  comparison prompts (unchanged — still one call per brand since the probe
  itself is per-brand there).
- [ ] System prompt for `ExtractMultiBrand` must instruct: return JSON object
  with one key per brand name provided, each value matching the existing
  `GEOSignal` shape; no markdown fences; if a brand is not mentioned, return
  `brand_mentioned: false` and nulls/empties for the rest of that brand's
  fields — do not omit the key.
- [ ] `json.Unmarshal` into `MultiBrandSignal`; if parsing fails, return an
  error per-call (existing rule: log but don't fail the whole run; store
  `extraction_error` in DB) — same as today, just at the multi-brand level.
- [ ] Merge `CitedURLs` from `ProbeResponse` into **every** brand's signal
  (existing rule for Perplexity citation merge — citations aren't brand-
  specific at the raw-response level, only at brand-mention level if the
  extraction model determines a citation is brand-relevant; keep this
  simple — global `CitedURLs` merged into all brands per current behavior
  unless/until citation-to-brand mapping becomes its own task).

### 8.2 — Wiring: avoid duplicate probes for organic (minimal runner touch)
- [ ] In the organic job path only: before calling `provider.Probe()` for a
  given (prompt, provider, sample), check whether a raw response was already
  fetched for that exact (prompt, provider, sample) earlier in the same run
  (i.e. for a different brand). If so, reuse the cached `ProbeResponse` and
  call `agent.ExtractMultiBrand()` once; do not probe again.
- [ ] Simplest implementation: restructure the organic job unit from
  (prompt, provider, brand, sample) to (prompt, provider, sample) — one
  probe, one `ExtractMultiBrand` call, results split into per-brand DB rows
  after extraction. This is the cleanest way to satisfy "extractor-only" in
  spirit (no per-brand probe duplication) while keeping the runner's
  fan-out shape conceptually the same, just with brand resolved post-hoc
  instead of pre-hoc.
- [ ] Comparison job path: unchanged, still (prompt, provider, brand,
  sample) since prompt text differs per brand.

### 8.3 — Results storage
- [ ] Confirm `results` table write path: one row per (run, prompt, provider,
  sample, brand) as today — multi-brand extraction just means N rows get
  written from 1 probe response instead of N probe responses each producing
  1 row. Row shape/columns unchanged.
- [ ] Store the same `latency_ms` / `tokens_input` / `tokens_output` from the
  single shared probe call on all brand rows derived from it (avoid
  double-counting cost in reporting — flag this explicitly in cost
  dashboards/queries if they sum these columns per row, since summing would
  now double-count a single probe's token cost across brand rows).

### 8.4 — Verification
- [ ] Run a small batch (e.g. 5 organic prompts × 1 provider × 1 sample) and
  confirm: exactly 1 probe call per prompt (not 2), 1 extraction call per
  prompt (not 2), and 2 result rows written (one per brand) with
  independently correct `brand_mentioned`/`sentiment`/etc. per brand.
- [ ] Confirm comparison prompts still produce 1 probe + 1 extraction + 1
  result row per (prompt, provider, brand, sample) — unchanged.
- [ ] Re-check cost model doc (~$8-15/month standard) — organic probe/
  extraction call volume should now roughly halve; update the cost table if
  the agent regenerates it.
- [ ] Confirm stability scoring still works correctly given the new
  per-(prompt,provider,sample) probe shape — stability should now be
  computed once per prompt×provider across samples, with brand-level
  mention/sentiment stability derived from the shared sample set rather
  than from brand-duplicated samples. Flag to a human if this changes the
  stability formula's inputs meaningfully.

### Out of scope for Task 8
- Comparison prompt flow (untouched).
- Provider interface / `Probe()` signature (untouched).
- Visibility score formula (untouched, but verify inputs still make sense
  post-change per 8.4).

---

## Out of scope (Tasks 1-7, web search / model defaults fix)
- Extraction agent (`internal/agent/extractor.go`) — unaffected by Tasks 1-7
  (it IS the subject of Task 8 above).
- Two-call design (probe → extract) — unchanged, still two separate calls.
- Stability scoring / visibility score formula — unaffected by Tasks 1-7.

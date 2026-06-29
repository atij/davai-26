# Lighthouse — Hackathon Coding Tasks
### Agent Instructions · geo-tracker-backend + geo-tracker-frontend

Work through tasks in order within each priority tier. Each task is self-contained and testable before moving to the next. Perplexity provider tasks are deferred — do not touch provider config for Perplexity.

---

## Ground rules (never violate)

- Probe calls have NO system prompt — do not add one
- `comparison` category is NEVER included in organic metrics
- API keys and passwords are NEVER logged at any level
- All DB queries use sqlx named parameters — no string concatenation
- No global variables — inject all dependencies
- Every result row must have a valid `run_id` before being written

---

## P0 — Demo breaks without these

---

### `BE-01` · Fix: Visibility Score never persisted to DB

**Problem:** `CalcVisibilityScore` is called in `printFancySummary` for display only. The result is never written to the database. The dashboard always shows 0.0 because the API has no stored score to return.

**Files to change:** `internal/db/schema.sql`, `internal/db/results.go`, `cmd/run.go`

**Step 1 — Add table to `internal/db/schema.sql`:**

```sql
CREATE TABLE IF NOT EXISTS visibility_scores (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id            BIGINT UNSIGNED NOT NULL,
    brand             VARCHAR(255) NOT NULL,
    score             DECIMAL(6,2) NOT NULL DEFAULT 0,
    mention_rate      DECIMAL(6,2) NOT NULL DEFAULT 0,
    first_rec_rate    DECIMAL(6,2) NOT NULL DEFAULT 0,
    sentiment_score   DECIMAL(5,3) NOT NULL DEFAULT 0,
    citation_score    DECIMAL(6,2) NOT NULL DEFAULT 0,
    stability_score   DECIMAL(6,2) NOT NULL DEFAULT 0,
    provider_coverage DECIMAL(6,2) NOT NULL DEFAULT 0,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_brand_run (brand, run_id),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);
```

**Step 2 — Add DB struct and methods to `internal/db/results.go`:**

```go
type VisibilityScoreRow struct {
    ID               uint64    `db:"id"                json:"id"`
    RunID            uint64    `db:"run_id"             json:"run_id"`
    Brand            string    `db:"brand"              json:"brand"`
    Score            float64   `db:"score"              json:"score"`
    MentionRate      float64   `db:"mention_rate"       json:"mention_rate"`
    FirstRecRate     float64   `db:"first_rec_rate"     json:"first_rec_rate"`
    SentimentScore   float64   `db:"sentiment_score"    json:"sentiment_score"`
    CitationScore    float64   `db:"citation_score"     json:"citation_score"`
    StabilityScore   float64   `db:"stability_score"    json:"stability_score"`
    ProviderCoverage float64   `db:"provider_coverage"  json:"provider_coverage"`
    CreatedAt        time.Time `db:"created_at"         json:"created_at"`
}

func (r *ResultRepo) InsertVisibilityScore(v *VisibilityScoreRow) error {
    q := `INSERT INTO visibility_scores
        (run_id, brand, score, mention_rate, first_rec_rate, sentiment_score,
         citation_score, stability_score, provider_coverage)
        VALUES (:run_id, :brand, :score, :mention_rate, :first_rec_rate, :sentiment_score,
         :citation_score, :stability_score, :provider_coverage)`
    _, err := r.db.NamedExec(q, v)
    return err
}

func (r *ResultRepo) GetLatestVisibilityScore(brand string) (*VisibilityScoreRow, error) {
    var v VisibilityScoreRow
    err := r.db.Get(&v, `
        SELECT vs.*
        FROM visibility_scores vs
        JOIN runs ON vs.run_id = runs.id
        WHERE vs.brand = ? AND runs.status = 'done'
        ORDER BY runs.started_at DESC
        LIMIT 1`, brand)
    if err != nil {
        return nil, err
    }
    return &v, nil
}
```

**Step 3 — Wire into `cmd/run.go` after the stability scoring loop:**

Find the comment block `// 2. Explainer & Recommender` and insert before it:

```go
// 3. Calculate and persist visibility scores (organic only, per brand)
for _, b := range brands {
    var brandResults []db.Result
    for _, r := range results {
        if r.Brand == b && r.Category != "comparison" {
            brandResults = append(brandResults, r)
        }
    }
    stabilityScores, _ := resultRepo.GetStabilityScores(run.ID, b)
    vScore := scoring.CalcVisibilityScore(b, int64(run.ID), brandResults, stabilityScores)
    if err := resultRepo.InsertVisibilityScore(&db.VisibilityScoreRow{
        RunID:            run.ID,
        Brand:            b,
        Score:            vScore.Score,
        MentionRate:      vScore.MentionRate,
        FirstRecRate:     vScore.FirstRecRate,
        SentimentScore:   vScore.SentimentScore,
        CitationScore:    vScore.CitationScore,
        StabilityScore:   vScore.StabilityScore,
        ProviderCoverage: vScore.ProviderCoverage,
    }); err != nil {
        logger.Error("failed to insert visibility score", zap.String("brand", b), zap.Error(err))
    }
}
```

**Step 4 — Also collect stabilityScores for the printFancySummary call** (currently it passes an empty slice). After the loop above, update `printFancySummary` to pass the actual stability scores so the stdout output also reflects real numbers.

**Verify:** After `geo-tracker run`, run `SELECT * FROM visibility_scores;` — should have one row per brand with a non-zero score.

---

### `BE-02` · Fix: `GET /api/brands/:brand/summary` must return `visibility_score` from DB

**Problem:** The summary handler recomputes metrics from raw results on every request but never returns `visibility_score`. The frontend reads `data.visibility_score` and gets `undefined` → renders 0.0.

**File:** `internal/api/handlers.go`

In `GetSummary`, after fetching the organic summary data, also fetch the stored visibility score and merge it into the response:

```go
func (h *Handlers) GetSummary(w http.ResponseWriter, r *http.Request) {
    brandRaw := chi.URLParam(r, "brand")
    brand := resolveBrand(brandRaw)
    repo := db.NewResultRepo(h.db)

    runID, err := repo.GetLatestRunID()
    if err != nil {
        sendError(w, http.StatusNotFound, "no runs found", "NO_RUNS")
        return
    }

    summary, err := repo.GetOrganicSummary(brand, int64(runID))
    if err != nil {
        sendError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
        return
    }

    // Fetch stored visibility score (computed by run command, not recomputed here)
    vscore, err := repo.GetLatestVisibilityScore(brand)
    if err == nil && vscore != nil {
        summary.VisibilityScore = vscore.Score
        summary.FirstRecRate = vscore.FirstRecRate
        summary.CitationScore = vscore.CitationScore
        summary.StabilityScore = vscore.StabilityScore
        summary.ProviderCoverage = vscore.ProviderCoverage
    }

    summary.PromptType = "organic"
    sendJSON(w, http.StatusOK, summary)
}
```

Make sure `BrandSummary` struct in `internal/db/results.go` has all these fields:

```go
type BrandSummary struct {
    Brand            string             `db:"-"    json:"brand"`
    PromptType       string             `db:"-"    json:"prompt_type"`
    RunID            int64              `db:"run_id"          json:"run_id"`
    VisibilityScore  float64            `db:"-"               json:"visibility_score"`
    MentionRate      float64            `db:"mention_rate"    json:"mention_rate"`
    FirstRecRate     float64            `db:"-"               json:"first_rec_rate"`
    SentimentScore   float64            `db:"sentiment_score" json:"sentiment_score"`
    CitationScore    float64            `db:"-"               json:"citation_score"`
    StabilityScore   float64            `db:"-"               json:"stability_score"`
    ProviderCoverage float64            `db:"-"               json:"provider_coverage"`
    ProviderRates    map[string]float64 `db:"-"               json:"provider_rates"`
}
```

**Verify:** `curl localhost:8080/api/brands/adore-me/summary | jq .visibility_score` — should return a non-zero number.

---

### `BE-03` · Fix: Stability scores not flowing into visibility score (empty slice bug)

**Problem:** In `printFancySummary`, `brandStability` is declared but never populated — it stays nil. The stability component of the visibility score is always 0. The `// ... logic to aggregate ...` comment was never implemented.

**File:** `cmd/run.go` — in `printFancySummary`, populate `brandStability` before calling `CalcVisibilityScore`:

```go
// Replace the // ... logic to aggregate ... comment with:
for _, pr := range providersList {
    for _, p := range prompts {
        var promptSamples []db.Result
        for _, r := range brandResults {
            if r.Provider == pr.Name() && r.PromptID == p.ID {
                promptSamples = append(promptSamples, r)
            }
        }
        if len(promptSamples) > 0 {
            score := scoring.CalcStabilityScore(promptSamples)
            brandStability = append(brandStability, score)
        }
    }
}
```

Note: `printFancySummary` is display-only. The real fix is `BE-01` which stores stability before computing visibility. This task just fixes the stdout summary table to also show correct numbers.

**Verify:** After a run, the stdout table should show non-zero scores in the Score column.

---

### `BE-04` · Fix: Recommender agent called with empty data (produces generic output)

**Problem:** In `cmd/run.go`, `agent.Recommend()` is called with only `Brand` and `RunID` — no organic summary, no citation gaps, no weak categories. The recommender has no data to work with and produces generic recommendations that aren't specific to the current run.

**File:** `cmd/run.go` — find the recommender call and enrich the request:

```go
for _, b := range brands {
    // Fetch data the recommender needs
    organicSummary, _ := resultRepo.GetOrganicSummary(b, int64(run.ID))
    citationGaps, _ := resultRepo.GetCitationGap(b, run.ID)
    stabilityScores, _ := resultRepo.GetStabilityScores(run.ID, b)
    competitors, _ := resultRepo.GetTopCompetitors(b, run.ID, 5)

    recReq := agent.RecommendationRequest{
        Brand:           b,
        RunID:           run.ID,
        OrganicSummary:  organicSummary,
        CitationGaps:    citationGaps,
        StabilityScores: stabilityScores,
        TopCompetitors:  competitors,
    }
    recs, err := agent.Recommend(context.Background(), recReq)
    if err != nil {
        logger.Error("recommender failed", zap.String("brand", b), zap.Error(err))
        continue
    }
    for i := range recs {
        rec := &db.Recommendation{
            RunID:          run.ID,
            Brand:          b,
            Category:       recs[i].Category,
            Action:         recs[i].Action,
            ExpectedImpact: recs[i].ExpectedImpact,
            Rationale:      recs[i].Rationale,
            Status:         "pending",
        }
        if insertErr := resultRepo.InsertRecommendation(rec); insertErr != nil {
            logger.Error("failed to insert recommendation", zap.Error(insertErr))
        }
    }
}
```

**File:** `internal/agent/recommender.go` — ensure the prompt passed to Claude Sonnet includes all the request data. The prompt must explicitly ask for 3-5 specific actions. Update the system prompt to:

```go
const recommenderSystemPrompt = `You are a GEO (Generative Engine Optimization) strategist.
You will receive brand visibility data from AI chatbot analysis.
Return ONLY a JSON array of 3-5 recommendation objects. No markdown fences. No preamble.

Each object must have:
{
  "category": "fit|purchase|discovery|gifting|comparison",
  "action": "specific actionable task (1-2 sentences)",
  "expected_impact": "estimated Visibility Score change and timeframe",
  "rationale": "cite specific data from the input (competitor name, domain, category gap)",
  "priority": 1
}

Priority 1 = highest impact. Actions must reference specific data points from the input.
Never produce generic advice. Every action must name a specific category, competitor, or domain.`
```

**Verify:** After a run, `SELECT COUNT(*) FROM recommendations;` should return 3-5 rows with specific, data-driven action text.

---

### `BE-05` · Fix: Runs table missing `started_at` / `duration` (shows N/A in UI)

**Problem:** The Runs page shows "N/A" for the time column. The `started_at` field is not being set when the run record is created.

**File:** `cmd/run.go` — when creating the run record, set `started_at` explicitly:

```go
run := db.Run{
    Status:    "running",
    StartedAt: time.Now(), // add this
    Brand:     strings.Join(brands, ","),
}
```

**File:** `internal/db/schema.sql` — verify `runs` table has:

```sql
started_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
completed_at DATETIME NULL,
duration_seconds INT NULL,
```

**File:** `internal/db/results.go` — verify `UpdateRunStatus` sets `completed_at` and calculates duration:

```go
func (r *ResultRepo) UpdateRunStatus(id uint64, status string, cost float64) error {
    _, err := r.db.Exec(`
        UPDATE runs
        SET status = ?,
            total_cost_usd = ?,
            completed_at = NOW(),
            duration_seconds = TIMESTAMPDIFF(SECOND, started_at, NOW())
        WHERE id = ?`, status, cost, id)
    return err
}
```

**File:** `internal/api/handlers.go` — ensure `GetRuns` returns `started_at`, `completed_at`, and `duration_seconds` in the JSON response.

**Verify:** After a run, `SELECT started_at, completed_at, duration_seconds FROM runs;` — all three columns should be populated.

---

### `FE-01` · Fix: Dashboard does not render `ExplainabilityPanel` or `LiveRunButton`

**Problem:** Two of the highest-impact demo components are not wired into `app/dashboard/page.tsx`. The spec requires `ExplainabilityPanel` (shows what changed and why) and `LiveRunButton` (floating button, fires a prompt live via SSE).

**File:** `app/dashboard/page.tsx`

Import and add both components. The full required layout is:

```tsx
import { ExplainabilityPanel } from '@/components/dashboard/ExplainabilityPanel'
import { LiveRunButton } from '@/components/dashboard/LiveRunButton'

// Inside the page, after CitationGapTable and PromptResultsTable:
{latestRunId && (
  <ExplainabilityPanel brand={brand} runId={latestRunId} />
)}

// Floating LiveRunButton — add at the bottom of the page, outside the main content div:
<LiveRunButton brand={brand} />
```

To get `latestRunId`, fetch it from the summary data:

```tsx
const { data: summary } = useSummary(brand)
const latestRunId = summary?.run_id ?? null
```

**Verify:** Reload the dashboard — the explainability panel should appear below the charts (or show nothing gracefully if no explanation exists for this run). The "Run Now" button should appear floating at bottom-right.

---

### `FE-02` · Fix: `VisibilityScoreCard` reads `visibility_score` field correctly

**Problem:** With the old API response missing `visibility_score`, the card renders 0.0. Now that `BE-02` adds it to the API response, verify the frontend is reading the right field name.

**File:** `lib/types.ts` — ensure `BrandSummary` interface has:

```ts
export interface BrandSummary {
  brand: string
  prompt_type: 'organic'
  run_id: number
  run_at: string
  visibility_score: number    // ← must match backend JSON key exactly
  mention_rate: number
  first_rec_rate: number
  sentiment_score: number
  citation_score: number
  stability_score: number
  provider_coverage: number
  provider_rates: Record<string, number>
}
```

**File:** `components/dashboard/VisibilityScoreCard.tsx` — verify it reads `summary.visibility_score`, not `summary.score` or any other key.

**File:** `components/dashboard/MetricsRow.tsx` — verify the four KPI cards read:
- `summary.mention_rate` for Mention Rate
- `summary.first_rec_rate` for First Rec Rate
- `summary.sentiment_score` for Sentiment Score
- `summary.stability_score` for Stability Score

**Verify:** Dashboard loads and shows the real composite score (should be ~10-15 with 31% mention rate).

---

### `FE-03` · Fix: `HeadToHeadSection` missing from dashboard page

**Problem:** The head-to-head comparison section is not rendered on the dashboard. Per spec it is conditionally shown — only when comparison data exists.

**File:** `app/dashboard/page.tsx`

Add after the `ExplainabilityPanel`:

```tsx
import { HeadToHeadSection } from '@/components/dashboard/HeadToHeadSection'

// HeadToHeadSection handles its own data fetching and renders nothing if no data
<HeadToHeadSection brand={brand} />
```

`HeadToHeadSection` internally calls `useComparisonSummary(brand)` and renders nothing if the response is null or returns `NO_COMPARISON_DATA` error. No conditional needed in the page itself.

**Verify:** If comparison prompts exist in the DB, the section appears. If not, nothing renders — no error, no empty state.

---

## P1 — Demo is weak without these

---

### `BE-06` · Fix: VS sentiment score always 0.0

**Problem:** Compare page shows Victoria's Secret sentiment as 0.0. The extraction agent extracts sentiment correctly per brand, but the aggregation query may be using the wrong brand name string for VS — likely `"Victoria's Secret"` vs `"Victoria's Secret"` encoding mismatch or the `resolveBrand` function not handling the VS alias.

**File:** `internal/api/handlers.go` — verify `resolveBrand`:

```go
func resolveBrand(raw string) string {
    switch strings.ToLower(strings.ReplaceAll(raw, "-", " ")) {
    case "adore me":
        return "Adore Me"
    case "victorias secret", "victoria's secret", "victoria secret":
        return "Victoria's Secret"
    default:
        return raw
    }
}
```

**File:** `internal/db/results.go` — in `GetOrganicSummary`, verify the sentiment aggregation query uses the brand parameter as a WHERE clause, not hardcoded:

```sql
WHERE r.run_id = :run_id
  AND r.brand  = :brand        -- must match exactly what is stored in results table
  AND p.category != 'comparison'
```

Run `SELECT DISTINCT brand FROM results;` to confirm the exact brand strings being stored, then make sure `resolveBrand` returns those exact strings.

**Verify:** `curl "localhost:8080/api/brands/victorias-secret/summary" | jq .sentiment_score` — should return a non-zero value.

---

### `BE-07` · Add: `GET /api/explain/:run_id` endpoint

**Problem:** The `ExplainabilityPanel` frontend component calls `GET /api/explain/:run_id?brand=X` but this endpoint may not be wired in the chi router.

**File:** `internal/api/server.go` — verify the route is registered:

```go
r.Get("/api/explain/{run_id}", h.GetExplain)
```

**File:** `internal/api/handlers.go` — add the handler if missing:

```go
func (h *Handlers) GetExplain(w http.ResponseWriter, r *http.Request) {
    runIDStr := chi.URLParam(r, "run_id")
    runID, _ := strconv.ParseUint(runIDStr, 10, 64)
    brand := resolveBrand(r.URL.Query().Get("brand"))

    repo := db.NewResultRepo(h.db)
    explanation, err := repo.GetExplanation(runID, brand)
    if err != nil {
        sendError(w, http.StatusNotFound, "no explanation found", "NOT_FOUND")
        return
    }
    sendJSON(w, http.StatusOK, explanation)
}
```

**File:** `internal/db/results.go` — add `GetExplanation` query against the `explanations` table (verify this table exists in schema.sql; add it if not):

```sql
CREATE TABLE IF NOT EXISTS explanations (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id      BIGINT UNSIGNED NOT NULL,
    brand       VARCHAR(255) NOT NULL,
    summary     TEXT NOT NULL,
    drivers     JSON NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_run_brand (run_id, brand)
);
```

**Verify:** After a run, `SELECT summary FROM explanations LIMIT 1;` should return non-empty text. The ExplainabilityPanel should render it.

---

### `BE-08` · Fix: Recommendations response shape missing `priority` field

**Problem:** The recommendations page sorts by `rec.priority` in the frontend, but the `Recommendation` DB struct and the recommender agent output may not be storing the `priority` field. The result is all recommendations sort identically (priority 0).

**File:** `internal/db/results.go` — add `priority` to the `Recommendation` struct:

```go
type Recommendation struct {
    ID             uint64     `db:"id"             json:"id"`
    RunID          uint64     `db:"run_id"          json:"run_id"`
    Brand          string     `db:"brand"           json:"brand"`
    Priority       int        `db:"priority"        json:"priority"`   // ← add this
    Category       string     `db:"category"        json:"category"`
    Action         string     `db:"action"          json:"action"`
    ExpectedImpact string     `db:"expected_impact" json:"expected_impact"`
    Rationale      string     `db:"rationale"       json:"rationale"`
    Status         string     `db:"status"          json:"status"`
    ImplementedAt  *time.Time `db:"implemented_at"  json:"implemented_at"`
    CreatedAt      time.Time  `db:"created_at"      json:"created_at"`
}
```

**File:** `internal/db/schema.sql` — add `priority` column to recommendations table:

```sql
priority        INT NOT NULL DEFAULT 1,
```

**File:** `internal/db/results.go` — update `InsertRecommendation` to include `priority` in the INSERT:

```sql
INSERT INTO recommendations (run_id, brand, priority, category, action, expected_impact, rationale, status)
VALUES (:run_id, :brand, :priority, :category, :action, :expected_impact, :rationale, :status)
```

**File:** `cmd/run.go` — when calling `InsertRecommendation`, map the `Priority` field from the agent response:

```go
rec := &db.Recommendation{
    RunID:          run.ID,
    Brand:          b,
    Priority:       recs[i].Priority,   // ← add this
    Category:       recs[i].Category,
    Action:         recs[i].Action,
    ExpectedImpact: recs[i].ExpectedImpact,
    Rationale:      recs[i].Rationale,
    Status:         "pending",
}
```

**Verify:** `SELECT priority, action FROM recommendations ORDER BY priority;` — should show 3-5 rows with distinct priority values 1 through N.

---

### `FE-04` · Fix: `PromptResultsTable` missing from dashboard

**Problem:** The dashboard layout in the spec includes a per-prompt results table showing which providers mentioned the brand. It is not being rendered in the current dashboard page.

**File:** `app/dashboard/page.tsx`

Add after `CitationGapTable`:

```tsx
import { PromptResultsTable } from '@/components/dashboard/PromptResultsTable'

// Fetch prompt results for the latest run
const { data: promptResults } = useRunDetail(latestRunId ?? 0)

{promptResults && promptResults.length > 0 && (
  <PromptResultsTable prompts={promptResults} />
)}
```

**File:** `hooks/useRunDetail.ts` — verify it is wired to `GET /api/runs/:id/results` and returns the right type. Skip rendering if `latestRunId` is null.

**Verify:** Dashboard shows a table of prompts with provider hit/miss indicators per row.

---

### `FE-05` · Improve: Trend chart empty state

**Problem:** With only one run in the DB, the trend chart shows a single floating dot with no context. It needs a clear message rather than a near-empty chart.

**File:** `components/charts/TrendChart.tsx`

Add a guard before the Recharts render:

```tsx
if (!data || data.length < 2) {
  return (
    <div className="bg-white rounded-3xl border border-slate-200 p-8 flex flex-col items-center justify-center h-64 gap-3">
      <p className="text-sm font-bold text-slate-900">Mention Trend</p>
      <p className="text-xs text-slate-400 text-center max-w-48">
        Trend appears after 2+ runs. Run the pipeline again tomorrow to see movement.
      </p>
      {data?.length === 1 && (
        <p className="text-2xl font-black text-indigo-600">{data[0].mention_rate.toFixed(1)}%</p>
      )}
    </div>
  )
}
```

**Verify:** Dashboard loads cleanly with one data point — shows the current mention rate and an explanatory message instead of a lonely dot.

---

### `BE-09` · Add: Seed 45 more prompts to `prompts/seed.yaml`

**Problem:** Only 5 prompts exist. Statistical signal is too thin for meaningful visibility scoring. The spec requires 50 prompts across 5 categories.

**File:** `prompts/seed.yaml`

The 5 existing prompts cover: 1 purchase, 1 comparison, 1 discovery, 1 purchase, 1 fit. Add the following to reach the full 50. Write each as a real customer query to an AI chatbot — conversational, not formal. No brand names in organic prompts.

Required counts to add:
- `purchase`: add 13 more (to reach 15 total)
- `discovery`: add 9 more (to reach 10 total)
- `fit`: add 9 more (to reach 10 total)
- `comparison`: add 9 more (to reach 10 total — brand names ARE allowed here)
- `gifting`: add 5 new

Example format:
```yaml
prompts:
  - text: "Where can I find a good wireless bra that doesn't dig in?"
    category: purchase
  - text: "What are the most comfortable everyday bras right now?"
    category: purchase
  - text: "I need a bra for my wide rib cage but small cup, any suggestions?"
    category: fit
  - text: "Which is better for everyday comfort, Adore Me or ThirdLove?"
    category: comparison
  - text: "What lingerie brands are actually sustainable in 2026?"
    category: discovery
  - text: "Good lingerie gift ideas for my wife who just had a baby?"
    category: gifting
```

After editing the file, re-import: `geo-tracker prompts import prompts/seed.yaml`

**Verify:** `geo-tracker prompts list` shows 50 active prompts across 5 categories.

---

### `BE-10` · Add: Generate and commit demo baseline dataset

**Problem:** With only one run and no historical data, the trend chart has one point and the explainability panel has nothing to diff. A pre-baked second run from "yesterday" solves both.

**Steps:**

1. Run `geo-tracker run` to completion (this is the second run — your current data is already run #1)
2. If you need to simulate historical data, insert a fake earlier run directly:

```sql
-- Insert a fake run from yesterday for trend demonstration
INSERT INTO runs (status, started_at, completed_at, duration_seconds, total_cost_usd)
VALUES ('done', DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 23 HOUR 55 MINUTE), 290, 0.48);

-- Use the new run ID (e.g. 0) to backfill visibility scores with slightly lower numbers
-- to show upward trend movement:
INSERT INTO visibility_scores (run_id, brand, score, mention_rate, first_rec_rate, sentiment_score, citation_score, stability_score, provider_coverage)
VALUES
  (0, 'Adore Me',          22.4, 24.2, 0.0, 0.28, 0.0, 0.0, 66.7),
  (0, 'Victoria''s Secret', 18.1, 19.8, 0.0, 0.0,  0.0, 0.0, 66.7);
```

Replace `0` with the actual inserted run ID.

3. Commit `demo/baseline.json`:
```bash
geo-tracker run --format json > demo/baseline.json
git add demo/baseline.json && git commit -m "chore: add demo baseline dataset"
```

**Verify:** Trend chart now shows 2 data points with upward movement. ExplainabilityPanel can diff run 1 vs run 2.

---

## P2 — Polish before the CTO demo

---

### `FE-06` · Polish: Runs page — show duration and fix timestamp

**File:** `app/runs/page.tsx` and `components/` — wherever the run row is rendered

Replace the "N/A" time display:

```tsx
// Replace the N/A logic with:
const duration = run.duration_seconds
  ? `${Math.floor(run.duration_seconds / 60)}m ${run.duration_seconds % 60}s`
  : '—'

const startedAt = run.started_at
  ? new Date(run.started_at).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  : '—'
```

---

### `FE-07` · Polish: Compare page — add Head-to-Head charts section

**File:** `app/compare/page.tsx`

The compare page currently shows only organic metrics. Add the head-to-head section below with a clear visual divider:

```tsx
import { HeadToHeadCharts } from '@/components/compare/HeadToHeadCharts'

// After the organic section:
<div className="border-t-2 border-slate-100 pt-8 mt-8">
  <div className="flex items-center gap-3 mb-6">
    <h2 className="text-lg font-black text-slate-900">Head-to-Head</h2>
    <span className="text-xs font-bold text-slate-400 bg-slate-100 px-2 py-1 rounded-full">
      COMPARISON PROMPTS ONLY
    </span>
  </div>
  <HeadToHeadCharts brands={['Adore Me', "Victoria's Secret"]} />
</div>
```

---

### `FE-08` · Polish: Prompts page — expandable rows with stability scores

**File:** `app/prompts/page.tsx`

The category filter tabs work but the expandable rows (click chevron to expand) need to show a provider hit/miss grid and stability score. The `>` chevron is visible in the screenshot but click does nothing.

Wire the row expansion:

```tsx
const [expandedId, setExpandedId] = useState<number | null>(null)

// In the row click handler:
onClick={() => setExpandedId(expandedId === prompt.id ? null : prompt.id)}

// In the expanded row content (shown when expandedId === prompt.id):
{expandedId === prompt.id && (
  <PromptExpandedRow promptId={prompt.id} />
)}
```

Create `components/prompts/PromptExpandedRow.tsx` that calls `usePromptResults(promptId)` → `GET /api/prompts/:id/results` and renders a grid of provider × sample results with ✓/✗ indicators and the stability score.

---

## Definition of done for hackathon demo

Before the demo starts, verify this checklist:

- [ ] Dashboard Visibility Score shows a real non-zero number for Adore Me
- [ ] Dashboard Mention Rate, First Rec Rate, Sentiment Score all show real values
- [ ] Share of Voice chart shows 3 bars (Claude, ChatGPT, Gemini — Perplexity added during hackathon)
- [ ] Trend chart shows 2+ data points with upward movement
- [ ] Explainability panel renders text explaining what changed between run 1 and run 2
- [ ] LiveRunButton is visible and clicking it opens the SSE modal
- [ ] Compare page shows both brands side by side with different values
- [ ] Recommendations page shows 3-5 specific, data-driven actions
- [ ] Runs page shows a real timestamp and duration (not N/A)
- [ ] Prompts page shows 50 prompts across all 5 category tabs

---

*Project Lighthouse · Hackathon Tasks · Adore Me Tech · June 2026*

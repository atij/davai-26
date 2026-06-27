# Project Lighthouse — Frontend Implementation
### Coding Agent Document · geo-tracker-frontend

---

## What this document is

Complete implementation specification for the `geo-tracker-frontend` Next.js application.
Work through sections in order. Each section is independently testable before moving on.
This app is read-only — it never writes data. All data comes from the backend API.

---

## Tech stack

| Concern | Choice |
|---|---|
| Framework | Next.js 14+ (App Router) |
| Language | TypeScript — strict mode, no `any` |
| Styling | Tailwind CSS |
| Data fetching | SWR |
| Charts | Recharts |
| Icons | Lucide React |

---

## Repository structure

```
geo-tracker-frontend/
├── app/
│   ├── layout.tsx
│   ├── page.tsx                    # → redirect to /dashboard
│   ├── dashboard/
│   │   ├── page.tsx                # main dashboard
│   │   └── loading.tsx
│   ├── compare/
│   │   ├── page.tsx                # side-by-side brand comparison
│   │   └── loading.tsx
│   ├── prompts/
│   │   ├── page.tsx                # prompt library + results
│   │   └── loading.tsx
│   ├── runs/
│   │   ├── page.tsx                # run history
│   │   └── [id]/page.tsx           # single run detail
│   └── recommendations/
│       └── page.tsx                # recommendation engine output
├── components/
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── Topbar.tsx
│   │   └── PageShell.tsx
│   ├── charts/
│   │   ├── ShareOfVoiceChart.tsx
│   │   ├── TrendChart.tsx
│   │   ├── SentimentDonut.tsx
│   │   ├── CompetitorBars.tsx
│   │   └── StabilityHeatmap.tsx
│   ├── dashboard/
│   │   ├── MetricCard.tsx
│   │   ├── VisibilityScoreCard.tsx
│   │   ├── MetricsRow.tsx
│   │   ├── PromptResultsTable.tsx
│   │   ├── RunStatusBadge.tsx
│   │   ├── HeadToHeadSection.tsx
│   │   ├── ExplainabilityPanel.tsx
│   │   ├── CitationGapTable.tsx
│   │   └── LiveRunButton.tsx
│   ├── compare/
│   │   ├── CompareMetrics.tsx
│   │   └── HeadToHeadCharts.tsx
│   ├── recommendations/
│   │   ├── RecommendationCard.tsx
│   │   └── RecommendationList.tsx
│   └── ui/
│       ├── Pill.tsx
│       ├── ProviderDot.tsx
│       ├── Skeleton.tsx
│       └── EmptyState.tsx
├── hooks/
│   ├── useSummary.ts
│   ├── useTrend.ts
│   ├── useComparisonSummary.ts
│   ├── useComparisonTrend.ts
│   ├── useHeadToHead.ts
│   ├── useStability.ts
│   ├── useCitationGap.ts
│   ├── useCompetitors.ts
│   ├── useExplain.ts
│   ├── useRecommendations.ts
│   ├── usePrompts.ts
│   ├── useRuns.ts
│   └── useRunDetail.ts
├── lib/
│   ├── api.ts
│   ├── types.ts
│   └── utils.ts
├── styles/globals.css
├── .env.example
├── .env.local                      # gitignored
├── tailwind.config.ts
├── tsconfig.json
└── next.config.ts
```

---

## Section 1 — Environment

```bash
# .env.example
NEXT_PUBLIC_API_URL=http://localhost:8080
```

One variable. In Kubernetes, set to internal backend service URL.
No secrets — frontend is read-only.

---

## Section 2 — TypeScript types

File: `lib/types.ts`

```ts
export type Sentiment  = 'positive' | 'neutral' | 'negative' | 'not_mentioned'
export type RunStatus  = 'running' | 'done' | 'failed'
export type PromptCategory = 'purchase' | 'discovery' | 'fit' | 'comparison' | 'gifting'
export type Provider   = 'claude' | 'chatgpt' | 'perplexity' | 'gemini'
export type PromptType = 'organic' | 'comparison'

export interface BrandSummary {
  brand:              string
  prompt_type:        'organic'
  run_id:             number
  run_at:             string
  visibility_score:   number        // 0-100 composite
  mention_rate:       number        // 0-100
  first_rec_rate:     number        // 0-100
  sentiment_score:    number        // -1 to 1
  citation_score:     number        // 0-100
  stability_score:    number        // 0-100
  provider_coverage:  number        // 0-100
  provider_rates:     Record<Provider, number>
  top_provider:       Provider
  weakest_provider:   Provider
}

export interface ComparisonSummary {
  brand:               string
  prompt_type:         'comparison'
  run_id:              number
  run_at:              string
  total_prompts:       number
  mention_rate:        number
  win_rate:            number
  avg_rank:            number | null
  sentiment_breakdown: SentimentBreakdown
  provider_breakdown:  Record<Provider, ComparisonProviderResult>
}

export interface ComparisonProviderResult {
  mention_rate: number
  win_rate:     number
  avg_rank:     number | null
  sentiment:    Sentiment
}

export interface SentimentBreakdown {
  positive: number
  neutral:  number
  negative: number
}

export interface TrendPoint {
  run_id:       number
  run_at:       string
  mention_rate: number
}

export interface ComparisonTrendPoint {
  run_id:       number
  run_at:       string
  mention_rate: number
  win_rate:     number
}

export interface StabilityScore {
  prompt_id:       number
  prompt_text:     string
  category:        PromptCategory
  provider:        Provider
  sample_count:    number
  mention_rate:    number
  rank_variance:   number
  stability_score: number          // 0-100
}

export interface CitationGapEntry {
  domain:         string
  citation_count: number
  category:       PromptCategory
}

export interface Competitor {
  name:      string
  frequency: number
}

export interface Recommendation {
  id:               number
  priority:         number
  category:         PromptCategory
  action:           string
  expected_impact:  string
  rationale:        string
  status:           'pending' | 'implemented'
  implemented_at:   string | null
}

export interface Explanation {
  summary:  string
  drivers:  string[]
}

export interface Prompt {
  id:         number
  text:       string
  category:   PromptCategory
  active:     boolean
  created_at: string
}

export interface Run {
  id:             number
  started_at:     string
  finished_at:    string | null
  status:         RunStatus
  prompt_count:   number
  brand_count:    number
  sample_count:   number
  total_cost_usd: number | null
  duration_seconds: number | null
}

export interface LiveRunResult {
  provider:        Provider
  brand:           string
  brand_mentioned: boolean
  sentiment:       Sentiment
  streaming:       boolean
}

// Chart data shapes — shape data to these before passing to chart components
export type SOVDatum        = { provider: string; rate: number }
export type TrendDatum      = { run_at: string; [brand: string]: number | string }
export type SentimentDatum  = { name: string; value: number; color: string }
export type CompetitorDatum = { name: string; frequency: number }
export type StabilityDatum  = { prompt: string; provider: Provider; score: number }
```

---

## Section 3 — API client

File: `lib/api.ts`

```ts
const BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'
const enc = encodeURIComponent

export const api = {
  health:              ()                        => fetch(`${BASE}/api/health`),
  runs:                (page = 1)               => fetch(`${BASE}/api/runs?page=${page}&per_page=20`),
  runDetail:           (id: number)              => fetch(`${BASE}/api/runs/${id}/results`),
  brands:              ()                        => fetch(`${BASE}/api/brands`),
  summary:             (brand: string)           => fetch(`${BASE}/api/brands/${enc(brand)}/summary`),
  trend:               (brand: string, n = 10)   => fetch(`${BASE}/api/brands/${enc(brand)}/trend?runs=${n}`),
  comparisonSummary:   (brand: string)           => fetch(`${BASE}/api/brands/${enc(brand)}/comparison-summary`),
  comparisonTrend:     (brand: string, n = 10)   => fetch(`${BASE}/api/brands/${enc(brand)}/comparison-trend?runs=${n}`),
  stability:           (brand: string)           => fetch(`${BASE}/api/brands/${enc(brand)}/stability`),
  citationGap:         (brand: string)           => fetch(`${BASE}/api/brands/${enc(brand)}/citation-gap`),
  compareOrganic:      (a: string, b: string)    => fetch(`${BASE}/api/compare/organic?brands=${enc(a)},${enc(b)}`),
  headToHead:          (a: string, b: string)    => fetch(`${BASE}/api/compare/head-to-head?brands=${enc(a)},${enc(b)}`),
  competitors:         (brand: string)           => fetch(`${BASE}/api/competitors?brand=${enc(brand)}`),
  recommendations:     (brand: string)           => fetch(`${BASE}/api/recommendations?brand=${enc(brand)}&status=pending`),
  explain:             (runId: number, brand: string) => fetch(`${BASE}/api/explain/${runId}?brand=${enc(brand)}`),
  prompts:             ()                        => fetch(`${BASE}/api/prompts`),
  promptResults:       (id: number)              => fetch(`${BASE}/api/prompts/${id}/results`),
  // SSE — returns EventSource, not fetch
  liveRun:             (promptId: number, providers: Provider[]) =>
    new EventSource(`${BASE}/api/runs/live?prompt_id=${promptId}&providers=${providers.join(',')}`),
}
```

All fetch functions go here. No inline fetch calls in components or pages.

---

## Section 4 — SWR hooks

File: one hook per API endpoint in `hooks/`.

Pattern for all hooks:
```ts
import useSWR from 'swr'
import { api } from '@/lib/api'

const fetcher = (url: string) => fetch(url).then(r => r.json())

export function useSummary(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/summary` : null,
    () => api.summary(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { data: data as BrandSummary | undefined, isLoading, error }
}
```

Special case — `useComparisonSummary`:
```ts
export function useComparisonSummary(brand: string) {
  const { data, isLoading, error } = useSWR(...)
  // If error.code === 'NO_COMPARISON_DATA' → return data: null, no error state
  // Component renders nothing when data is null — no empty state shown
  const noData = error?.code === 'NO_COMPARISON_DATA'
  return {
    data: noData ? null : data as ComparisonSummary | null,
    isLoading: noData ? false : isLoading,
    error: noData ? null : error,
  }
}
```

All hooks use `refreshInterval: 30_000`.
Hooks return typed `{ data, isLoading, error }` — never implicit `any`.

---

## Section 5 — Colour system

File: `tailwind.config.ts` — add custom tokens:

```ts
colors: {
  brand: {
    adoreme: '#7F77DD',
    vs:      '#1D9E75',
  },
  provider: {
    claude:     '#AFA9EC',
    chatgpt:    '#5DCAA5',
    perplexity: '#EF9F27',
    gemini:     '#F0997B',
  },
  sentiment: {
    positive: '#5DCAA5',
    neutral:  '#B4B2A9',
    negative: '#F09595',
  },
}
```

File: `lib/utils.ts` — colour and label maps:

```ts
export const providerColour: Record<Provider, string> = {
  claude: '#AFA9EC', chatgpt: '#5DCAA5', perplexity: '#EF9F27', gemini: '#F0997B',
}

export const brandColour: Record<string, string> = {
  'Adore Me':           '#7F77DD',
  "Victoria's Secret":  '#1D9E75',
}

export const sentimentClass: Record<Sentiment, string> = {
  positive:      'bg-[#E1F5EE] text-[#085041]',
  neutral:       'bg-[#F1EFE8] text-[#444441]',
  negative:      'bg-[#FCEBEB] text-[#791F1F]',
  not_mentioned: 'bg-gray-100 text-gray-400',
}

export const categoryClass: Record<PromptCategory, string> = {
  purchase:   'bg-[#EEEDFE] text-[#3C3489]',
  discovery:  'bg-[#E1F5EE] text-[#085041]',
  comparison: 'bg-[#FAEEDA] text-[#633806]',
  fit:        'bg-[#FAECE7] text-[#712B13]',
  gifting:    'bg-[#E6F1FB] text-[#0C447C]',
}

export const formatPercent  = (n: number) => `${n.toFixed(1)}%`
export const formatScore    = (n: number) => n.toFixed(1)
export const formatDate     = (s: string) => new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
export const formatCost     = (n: number) => `$${n.toFixed(4)}`
```

Never hardcode hex values in components. Always use these maps.

---

## Section 6 — UI primitives (`components/ui/`)

### `Pill.tsx`
Variants: category (5 colours), sentiment (4 colours), run status (3 colours).
Props: `{ variant: PromptCategory | Sentiment | RunStatus; label?: string }`

### `ProviderDot.tsx`
Shows Cl / GP / Px / Gm abbreviation. Hit = coloured background, miss = muted.
Props: `{ provider: Provider; hit: boolean }`

### `Skeleton.tsx`
Pulse animation block. Props: `{ className?: string }`

### `EmptyState.tsx`
Icon + message + optional CTA. Props: `{ message: string; cta?: { label: string; onClick: () => void } }`

---

## Section 7 — Charts (`components/charts/`)

All charts:
- Wrapped in `<ResponsiveContainer width="100%" height={N}>`
- No data transformation inside chart — receive pre-shaped props
- Tooltip formatter appends `%` for rate values
- Colours from `lib/utils.ts` maps — never hardcoded in chart

### `ShareOfVoiceChart.tsx`
`BarChart` grouped by provider. Props: `{ data: SOVDatum[]; color: string }`

### `TrendChart.tsx`
`LineChart` with one line per brand. Props: `{ data: TrendDatum[]; brands: string[] }`
Line colour per brand from `brandColour` map.

### `SentimentDonut.tsx`
`PieChart` with `innerRadius`. Props: `{ data: SentimentDatum[] }`

### `CompetitorBars.tsx`
`BarChart` horizontal. Props: `{ data: CompetitorDatum[] }`

### `StabilityHeatmap.tsx`
Grid where rows = prompts (truncated), columns = providers.
Cell colour = stability score (red → amber → green gradient).
Props: `{ data: StabilityDatum[] }`
Renders as a custom SVG grid — not a Recharts component.
Show only the 10 least stable prompts by default with a "show all" toggle.

---

## Section 8 — Dashboard components

### `VisibilityScoreCard.tsx`

The hero metric. Large circular progress indicator showing the 0-100 Visibility Score.
Ring colour changes: 0-40 = red, 40-70 = amber, 70-100 = green.
Shows score delta vs previous run (+3.2 / -1.4).
Props: `{ score: number; delta?: number; brand: string }`

### `MetricCard.tsx`
Standard KPI tile. Props: `{ label: string; value: string; sub?: string; up?: boolean; subtitle?: string }`
When `subtitle` is "organic prompts only" — render in muted text below label.

### `MetricsRow.tsx`
4-card responsive grid. Props: `{ summary: BrandSummary }`
Cards: Mention Rate (subtitle: "organic prompts only"), First Rec. Rate, Sentiment Score, Stability Score.

### `PromptResultsTable.tsx`
Table: prompt text, category pill, mention count (x/4 providers), provider dots.
Props: `{ prompts: PromptResult[] }`

### `RunStatusBadge.tsx`
Coloured pill. Props: `{ status: RunStatus }`

### `HeadToHeadSection.tsx`
Self-contained section. Renders nothing if `data === null`.
Contains: section heading with "comparison prompts only" label, win rate card, avg rank card, per-provider breakdown table.
Props: `{ brand: string }`
Internally calls `useComparisonSummary(brand)`.

### `ExplainabilityPanel.tsx`
Shows the Explanation for the latest run.
Summary text in a styled blockquote. Drivers as a bullet list.
"Why did visibility change?" heading. Link to previous run for comparison.
Renders nothing if no explanation exists (first run).
Props: `{ brand: string; runId: number }`

### `CitationGapTable.tsx`
"Sources AI cites when we're not mentioned" — most actionable view.
Shows domain, citation count, category pill.
Sorted by citation_count desc. Max 10 rows with "show more" toggle.
Props: `{ brand: string }`
Internally calls `useCitationGap(brand)`.

### `LiveRunButton.tsx`
"Run now" button. On click: opens a modal showing results streaming in via SSE.
One row per provider, populates as results arrive.
Uses `api.liveRun()` which returns an `EventSource`.
Requires user to select a prompt from a dropdown before firing.
Close button stops the SSE stream.
Props: `{ brand: string }`

---

## Section 9 — Pages

### `/dashboard`

Layout:
```
Topbar (brand switcher: Adore Me | Victoria's Secret)
VisibilityScoreCard                    ← hero metric
MetricsRow (4 KPI cards)
[ShareOfVoiceChart] [TrendChart]       ← 2-col grid
[CompetitorBars]    [SentimentDonut]   ← 2-col grid
CitationGapTable                       ← full width
PromptResultsTable                     ← full width
HeadToHeadSection                      ← conditionally rendered
ExplainabilityPanel                    ← conditionally rendered
LiveRunButton                          ← floating bottom-right
```

Brand held in URL param `?brand=adoreme`. Read with `useSearchParams()`.
Default: `adoreme`.

### `/compare`

Layout:
```
── Organic visibility ──────────────────────────
CompareMetrics (two MetricsRows side by side)
Overlaid TrendChart (two lines, different colours)
Grouped ShareOfVoiceChart (both brands per provider)

── Head-to-head (comparison prompts) ───────────
HeadToHeadCharts
```

Always shows both brands. No brand switcher.
Section dividers with clear labels — visual separation is mandatory.

### `/prompts`

Category filter tabs: All | Purchase | Discovery | Fit | Comparison | Gifting
Table: id, category pill, text, active status, created date.
Expandable row: click → show provider hit/miss grid for latest run across all samples.
Show stability score per prompt in expanded row.

### `/runs`

Paginated list: id, started_at, status badge, prompt_count, sample_count, total_cost_usd, duration.
Click row → `/runs/:id`.

### `/runs/:id`

Full result table for one run.
Filterable by brand, provider, category, brand_mentioned.
Show raw response in expandable row.

### `/recommendations`

`RecommendationList` grouped by priority.
`RecommendationCard` shows: priority badge, category pill, action text, expected impact, rationale, implement button.
"Mark as implemented" button calls `POST /api/recommendations/:id/implement`.
After marking, card shows implementation date and moves to "implemented" section.
Filter: pending | implemented | all.

---

## Section 10 — Identity

The app is called **Project Lighthouse**.

### Sidebar branding
- Lighthouse icon (lucide `Lighthouse` or a custom SVG lighthouse mark)
- "Project Lighthouse" wordmark
- Tagline: "AI Discovery Observatory"

### Nav items (Sidebar)
- Dashboard
- Compare
- Prompts
- Runs
- Recommendations

### Topbar
- Page title (left)
- Brand switcher on Dashboard page only (right): Adore Me | Victoria's Secret pills
- Last run timestamp (right, muted)

### Mobile
- Sidebar collapses to bottom nav on screens < 768px
- Metric cards stack to 2-col grid, then 1-col
- Charts maintain 100% width

---

## Section 11 — Prompt type split rules

**Critical — read before writing any component.**

All `BrandSummary` data from the API is **organic only**.
All `ComparisonSummary` data is **head-to-head only**.

Rules:
- Never display organic and comparison metrics in the same chart or card
- `HeadToHeadSection` and `HeadToHeadCharts` are the ONLY components that render `ComparisonSummary`
- If `useComparisonSummary` returns `null` — render nothing, no empty state
- `MetricCard` for mention rate MUST show `subtitle="organic prompts only"`
- `VisibilityScoreCard` subtitle: "organic visibility score"
- The `/compare` page MUST have a visual section divider between organic and comparison sections

---

## Section 12 — Implementation order

Work tasks in this sequence:

1. Project scaffold — `npx create-next-app`, install dependencies, custom Tailwind colours
2. `lib/types.ts` — all TypeScript interfaces
3. `lib/api.ts` — all fetch functions
4. `lib/utils.ts` — colour maps, formatters, label maps
5. All SWR hooks (`hooks/`)
6. UI primitives — Pill, ProviderDot, Skeleton, EmptyState
7. Layout — Sidebar (with Lighthouse identity), Topbar, PageShell, root layout
8. Charts — ShareOfVoice, Trend, SentimentDonut, CompetitorBars, StabilityHeatmap
9. Dashboard components — VisibilityScoreCard, MetricCard, MetricsRow, PromptResultsTable
10. Contextual components — HeadToHeadSection, ExplainabilityPanel, CitationGapTable
11. LiveRunButton (SSE)
12. Dashboard page — assemble all components
13. Compare page — CompareMetrics, HeadToHeadCharts
14. Prompts page — category tabs, expandable rows
15. Runs pages — list + detail
16. Recommendations page — RecommendationCard, mark as implemented
17. Mobile responsiveness — sidebar collapse, grid breakpoints
18. `app/not-found.tsx`, `app/error.tsx`

---

## Section 13 — Component conventions

- Components do not fetch data — receive typed props and render
- Hooks do not render — fetch and return typed data
- Named exports everywhere except Next.js page files
- No logic in JSX — extract to variables above the return
- No inline styles — Tailwind only
- No hardcoded hex — use `lib/utils.ts` maps
- Loading → `<Skeleton />`
- Error / empty → `<EmptyState />`
- Props interface named `[ComponentName]Props` in the same file

---

## Section 14 — Testing requirements

- Hook test: `useComparisonSummary` with `NO_COMPARISON_DATA` error returns `{ data: null, error: null }`
- Component test: `HeadToHeadSection` renders nothing when data is null
- Component test: `MetricCard` with subtitle prop renders subtitle text
- Component test: `VisibilityScoreCard` ring colour changes at 40 and 70 thresholds
- Component test: `CitationGapTable` sorted by citation_count desc
- Chart test: `TrendChart` receives correct data shape before rendering
- Page test: `/dashboard` has Topbar with brand switcher
- Page test: `/compare` has both organic and head-to-head sections with dividers

---

## Section 15 — Running locally

```bash
npm install
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
npm run dev     # → http://localhost:3000
npm run type-check
npm run lint
npm run build
```

Backend must be running at `NEXT_PUBLIC_API_URL`.
Use `geo-tracker config validate` to verify backend health before starting.

---

## What not to build

- No authentication
- No data mutations except mark-recommendation-implemented
- No direct database access
- No Redux or Zustand
- No CSS-in-JS
- No custom chart drawing
- No i18n

---

## Companion repo

Backend: `geo-tracker-backend`
API spec: see `geo-tracker-backend/CLAUDE.md`

---

*Project Lighthouse · Frontend · Adore Me Tech Hackathon · June 2026*

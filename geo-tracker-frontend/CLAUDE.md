# CLAUDE.md — geo-tracker-frontend

This file tells you everything you need to know to work in this codebase.
Read it fully before writing any code.

---

## What this app does

A Next.js dashboard that visualises GEO (Generative Engine Optimization) brand
visibility data for Adore Me and Victoria's Secret across 4 AI providers.
It is a **read-only dashboard** — it fetches all data from the `geo-tracker-backend`
JSON API. No database access. No authentication.

The companion backend repo is `geo-tracker-backend`.
Backend API base URL is configured via `NEXT_PUBLIC_API_URL`.

---

## Tech stack

| Concern | Library |
|---|---|
| Framework | Next.js 14+ (App Router) |
| Language | TypeScript — strict mode, no `any` |
| Styling | Tailwind CSS — utility classes only |
| Data fetching | SWR |
| Charts | Recharts |
| Linting | ESLint + Prettier |

Do not add dependencies without a clear reason. Prefer what is already installed.

---

## Repository structure

```
geo-tracker-frontend/
├── app/                        # Next.js App Router pages
│   ├── layout.tsx              # root layout, fonts, sidebar
│   ├── page.tsx                # redirects to /dashboard
│   ├── dashboard/page.tsx      # main KPI + charts view
│   ├── compare/page.tsx        # side-by-side brand comparison
│   ├── prompts/page.tsx        # prompt library + results
│   └── runs/
│       ├── page.tsx            # run history list
│       └── [id]/page.tsx       # single run detail
├── components/
│   ├── layout/                 # Sidebar, Topbar, PageShell
│   ├── charts/                 # Recharts wrappers
│   ├── dashboard/              # MetricCard, PromptResultsTable, etc.
│   ├── compare/                # CompareMetrics, CompareCharts
│   └── ui/                     # Pill, ProviderDot, Skeleton, EmptyState
├── hooks/                      # SWR data hooks — one per API endpoint
├── lib/
│   ├── api.ts                  # all fetch functions — single source of truth
│   ├── types.ts                # TypeScript interfaces matching backend DTOs
│   └── utils.ts                # formatters, colour maps, label maps
├── styles/globals.css          # Tailwind directives only
├── .env.example                # committed — shows required env vars
├── .env.local                  # gitignored — local values
├── tailwind.config.ts
├── tsconfig.json
└── next.config.ts
```

---

## Environment variables

```bash
# .env.example — only one variable required
NEXT_PUBLIC_API_URL=http://localhost:8080
```

In Kubernetes, `NEXT_PUBLIC_API_URL` is set to the internal backend service URL.
No secrets — the frontend only reads public data from the backend.

---

## Pages and what they show

| Route | Purpose |
|---|---|
| `/dashboard` | Main view — metric cards, charts, prompt table. Brand switcher in topbar. |
| `/compare` | Side-by-side Adore Me vs Victoria's Secret — always shows both brands. |
| `/prompts` | Prompt library with category filter tabs. Expandable rows show provider results. |
| `/runs` | Paginated run history. Click a run to see full result breakdown. |

---

## Data fetching rules

**All fetch calls live in `lib/api.ts`.**
No inline `fetch()` calls anywhere in components or pages.

**All client-side data fetching uses SWR hooks from `hooks/`.**
One hook per API endpoint. Hooks return `{ data, isLoading, error }` typed correctly.

**Server Components may fetch directly** when data does not need to be reactive
(e.g. initial prompt list). Use `async/await` with `lib/api.ts` functions.

**Always handle three states:**
- `isLoading` → render `<Skeleton />`
- `error` → render `<EmptyState />` with error message
- `data` → render the component

**Refresh interval:** set `refreshInterval: 30_000` on all dashboard hooks so the
page auto-updates while a run is in progress.

**Brand state** is held in the URL search param `?brand=adoreme` — not in React state.
This makes dashboard URLs shareable.

```ts
// lib/api.ts — shape of all fetch functions
const BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

export const api = {
  summary:     (brand: string)         => fetch(`${BASE}/api/brands/${enc(brand)}/summary`),
  trend:       (brand: string, n = 10) => fetch(`${BASE}/api/brands/${enc(brand)}/trend?runs=${n}`),
  compare:     (a: string, b: string)  => fetch(`${BASE}/api/compare?brands=${enc(a)},${enc(b)}`),
  competitors: (brand: string)         => fetch(`${BASE}/api/competitors?brand=${enc(brand)}`),
  prompts:     ()                      => fetch(`${BASE}/api/prompts`),
  runs:        (page = 1)              => fetch(`${BASE}/api/runs?page=${page}&per_page=20`),
  runDetail:   (id: number)            => fetch(`${BASE}/api/runs/${id}/results`),
  health:      ()                      => fetch(`${BASE}/api/health`),
}

const enc = encodeURIComponent
```

---

## TypeScript rules

- Strict mode is on — `"strict": true` in `tsconfig.json`
- No `any` — ever. Use `unknown` and narrow, or define the correct type
- All component props have a named interface: `[ComponentName]Props`
- All hook return values are typed — no implicit `any` from SWR
- Types that match backend DTOs live in `lib/types.ts` — not scattered across files
- Enums as union string literals — not TypeScript `enum` keyword

```ts
// correct
type Sentiment = 'positive' | 'neutral' | 'negative' | 'not_mentioned'

// wrong
enum Sentiment { positive, neutral, negative, not_mentioned }
```

---

## Component rules

**Components do not fetch data.** They receive typed props and render.
**Hooks do not render.** They fetch and return typed data.

```
Page → hook (SWR) → component → ui primitive
```

- Named exports everywhere except Next.js page files (which require default export)
- No logic in JSX — extract to a variable or helper function above the return
- No inline styles — Tailwind classes only
- No hardcoded colours — use the custom colour tokens from `tailwind.config.ts`
- Loading states use `<Skeleton />` — not spinners or "Loading..." text
- Empty / error states use `<EmptyState />` with a descriptive message

---

## Colour system

All colours are defined as custom tokens in `tailwind.config.ts`.
Reference by class name — never hardcode hex values in components.

```ts
// tailwind.config.ts
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

Utility maps in `lib/utils.ts` return the right Tailwind class or hex for a
given provider/sentiment/category — use these in components, not switch statements.

```ts
// lib/utils.ts
export const providerColour: Record<Provider, string> = {
  claude:     '#AFA9EC',
  chatgpt:    '#5DCAA5',
  perplexity: '#EF9F27',
  gemini:     '#F0997B',
}

export const sentimentClass: Record<Sentiment, string> = {
  positive:      'bg-sentiment-positive text-white',
  neutral:       'bg-sentiment-neutral text-gray-700',
  negative:      'bg-sentiment-negative text-white',
  not_mentioned: 'bg-gray-100 text-gray-400',
}

export const categoryClass: Record<PromptCategory, string> = {
  purchase:   'bg-[#EEEDFE] text-[#3C3489]',
  discovery:  'bg-[#E1F5EE] text-[#085041]',
  comparison: 'bg-[#FAEEDA] text-[#633806]',
  fit:        'bg-[#FAECE7] text-[#712B13]',
  gifting:    'bg-[#E6F1FB] text-[#0C447C]',
}
```

---

## Chart rules (Recharts)

- Every chart is wrapped in `<ResponsiveContainer width="100%" height={N}>` — no fixed pixel widths
- Charts receive **pre-shaped data as props** — no data transformation inside chart components
- Data shaping happens in the hook or the page, not inside the chart
- Tooltip values append `%` for rate metrics — use `formatter` prop
- Axis tick formatting uses helpers from `lib/utils.ts`
- Provider colours always come from `providerColour` map in `lib/utils.ts`
- Legend labels use human-readable names, not raw provider keys

### Required data shapes (shape data to these before passing to charts)

```ts
// ShareOfVoiceChart
type SOVDatum = { provider: string; rate: number }

// TrendChart
type TrendDatum = { run_at: string; [brand: string]: number | string }

// SentimentDonut
type SentimentDatum = { name: string; value: number; color: string }

// CompetitorBars
type CompetitorDatum = { name: string; frequency: number }
```

---

## Routing and URL state

- Brand selection: `?brand=adoreme` or `?brand=vs` — read with `useSearchParams()`
- Pagination: `?page=1` — read with `useSearchParams()`
- Run detail: `/runs/[id]` — dynamic segment
- No other state in the URL — filters that don't need to be shareable use `useState`

---

## What goes where

| Decision | Where |
|---|---|
| Fetch URL construction | `lib/api.ts` |
| TypeScript types | `lib/types.ts` |
| Colour/label maps, formatters | `lib/utils.ts` |
| SWR fetching + cache key | `hooks/use*.ts` |
| Data shaping for charts | hook or page, before passing as props |
| Rendering | `components/**` |
| Page assembly | `app/**/page.tsx` |
| Loading UI | `app/**/loading.tsx` |

If you are unsure where something goes, put it one level closer to `lib/` than you think.

---

## Formatting and linting

- Prettier for formatting — run `npm run format` before committing
- ESLint with Next.js rules — run `npm run lint` before committing
- No unused imports, no unused variables
- Import order: React → Next.js → third-party → internal (`lib/` → `hooks/` → `components/`)
- Trailing commas in all multiline expressions

---

## Running locally

```bash
# Install
npm install

# Configure backend URL
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local

# Dev server
npm run dev
# → http://localhost:3000

# Type check
npm run type-check

# Lint
npm run lint

# Build
npm run build
```

The backend (`geo-tracker serve`) must be running at `NEXT_PUBLIC_API_URL`.
Use `geo-tracker config validate` to confirm the backend is healthy before starting.

---

## What not to build

- No authentication or login screen — internal tool on a private network
- No data mutations — read-only dashboard, no forms that POST to the backend
- No direct database access — everything through the backend API
- No Redux, Zustand, or other state managers — SWR + URL params is sufficient
- No CSS-in-JS or styled-components — Tailwind only
- No custom chart drawing — Recharts handles all visualisation
- No i18n — English only

---

## Companion repo

Backend: `geo-tracker-backend`
API spec: see `geo-tracker-backend/CLAUDE.md` → serve command section

The full list of backend endpoints this frontend consumes:

```
GET /api/health
GET /api/runs
GET /api/runs/:id/results
GET /api/brands/:brand/summary
GET /api/brands/:brand/trend
GET /api/compare?brands=A,B
GET /api/competitors?brand=X
GET /api/prompts
GET /api/prompts/:id/results
```

---

*GEO Tracker Frontend · Adore Me Tech · June 2026*

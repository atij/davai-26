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
  liveRun:             (promptId: number, providers: string[]) =>
    new EventSource(`${BASE}/api/runs/live?prompt_id=${promptId}&providers=${providers.join(',')}`),
}

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

export interface RunTrace {
  id:          number
  run_id:      number
  phase:       string
  agent_name:  string
  started_at:  string
  finished_at: string | null
  duration_ms: number | null
  status:      'running' | 'success' | 'error' | 'retried'
  error_text:  string | null
}

export type ChatEventType = 'chunk' | 'tool_call' | 'tool_result' | 'done' | 'error'

export interface ChatEvent {
  type:    ChatEventType
  text?:   string
  tool?:   string
  args?:   any
  preview?: string
  error?:  string
}

// Chart data shapes — shape data to these before passing to chart components
export type SOVDatum        = { provider: string; rate: number }
export type TrendDatum      = { run_at: string; [brand: string]: number | string }
export type SentimentDatum  = { name: string; value: number; color: string }
export type CompetitorDatum = { name: string; frequency: number }
export type StabilityDatum  = { prompt: string; provider: Provider; score: number }


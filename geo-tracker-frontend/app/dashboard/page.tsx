"use client"

import { useSearchParams } from "next/navigation"
import { Topbar } from "@/components/layout/Topbar"
import { PageShell } from "@/components/layout/PageShell"
import { MetricsRow } from "@/components/dashboard/MetricsRow"
import { VisibilityScoreCard } from "@/components/dashboard/VisibilityScoreCard"
import { ShareOfVoiceChart } from "@/components/charts/ShareOfVoiceChart"
import { TrendChart } from "@/components/charts/TrendChart"
import { CompetitorBars } from "@/components/charts/CompetitorBars"
import { SentimentDonut } from "@/components/charts/SentimentDonut"
import { CitationGapTable } from "@/components/dashboard/CitationGapTable"
import { PromptResultsTable } from "@/components/dashboard/PromptResultsTable"
import { useSummary } from "@/hooks/useSummary"
import { useTrend } from "@/hooks/useTrend"
import { useCompetitors } from "@/hooks/useCompetitors"
import { useRunDetail } from "@/hooks/useRunDetail"
import { Skeleton } from "@/components/ui/Skeleton"
import { EmptyState } from "@/components/ui/EmptyState"
import { AlertCircle } from "lucide-react"
import { Suspense } from "react"
import { providerColour } from "@/lib/utils"

function DashboardContent() {
  const searchParams = useSearchParams()
  const brandKey = searchParams.get("brand") || "adoreme"
  const brandName = brandKey === "adoreme" ? "Adore Me" : "Victoria's Secret"
  
  const { data: summary, isLoading: sumLoading, error: sumError } = useSummary(brandName)
  const { trend, isLoading: trendLoading } = useTrend(brandName)
  const { competitors, isLoading: compLoading } = useCompetitors(brandName)
  const { results, isLoading: resLoading } = useRunDetail(summary?.run_id || 0)

  if (sumError) {
    return (
      <PageShell>
        <EmptyState 
          title="Error loading dashboard" 
          message="We couldn't fetch the data for this brand. Please try again later." 
          icon={<AlertCircle size={48} />}
        />
      </PageShell>
    )
  }

  const sovData = summary?.provider_rates ? Object.entries(summary.provider_rates).map(([provider, rate]) => ({
    provider,
    rate
  })) : []

  const trendData = trend ? trend.map(p => ({
    run_at: p.run_at,
    [brandName]: p.mention_rate
  })) : []

  const sentimentData = summary ? [
    { name: "Positive", value: (summary as any).sentiment_positive || 0, color: '#5DCAA5' },
    { name: "Neutral", value: (summary as any).sentiment_neutral || 0, color: '#B4B2A9' },
    { name: "Negative", value: (summary as any).sentiment_negative || 0, color: '#F09595' },
  ].filter(d => d.value > 0) : []

  return (
    <>
      <Topbar title="Dashboard" />
      <PageShell>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
          <div className="lg:col-span-1">
             {sumLoading ? <Skeleton className="h-full w-full rounded-3xl" /> : 
              <VisibilityScoreCard score={summary?.visibility_score || 0} brand={brandName} />
             }
          </div>
          <div className="lg:col-span-2">
            {sumLoading ? <Skeleton className="h-full w-full rounded-3xl" /> : 
              summary && <MetricsRow summary={summary} />
            }
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
            <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8 text-center">Share of Voice</h3>
            {sumLoading ? <Skeleton className="h-[300px] rounded-xl" /> : <ShareOfVoiceChart data={sovData} />}
          </section>
          
          <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
            <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8 text-center">Mention Trend</h3>
            {trendLoading ? <Skeleton className="h-[300px] rounded-xl" /> : (
              <TrendChart 
                data={trendData} 
                brands={[brandName]} 
              />
            )}
          </section>

          <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
            <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8 text-center">Top Competitors</h3>
            {compLoading ? <Skeleton className="h-[300px] rounded-xl" /> : <CompetitorBars data={competitors || []} />}
          </section>

          <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
            <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8 text-center">Sentiment Split</h3>
            {sumLoading ? <Skeleton className="h-[300px] rounded-xl" /> : <SentimentDonut data={sentimentData} />}
          </section>
        </div>

        <div className="mb-8">
          <CitationGapTable brand={brandName} />
        </div>

        <section>
          <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-6 px-1">Latest Prompt Results</h3>
          {resLoading ? <Skeleton className="h-64 rounded-3xl" /> : results && <PromptResultsTable results={results as any} />}
        </section>
      </PageShell>
    </>
  )
}

export default function DashboardPage() {
  return (
    <Suspense fallback={<Skeleton className="h-screen w-full" />}>
      <DashboardContent />
    </Suspense>
  )
}

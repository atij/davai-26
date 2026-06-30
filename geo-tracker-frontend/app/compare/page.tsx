"use client"

import React from 'react'
import { Topbar } from '@/components/layout/Topbar'
import { PageShell } from '@/components/layout/PageShell'
import { CompareMetrics } from '@/components/compare/CompareMetrics'
import { HeadToHeadCharts } from '@/components/compare/HeadToHeadCharts'
import { TrendChart } from '@/components/charts/TrendChart'
import { useSummary } from '@/hooks/useSummary'
import { useTrend } from '@/hooks/useTrend'
import { useComparisonSummary } from '@/hooks/useComparisonSummary'
import { Skeleton } from '@/components/ui/Skeleton'

export default function ComparePage() {
  const brandA = "Adore Me"
  const brandB = "Victoria's Secret"

  const { data: sumA, isLoading: loadingSumA } = useSummary(brandA)
  const { data: sumB, isLoading: loadingSumB } = useSummary(brandB)
  const { trend: trendA, isLoading: loadingTrendA } = useTrend(brandA)
  const { trend: trendB, isLoading: loadingTrendB } = useTrend(brandB)
  const { data: compSumA, isLoading: loadingCompA } = useComparisonSummary(brandA)
  const { data: compSumB, isLoading: loadingCompB } = useComparisonSummary(brandB)

  const isLoading = loadingSumA || loadingSumB || loadingTrendA || loadingTrendB

  // Merge trend data
  const combinedTrend = trendA?.map(p => {
      const match = trendB?.find(pb => pb.run_id === p.run_id)
      return {
          run_at: p.run_at,
          [brandA]: p.mention_rate,
          [brandB]: match ? match.mention_rate : 0
      }
  }) || []

  return (
    <>
      <Topbar title="Side-by-Side Comparison" />
      <PageShell>
        <section className="mb-16">
          <div className="flex items-center gap-4 mb-10">
            <h2 className="text-xl font-black text-slate-900 tracking-tight uppercase">Organic Visibility</h2>
            <div className="h-px flex-1 bg-slate-200" />
            <span className="text-[10px] font-black text-slate-400 uppercase tracking-[0.2em] bg-slate-100 px-4 py-1.5 rounded-full border border-slate-200 shadow-sm">Organic Only</span>
          </div>
          
          {isLoading ? (
            <div className="space-y-8">
              <Skeleton className="h-64 rounded-3xl" />
              <Skeleton className="h-96 rounded-3xl" />
            </div>
          ) : (
            <div className="space-y-12">
              {sumA && sumB && <CompareMetrics summaryA={sumA} summaryB={sumB} />}
              
              <div className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
                <h3 className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-8">Aggregated Mention Trend</h3>
                <TrendChart data={combinedTrend} brands={[brandA, brandB]} />
              </div>
            </div>
          )}
        </section>

        {(loadingCompA || loadingCompB || (compSumA && compSumB)) && (
          <section className="mb-16">
            <div className="flex items-center gap-4 mb-10">
              <h2 className="text-xl font-black text-slate-900 tracking-tight uppercase">Head-to-Head Analysis</h2>
              <div className="h-px flex-1 bg-slate-200" />
              <span className="text-[10px] font-black text-slate-400 uppercase tracking-[0.2em] bg-slate-100 px-4 py-1.5 rounded-full border border-slate-200 shadow-sm">Comparison Only</span>
            </div>
            
            {loadingCompA || loadingCompB ? (
              <Skeleton className="h-96 rounded-3xl" />
            ) : (
              compSumA && compSumB && <HeadToHeadCharts summaryA={compSumA} summaryB={compSumB} />
            )}
          </section>
        )}
      </PageShell>
    </>
  )
}

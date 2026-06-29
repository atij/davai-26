import React from 'react'
import { useComparisonSummary } from '@/hooks/useComparisonSummary'
import { MetricCard } from './MetricCard'
import { Skeleton } from '@/components/ui/Skeleton'
import { formatPercent, formatScore } from '@/lib/api' // Note: formatters were in utils in spec but suggested api in some snippets. Let's check. Actually spec said utils.

// Actually re-checking spec Section 5 says formatPercent is in lib/utils.ts
import { formatPercent as fp } from '@/lib/utils'

interface HeadToHeadSectionProps {
  brand: string
}

export const HeadToHeadSection: React.FC<HeadToHeadSectionProps> = ({ brand }) => {
  const { data: summary, isLoading, error } = useComparisonSummary(brand)

  if (isLoading) return <Skeleton className="h-64 w-full rounded-3xl mb-8" />
  if (error || !summary) return null

  return (
    <section className="mb-8">
      <div className="flex items-center gap-3 mb-6">
        <h2 className="text-lg font-black text-slate-900 uppercase tracking-tight">Head-to-Head</h2>
        <span className="text-[10px] font-bold text-slate-400 bg-slate-100 px-2 py-1 rounded-full uppercase tracking-widest">Comparison Prompts Only</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <MetricCard 
          label="Direct Win Rate" 
          value={fp(summary.win_rate)} 
          up={summary.win_rate > 50}
          subtitle="Preferred over competitors"
        />
        <MetricCard 
          label="Head-to-Head Mention" 
          value={fp(summary.mention_rate)} 
          subtitle="Frequency in comparison"
        />
        <MetricCard 
          label="Avg. Recommendation Rank" 
          value={summary.avg_rank ? summary.avg_rank.toFixed(1) : 'N/A'} 
          subtitle="Lower is better (1st choice)"
        />
      </div>
    </section>
  )
}

import React from 'react'
import { BrandSummary } from '@/lib/types'
import { MetricCard } from './MetricCard'
import { formatPercent, formatScore } from '@/lib/utils'
import { Activity, Star, Zap, BarChart2 } from 'lucide-react'

interface MetricsRowProps {
  summary: BrandSummary
}

export const MetricsRow: React.FC<MetricsRowProps> = ({ summary }) => {
  if (!summary) return null;
  
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <MetricCard
        label="Mention Rate"
        value={formatPercent(summary?.mention_rate)}
        subtitle="organic prompts only"
        icon={<Activity size={20} />}
      />
      <MetricCard
        label="First Rec. Rate"
        value={formatPercent(summary?.first_rec_rate)}
        icon={<Zap size={20} />}
      />
      <MetricCard
        label="Sentiment Score"
        value={formatScore(summary?.sentiment_score)}
        sub="(-1 to 1)"
        icon={<Star size={20} />}
      />
      <MetricCard
        label="Stability Score"
        value={formatScore(summary?.stability_score)}
        icon={<BarChart2 size={20} />}
      />
    </div>
  )
}

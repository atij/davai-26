"use client"

import React from 'react'
import { BrandSummary, ComparisonSummary } from '@/lib/types'
import { MetricsRow } from '@/components/dashboard/MetricsRow'
import { formatPercent } from '@/lib/utils'

interface CompareMetricsProps {
  summaryA: BrandSummary
  summaryB: BrandSummary
}

export const CompareMetrics: React.FC<CompareMetricsProps> = ({ summaryA, summaryB }) => {
  return (
    <div className="grid grid-cols-1 xl:grid-cols-2 gap-12">
      <div>
        <div className="flex items-center gap-3 mb-6">
           <div className="w-3 h-3 rounded-full bg-[#7F77DD]" />
           <h3 className="text-lg font-bold text-slate-900">{summaryA.brand}</h3>
        </div>
        <MetricsRow summary={summaryA} />
      </div>
      <div>
        <div className="flex items-center gap-3 mb-6">
           <div className="w-3 h-3 rounded-full bg-[#1D9E75]" />
           <h3 className="text-lg font-bold text-slate-900">{summaryB.brand}</h3>
        </div>
        <MetricsRow summary={summaryB} />
      </div>
    </div>
  )
}

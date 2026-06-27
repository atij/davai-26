"use client"

import React from 'react'
import { ComparisonSummary } from '@/lib/types'
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  Legend
} from 'recharts'

interface HeadToHeadChartsProps {
  summaryA: ComparisonSummary
  summaryB: ComparisonSummary
}

export const HeadToHeadCharts: React.FC<HeadToHeadChartsProps> = ({ summaryA, summaryB }) => {
  const providers = Object.keys(summaryA.provider_breakdown) as (keyof typeof summaryA.provider_breakdown)[]
  
  const winRateData = providers.map(p => ({
    provider: p.charAt(0).toUpperCase() + p.slice(1),
    [summaryA.brand]: summaryA.provider_breakdown[p].win_rate,
    [summaryB.brand]: summaryB.provider_breakdown[p].win_rate,
  }))

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <div className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
        <h3 className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-8">Win Rate by Provider</h3>
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={winRateData}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
              <XAxis dataKey="provider" axisLine={false} tickLine={false} tick={{ fill: '#94a3b8', fontSize: 12 }} />
              <YAxis axisLine={false} tickLine={false} tick={{ fill: '#94a3b8', fontSize: 12 }} tickFormatter={v => `${v}%`} />
              <Tooltip 
                contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1)' }}
                formatter={(v: number) => [`${v.toFixed(1)}%`, 'Win Rate']}
              />
              <Legend verticalAlign="top" align="right" iconType="circle" />
              <Bar dataKey={summaryA.brand} fill="#7F77DD" radius={[4, 4, 0, 0]} />
              <Bar dataKey={summaryB.brand} fill="#1D9E75" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm">
        <h3 className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-8">Comparison Metrics</h3>
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="p-6 bg-slate-50 rounded-2xl">
              <p className="text-[10px] font-bold text-slate-400 uppercase mb-1">Overall Win Rate</p>
              <div className="flex items-baseline gap-2">
                <span className="text-2xl font-bold text-slate-900">{summaryA.win_rate.toFixed(1)}%</span>
                <span className="text-xs text-slate-400">vs {summaryB.win_rate.toFixed(1)}%</span>
              </div>
            </div>
            <div className="p-6 bg-slate-50 rounded-2xl">
              <p className="text-[10px] font-bold text-slate-400 uppercase mb-1">Avg Rank</p>
              <div className="flex items-baseline gap-2">
                <span className="text-2xl font-bold text-slate-900">{summaryA.avg_rank?.toFixed(1) || '-'}</span>
                <span className="text-xs text-slate-400">vs {summaryB.avg_rank?.toFixed(1) || '-'}</span>
              </div>
            </div>
          </div>
          <p className="text-xs text-slate-400 leading-relaxed italic">
            * Comparison metrics are derived exclusively from head-to-head prompts where both brands are explicitly compared.
          </p>
        </div>
      </div>
    </div>
  )
}

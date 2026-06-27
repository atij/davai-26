"use client"

import React from 'react'
import { Topbar } from '@/components/layout/Topbar'
import { PageShell } from '@/components/layout/PageShell'
import { useRuns } from '@/hooks/useRuns'
import { Skeleton } from '@/components/ui/Skeleton'
import { Pill } from '@/components/ui/Pill'
import { formatDate, formatCost } from '@/lib/utils'
import Link from 'next/link'
import { ChevronRight, Calendar, Clock, Database, DollarSign } from 'lucide-react'

export default function RunsPage() {
  const { runs, isLoading } = useRuns()

  return (
    <>
      <Topbar title="Execution History" />
      <PageShell>
        <div className="bg-white rounded-3xl border border-slate-200 overflow-hidden shadow-sm">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50/50 border-b border-slate-100">
                <th className="px-8 py-5 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Run ID</th>
                <th className="px-8 py-5 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Date</th>
                <th className="px-8 py-5 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Status</th>
                <th className="px-8 py-5 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-center">Workload</th>
                <th className="px-8 py-5 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-right">Cost</th>
                <th className="w-10 px-8 py-5"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {isLoading ? (
                Array(5).fill(0).map((_, i) => (
                  <tr key={i}><td colSpan={6} className="p-4"><Skeleton className="h-14 w-full" /></td></tr>
                ))
              ) : (
                runs?.map((run) => (
                  <tr key={run.id} className="group hover:bg-slate-50/30 transition-colors">
                    <td className="px-8 py-5">
                      <span className="text-sm font-black text-slate-900 tracking-tight">#{run.id}</span>
                    </td>
                    <td className="px-8 py-5">
                       <div className="flex flex-col">
                           <span className="text-sm font-semibold text-slate-700">{formatDate(run.started_at)}</span>
                           <span className="text-[10px] text-slate-400 font-medium flex items-center gap-1 mt-0.5">
                               <Clock size={10} /> {run.duration_seconds ? `${Math.floor(run.duration_seconds / 60)}m ${run.duration_seconds % 60}s` : 'N/A'}
                           </span>
                       </div>
                    </td>
                    <td className="px-8 py-5">
                      <Pill variant={run.status} />
                    </td>
                    <td className="px-8 py-5 text-center">
                       <div className="flex items-center justify-center gap-2">
                           <div className="flex flex-col items-center">
                               <span className="text-xs font-bold text-slate-900">{run.prompt_count}</span>
                               <span className="text-[8px] font-bold text-slate-400 uppercase tracking-tighter">Prompts</span>
                           </div>
                           <div className="w-px h-6 bg-slate-100" />
                           <div className="flex flex-col items-center">
                               <span className="text-xs font-bold text-slate-900">{run.sample_count}</span>
                               <span className="text-[8px] font-bold text-slate-400 uppercase tracking-tighter">Samples</span>
                           </div>
                       </div>
                    </td>
                    <td className="px-8 py-5 text-right">
                       <span className="text-sm font-mono font-bold text-slate-700">
                           {run.total_cost_usd ? formatCost(run.total_cost_usd) : '$0.00'}
                       </span>
                    </td>
                    <td className="px-8 py-5">
                       <Link 
                         href={`/runs/${run.id}`}
                         className="p-2 bg-slate-50 rounded-xl text-slate-400 group-hover:bg-indigo-600 group-hover:text-white transition-all flex items-center justify-center"
                       >
                           <ChevronRight size={16} />
                       </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </PageShell>
    </>
  )
}

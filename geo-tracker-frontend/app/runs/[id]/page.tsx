"use client"

import React, { useState, use } from 'react'
import { Topbar } from '@/components/layout/Topbar'
import { PageShell } from '@/components/layout/PageShell'
import { useRunDetail } from '@/hooks/useRunDetail'
import { Skeleton } from '@/components/ui/Skeleton'
import { Pill } from '@/components/ui/Pill'
import { ProviderDot } from '@/components/ui/ProviderDot'
import { capitalize, cn } from '@/lib/utils'
import { ChevronLeft, Filter, Search, ChevronDown, ChevronRight, Info, Activity } from 'lucide-react'
import Link from 'next/link'
import { RunTraceTimeline } from '@/components/runs/RunTraceTimeline'

export default function RunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const runId = parseInt(id)
  const { results, isLoading } = useRunDetail(runId)
  
  const [filterBrand, setFilterBrand] = useState<string>('all')
  const [filterProvider, setFilterProvider] = useState<string>('all')
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [showTrace, setShowTrace] = useState(false)

  const brands = Array.from(new Set(results?.map(r => r.brand) || []))
  const providers = ['claude', 'chatgpt', 'perplexity', 'gemini']

  const filteredResults = results?.filter(r => {
      const matchBrand = filterBrand === 'all' || r.brand === filterBrand
      const matchProvider = filterProvider === 'all' || r.provider === filterProvider
      return matchBrand && matchProvider
  }) || []

  return (
    <>
      <Topbar 
        title={`Run Detail #${id}`} 
        leftContent={
            <Link href="/runs" className="mr-4 p-2 hover:bg-slate-100 rounded-xl transition-colors">
                <ChevronLeft size={20} className="text-slate-600" />
            </Link>
        }
      />
      <PageShell>
        <div className="flex flex-wrap items-center justify-between gap-4 mb-8">
            <div className="flex items-center gap-4">
                <div className="flex items-center gap-2 bg-white px-4 py-2 rounded-xl border border-slate-200 shadow-sm">
                    <Filter size={14} className="text-slate-400" />
                    <select 
                      className="text-xs font-bold text-slate-600 outline-none bg-transparent"
                      value={filterBrand}
                      onChange={(e) => setFilterBrand(e.target.value)}
                    >
                        <option value="all">All Brands</option>
                        {brands.map(b => <option key={b} value={b}>{b}</option>)}
                    </select>
                </div>
                <div className="flex items-center gap-2 bg-white px-4 py-2 rounded-xl border border-slate-200 shadow-sm">
                    <div className="w-2 h-2 rounded-full bg-indigo-600" />
                    <select 
                      className="text-xs font-bold text-slate-600 outline-none bg-transparent"
                      value={filterProvider}
                      onChange={(e) => setFilterProvider(e.target.value)}
                    >
                        <option value="all">All Providers</option>
                        {providers.map(p => <option key={p} value={p}>{capitalize(p)}</option>)}
                    </select>
                </div>
                <button 
                  onClick={() => setShowTrace(!showTrace)}
                  className={cn(
                    "flex items-center gap-2 px-4 py-2 rounded-xl border transition-all text-xs font-bold",
                    showTrace 
                      ? "bg-indigo-600 text-white border-indigo-600 shadow-lg shadow-indigo-100" 
                      : "bg-white text-slate-600 border-slate-200 shadow-sm hover:border-slate-300"
                  )}
                >
                    <Activity size={14} />
                    {showTrace ? 'Hide Agent Graph' : 'Show Agent Graph'}
                </button>
            </div>
            
            <div className="text-xs font-bold text-slate-400 uppercase tracking-widest">
                Showing {filteredResults.length} Results
            </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 items-start">
            <div className={cn("lg:col-span-2", !showTrace && "lg:col-span-3")}>
                <div className="bg-white rounded-3xl border border-slate-200 overflow-hidden shadow-sm">
                   <table className="w-full text-left border-collapse">
               <thead>
                   <tr className="bg-slate-50/50 border-b border-slate-100">
                        <th className="w-10 px-6 py-4"></th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Brand</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Provider</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Mentioned</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Sentiment</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Rank</th>
                   </tr>
               </thead>
               <tbody className="divide-y divide-slate-50">
                    {isLoading ? (
                        Array(8).fill(0).map((_, i) => (
                            <tr key={i}><td colSpan={6} className="p-4"><Skeleton className="h-10 w-full" /></td></tr>
                        ))
                    ) : (
                        filteredResults.map((res, i) => (
                            <React.Fragment key={i}>
                                <tr 
                                  className={cn(
                                      "hover:bg-slate-50/30 transition-colors cursor-pointer",
                                      expandedId === i && "bg-slate-50/50"
                                  )}
                                  onClick={() => setExpandedId(expandedId === i ? null : i)}
                                >
                                    <td className="px-6 py-4">
                                        {expandedId === i ? <ChevronDown size={14} className="text-slate-400" /> : <ChevronRight size={14} className="text-slate-400" />}
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className="text-sm font-bold text-slate-900">{res.brand}</span>
                                    </td>
                                    <td className="px-6 py-4 text-xs font-bold text-slate-500">
                                        <div className="flex items-center gap-2">
                                            <ProviderDot provider={res.provider as any} hit={true} />
                                            {capitalize(res.provider)}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <div className={cn(
                                            "w-3 h-3 rounded-full",
                                            res.brand_mentioned ? "bg-emerald-500" : "bg-red-400"
                                        )} />
                                    </td>
                                    <td className="px-6 py-4">
                                        <Pill variant={res.sentiment as any} />
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className="text-sm font-mono font-bold text-slate-700">
                                            {res.recommendation_rank || '-'}
                                        </span>
                                    </td>
                                </tr>
                                {expandedId === i && (
                                    <tr className="bg-slate-50/30">
                                        <td colSpan={6} className="px-12 py-8">
                                            <div className="space-y-6 max-w-4xl">
                                                <div className="space-y-2">
                                                    <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Raw Provider Response</p>
                                                    <div className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm text-sm text-slate-600 leading-relaxed overflow-auto max-h-[300px] whitespace-pre-wrap">
                                                        {(res as any).raw_response}
                                                    </div>
                                                </div>
                                                
                                                <div className="grid grid-cols-2 gap-8">
                                                    <div className="space-y-2">
                                                        <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Competitors Mentioned</p>
                                                        <div className="flex flex-wrap gap-2">
                                                            {res.competitors_mentioned?.length > 0 ? (
                                                                res.competitors_mentioned.map(c => (
                                                                    <span key={c} className="px-2 py-1 bg-slate-100 rounded-md text-[10px] font-bold text-slate-500">{c}</span>
                                                                ))
                                                            ) : <span className="text-xs italic text-slate-400">None detected</span>}
                                                        </div>
                                                    </div>
                                                    <div className="space-y-2">
                                                        <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Cited URLs</p>
                                                        <div className="flex flex-col gap-1">
                                                            {res.cited_urls?.length > 0 ? (
                                                                res.cited_urls.map(u => (
                                                                    <a key={u} href={u} target="_blank" rel="noreferrer" className="text-[10px] font-bold text-indigo-600 truncate hover:underline">{u}</a>
                                                                ))
                                                            ) : <span className="text-xs italic text-slate-400">None detected</span>}
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        </td>
                                    </tr>
                                )}
                            </React.Fragment>
                        ))
                    )}
               </tbody>
           </table>
        </div>
            </div>
            {showTrace && (
                <div className="lg:col-span-1 sticky top-24">
                    <RunTraceTimeline runId={runId} />
                </div>
            )}
        </div>
      </PageShell>
    </>
  )
}

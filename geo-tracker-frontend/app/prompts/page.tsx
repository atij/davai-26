"use client"

import React, { useState } from 'react'
import { Topbar } from '@/components/layout/Topbar'
import { PageShell } from '@/components/layout/PageShell'
import { usePrompts } from '@/hooks/usePrompts'
import { useStability } from '@/hooks/useStability'
import { Pill } from '@/components/ui/Pill'
import { Skeleton } from '@/components/ui/Skeleton'
import { PromptCategory, Provider } from '@/lib/types'
import { cn, capitalize, providerColour } from '@/lib/utils'
import { ChevronRight, ChevronDown, Activity } from 'lucide-react'

export default function PromptsPage() {
  const { prompts, isLoading: loadingPrompts } = usePrompts()
  const [activeCategory, setActiveCategory] = useState<PromptCategory | 'all'>('all')
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const categories: (PromptCategory | 'all')[] = ['all', 'purchase', 'discovery', 'fit', 'comparison', 'gifting']

  const filteredPrompts = prompts?.filter(p => activeCategory === 'all' || p.category === activeCategory) || []

  return (
    <>
      <Topbar title="Prompt Library" />
      <PageShell>
        <div className="flex items-center gap-2 mb-8 overflow-x-auto pb-2 no-scrollbar">
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              className={cn(
                "px-5 py-2 rounded-xl text-xs font-bold uppercase tracking-widest transition-all whitespace-nowrap",
                activeCategory === cat 
                  ? "bg-indigo-600 text-white shadow-lg shadow-indigo-100" 
                  : "bg-white text-slate-400 border border-slate-200 hover:border-slate-300"
              )}
            >
              {cat}
            </button>
          ))}
        </div>

        <div className="bg-white rounded-3xl border border-slate-200 overflow-hidden shadow-sm">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50/50 border-b border-slate-100">
                <th className="w-10 px-6 py-4"></th>
                <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">ID</th>
                <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Category</th>
                <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Prompt Text</th>
                <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {loadingPrompts ? (
                Array(5).fill(0).map((_, i) => (
                  <tr key={i}><td colSpan={5} className="p-4"><Skeleton className="h-12 w-full" /></td></tr>
                ))
              ) : (
                filteredPrompts.map(prompt => (
                  <React.Fragment key={prompt.id}>
                    <tr 
                      className={cn(
                        "hover:bg-slate-50/30 transition-colors cursor-pointer",
                        expandedId === prompt.id && "bg-slate-50/50"
                      )}
                      onClick={() => setExpandedId(expandedId === prompt.id ? null : prompt.id)}
                    >
                      <td className="px-6 py-4">
                        {expandedId === prompt.id ? <ChevronDown size={16} className="text-slate-400" /> : <ChevronRight size={16} className="text-slate-400" />}
                      </td>
                      <td className="px-6 py-4 text-xs font-bold text-slate-400">#{prompt.id}</td>
                      <td className="px-6 py-4">
                        <Pill variant={prompt.category} />
                      </td>
                      <td className="px-6 py-4 text-sm text-slate-700 font-medium">
                        {prompt.text}
                      </td>
                      <td className="px-6 py-4">
                        <div className={cn(
                          "w-2 h-2 rounded-full",
                          prompt.active ? "bg-emerald-500" : "bg-slate-300"
                        )} />
                      </td>
                    </tr>
                    {expandedId === prompt.id && (
                      <tr>
                        <td colSpan={5} className="px-6 py-8 bg-slate-50/30">
                           <PromptDetail brand="Adore Me" promptId={prompt.id} />
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                ))
              )}
            </tbody>
          </table>
        </div>
      </PageShell>
    </>
  )
}

function PromptDetail({ brand, promptId }: { brand: string, promptId: number }) {
    const { data: scores, isLoading } = useStability(brand)
    
    if (isLoading) return <Skeleton className="h-20 w-full rounded-2xl" />
    
    const promptScores = scores?.filter(s => s.prompt_id === promptId) || []
    const providers: Provider[] = ['claude', 'chatgpt', 'perplexity', 'gemini']

    return (
        <div className="flex flex-col gap-6">
            <div className="flex items-center gap-4">
                <div className="p-2 bg-white rounded-xl shadow-sm border border-slate-100">
                    <Activity size={16} className="text-indigo-600" />
                </div>
                <h4 className="text-xs font-bold text-slate-900 uppercase tracking-widest">Stability Analysis (Latest Run)</h4>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                {providers.map(p => {
                    const score = promptScores.find(s => s.provider === p)
                    return (
                        <div key={p} className="bg-white p-4 rounded-2xl border border-slate-100 shadow-sm">
                            <div className="flex items-center justify-between mb-3">
                                <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{capitalize(p)}</span>
                                <div className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: (providerColour as any)[p] }} />
                            </div>
                            <div className="flex items-baseline gap-1">
                                <span className="text-xl font-black text-slate-900">{score ? score.stability_score.toFixed(0) : '-'}</span>
                                <span className="text-[10px] font-bold text-slate-400">%</span>
                            </div>
                            <div className="mt-2 h-1 w-full bg-slate-50 rounded-full overflow-hidden">
                                <div 
                                    className={cn(
                                        "h-full rounded-full transition-all",
                                        score && score.stability_score < 40 ? "bg-red-400" : score && score.stability_score < 70 ? "bg-amber-400" : "bg-emerald-400"
                                    )}
                                    style={{ width: score ? `${score.stability_score}%` : '0%' }}
                                />
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}

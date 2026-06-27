import React from 'react'
import { PromptResult, Provider } from '@/lib/types'
import { Pill } from '../ui/Pill'
import { ProviderDot } from '../ui/ProviderDot'

interface PromptResultsTableProps {
  results: PromptResult[]
}

export const PromptResultsTable: React.FC<PromptResultsTableProps> = ({ results }) => {
  // Group results by prompt_id
  const grouped = results.reduce((acc, curr) => {
    if (!acc[curr.prompt_id]) {
      acc[curr.prompt_id] = {
        text: curr.prompt_text,
        category: curr.category,
        providers: {} as Record<Provider, boolean>
      }
    }
    acc[curr.prompt_id].providers[curr.provider] = curr.brand_mentioned
    return acc
  }, {} as Record<number, { text: string, category: any, providers: Record<Provider, boolean> }>)

  const providers: Provider[] = ['claude', 'chatgpt', 'perplexity', 'gemini']

  return (
    <div className="bg-white rounded-3xl border border-slate-200 overflow-hidden shadow-sm">
      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-slate-50/50 border-b border-slate-100">
              <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Prompt</th>
              <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Category</th>
              <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-center">Mentions</th>
              <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-center">Providers</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-50">
            {Object.entries(grouped).map(([id, data]) => {
              const mentionCount = Object.values(data.providers).filter(Boolean).length
              return (
                <tr key={id} className="hover:bg-slate-50/30 transition-colors">
                  <td className="px-6 py-4 text-sm text-slate-700 font-medium max-w-md truncate">
                    {data.text}
                  </td>
                  <td className="px-6 py-4">
                    <Pill variant={data.category} />
                  </td>
                  <td className="px-6 py-4 text-center">
                    <span className="text-sm font-bold text-slate-900">{mentionCount}</span>
                    <span className="text-xs text-slate-400">/4</span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex justify-center gap-1.5">
                      {providers.map(p => (
                        <ProviderDot key={p} provider={p} hit={!!data.providers[p]} />
                      ))}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

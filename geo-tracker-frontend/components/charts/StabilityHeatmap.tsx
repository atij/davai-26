"use client"

import React, { useState } from 'react'
import { StabilityDatum, Provider } from '@/lib/types'
import { providerColour, capitalize } from '@/lib/utils'

interface StabilityHeatmapProps {
  data: StabilityDatum[]
}

const providers: Provider[] = ['claude', 'chatgpt', 'perplexity', 'gemini']

export const StabilityHeatmap: React.FC<StabilityHeatmapProps> = ({ data }) => {
  const [showAll, setShowAll] = useState(false)

  // Group data by prompt
  const promptsMap = new Map<string, Record<Provider, number>>()
  data.forEach(d => {
    if (!promptsMap.has(d.prompt)) {
      promptsMap.set(d.prompt, {} as Record<Provider, number>)
    }
    promptsMap.get(d.prompt)![d.provider] = d.score
  })

  // Sort by average stability (lowest first)
  const sortedPrompts = Array.from(promptsMap.entries())
    .map(([text, scores]) => {
      const values = Object.values(scores)
      const avg = values.reduce((a, b) => a + b, 0) / values.length
      return { text, scores, avg }
    })
    .sort((a, b) => a.avg - b.avg)

  const displayedPrompts = showAll ? sortedPrompts : sortedPrompts.slice(0, 10)

  const getCellColor = (score: number | undefined) => {
    if (score === undefined) return 'bg-slate-50'
    if (score < 40) return 'bg-red-500'
    if (score < 70) return 'bg-amber-400'
    return 'bg-emerald-500'
  }

  return (
    <div className="w-full">
      <div className="overflow-x-auto">
        <table className="w-full border-separate border-spacing-1">
          <thead>
            <tr>
              <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-wider p-2 w-1/2">Prompt</th>
              {providers.map(p => (
                <th key={p} className="p-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider text-center">
                  {capitalize(p)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {displayedPrompts.map(({ text, scores }) => (
              <tr key={text}>
                <td className="p-2 text-xs text-slate-600 truncate max-w-0" title={text}>
                  {text}
                </td>
                {providers.map(p => (
                  <td key={p} className="p-0">
                    <div 
                      className={`h-8 rounded-sm transition-opacity hover:opacity-80 ${getCellColor(scores[p])}`}
                      title={scores[p] !== undefined ? `${capitalize(p)}: ${scores[p].toFixed(1)}%` : 'No data'}
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      
      {sortedPrompts.length > 10 && (
        <button 
          onClick={() => setShowAll(!showAll)}
          className="mt-4 text-xs font-semibold text-indigo-600 hover:text-indigo-800"
        >
          {showAll ? 'Show less' : `Show all ${sortedPrompts.length} prompts`}
        </button>
      )}
    </div>
  )
}

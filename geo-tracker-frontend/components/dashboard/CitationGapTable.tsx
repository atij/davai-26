"use client"

import React, { useState } from 'react'
import { useCitationGap } from '@/hooks/useCitationGap'
import { Pill } from '../ui/Pill'
import { Skeleton } from '../ui/Skeleton'
import { EmptyState } from '../ui/EmptyState'
import { ChevronDown, ChevronUp, Link as LinkIcon } from 'lucide-react'

interface CitationGapTableProps {
  brand: string
}

export const CitationGapTable: React.FC<CitationGapTableProps> = ({ brand }) => {
  const { data: gaps, isLoading, error } = useCitationGap(brand)
  const [showMore, setShowAll] = useState(false)

  if (isLoading) return <Skeleton className="h-64 w-full rounded-3xl" />
  if (error) return null // Fail silently or show minimal error
  if (!gaps || gaps.length === 0) {
    return (
      <EmptyState 
        message="No citation gaps found for this brand." 
        className="h-64"
      />
    )
  }

  const displayedGaps = showMore ? gaps : gaps.slice(0, 10)

  return (
    <div className="bg-white rounded-3xl border border-slate-200 overflow-hidden shadow-sm">
      <div className="px-8 py-6 border-b border-slate-100 flex justify-between items-center">
        <div>
          <h3 className="text-sm font-bold text-slate-900 mb-1">Citation Gaps</h3>
          <p className="text-xs text-slate-400">Sources AI cites when {brand} is NOT mentioned</p>
        </div>
      </div>
      
      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-slate-50/50">
              <th className="px-8 py-3 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Domain</th>
              <th className="px-8 py-3 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-center">Citations</th>
              <th className="px-8 py-3 text-[10px] font-bold text-slate-400 uppercase tracking-widest text-right">Category</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-50">
            {displayedGaps.map((gap, i) => (
              <tr key={i} className="hover:bg-slate-50/30 transition-colors">
                <td className="px-8 py-4">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-slate-100 rounded-lg text-slate-400">
                      <LinkIcon size={12} />
                    </div>
                    <span className="text-sm font-semibold text-slate-700">{gap.domain}</span>
                  </div>
                </td>
                <td className="px-8 py-4 text-center">
                  <span className="px-2.5 py-1 bg-indigo-50 text-indigo-600 rounded-lg text-xs font-bold">
                    {gap.citation_count}
                  </span>
                </td>
                <td className="px-8 py-4 text-right">
                  <Pill variant={gap.category} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {gaps.length > 10 && (
        <button 
          onClick={() => setShowAll(!showMore)}
          className="w-full py-4 bg-slate-50/50 text-xs font-bold text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors flex items-center justify-center gap-2 border-t border-slate-100"
        >
          {showMore ? (
            <><ChevronUp size={14} /> Show Less</>
          ) : (
            <><ChevronDown size={14} /> Show {gaps.length - 10} More Sources</>
          )}
        </button>
      )}
    </div>
  )
}

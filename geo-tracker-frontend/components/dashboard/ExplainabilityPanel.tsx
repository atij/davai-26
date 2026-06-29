import React from 'react'
import { useExplain } from '@/hooks/useExplain'
import { Skeleton } from '@/components/ui/Skeleton'
import { AlertCircle } from 'lucide-react'

interface ExplainabilityPanelProps {
  brand: string
  runId: number
}

export const ExplainabilityPanel: React.FC<ExplainabilityPanelProps> = ({ brand, runId }) => {
  const { data: explanation, isLoading, error } = useExplain(runId, brand)

  if (isLoading) return <Skeleton className="h-48 w-full rounded-3xl mb-8" />
  if (error || !explanation) return null

  return (
    <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm mb-8">
      <div className="flex items-center gap-2 mb-6">
        <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em]">Why did visibility change?</h3>
      </div>
      
      <blockquote className="text-lg font-medium text-slate-900 mb-6 border-l-4 border-indigo-500 pl-6 py-2">
        {explanation.summary}
      </blockquote>

      {explanation.drivers && explanation.drivers.length > 0 && (
        <div className="space-y-4">
          <h4 className="text-xs font-bold text-slate-500 uppercase tracking-wider">Key Drivers</h4>
          <ul className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {explanation.drivers.map((driver, i) => (
              <li key={i} className="flex items-start gap-3 text-sm text-slate-600">
                <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-indigo-400 shrink-0" />
                {driver}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}

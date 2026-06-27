import React from 'react'
import { TrendingUp, TrendingDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface VisibilityScoreCardProps {
  score: number
  delta?: number
  brand: string
}

export const VisibilityScoreCard: React.FC<VisibilityScoreCardProps> = ({ score, delta, brand }) => {
  const getRingColor = (s: number) => {
    if (s < 40) return 'text-red-500'
    if (s < 70) return 'text-amber-400'
    return 'text-emerald-500'
  }

  const radius = 70
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (score / 100) * circumference

  return (
    <div className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm flex flex-col items-center justify-center text-center">
      <div className="relative mb-4">
        {/* Background Circle */}
        <svg className="w-40 h-40 transform -rotate-90">
          <circle
            cx="80"
            cy="80"
            r={radius}
            stroke="currentColor"
            strokeWidth="12"
            fill="transparent"
            className="text-slate-100"
          />
          {/* Progress Circle */}
          <circle
            cx="80"
            cy="80"
            r={radius}
            stroke="currentColor"
            strokeWidth="12"
            fill="transparent"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            className={cn("transition-all duration-1000 ease-out", getRingColor(score))}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-4xl font-bold text-slate-900">{score.toFixed(1)}</span>
          <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Score</span>
        </div>
      </div>
      
      <h2 className="text-xl font-bold text-slate-900 mb-1">{brand}</h2>
      <p className="text-xs text-slate-400 font-medium uppercase tracking-wider mb-4">Organic Visibility Score</p>
      
      {delta !== undefined && (
        <div className={cn(
          "flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold",
          delta >= 0 ? "bg-emerald-50 text-emerald-600" : "bg-red-50 text-red-600"
        )}>
          {delta >= 0 ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
          {delta > 0 ? '+' : ''}{delta.toFixed(1)} pts vs last run
        </div>
      )}
    </div>
  )
}

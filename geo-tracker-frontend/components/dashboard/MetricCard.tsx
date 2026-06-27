import React from 'react'
import { TrendingUp, TrendingDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface MetricCardProps {
  label: string
  value: string | number
  sub?: string
  up?: boolean
  subtitle?: string
  icon?: React.ReactNode
}

export const MetricCard: React.FC<MetricCardProps> = ({ 
  label, 
  value, 
  sub, 
  up, 
  subtitle,
  icon 
}) => {
  return (
    <div className="bg-white p-6 rounded-3xl border border-slate-200 shadow-sm transition-all hover:shadow-md">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">{label}</h3>
          {subtitle && <p className="text-[10px] text-slate-400 font-medium italic">{subtitle}</p>}
        </div>
        {icon && <div className="text-slate-300 bg-slate-50 p-2 rounded-xl">{icon}</div>}
      </div>
      
      <div className="flex items-baseline gap-2">
        <span className="text-3xl font-bold text-slate-900">{value}</span>
        {sub && (
          <span className={cn(
            "flex items-center text-xs font-bold",
            up === true ? "text-emerald-500" : up === false ? "text-red-500" : "text-slate-400"
          )}>
            {up === true && <TrendingUp size={12} className="mr-0.5" />}
            {up === false && <TrendingDown size={12} className="mr-0.5" />}
            {sub}
          </span>
        )}
      </div>
    </div>
  )
}

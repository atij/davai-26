import React from 'react'
import { Inbox } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  message: string
  title?: string
  icon?: React.ReactNode
  cta?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export const EmptyState: React.FC<EmptyStateProps> = ({ 
  message, 
  title, 
  icon, 
  cta,
  className 
}) => {
  return (
    <div className={cn(
      "flex flex-col items-center justify-center p-12 text-center bg-slate-50/50 rounded-3xl border-2 border-dashed border-slate-200",
      className
    )}>
      <div className="text-slate-300 mb-4">
        {icon || <Inbox size={48} strokeWidth={1.5} />}
      </div>
      {title && <h3 className="text-lg font-semibold text-slate-900 mb-1">{title}</h3>}
      <p className="text-sm text-slate-500 max-w-[260px] mx-auto mb-6">
        {message}
      </p>
      {cta && (
        <button
          onClick={cta.onClick}
          className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-xl hover:bg-indigo-700 transition-colors"
        >
          {cta.label}
        </button>
      )}
    </div>
  )
}

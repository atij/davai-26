import React from 'react'
import { cn } from '@/lib/utils' 
import { Sentiment, PromptCategory, RunStatus } from '@/lib/types'
import { sentimentClass, categoryClass, statusClass } from '@/lib/utils'

interface PillProps {
  variant: PromptCategory | Sentiment | RunStatus
  label?: string
  className?: string
}

export const Pill: React.FC<PillProps> = ({ variant, label, className }) => {
  const colorClass = 
    (categoryClass as any)[variant] || 
    (sentimentClass as any)[variant] || 
    (statusClass as any)[variant] || 
    'bg-gray-100 text-gray-800'

  return (
    <span className={cn(
      "px-2.5 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider",
      colorClass,
      className
    )}>
      {label || variant.replace('_', ' ')}
    </span>
  )
}

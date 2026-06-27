import { cn } from "@/lib/utils"
import { RunStatus } from "@/lib/types"

interface RunStatusBadgeProps {
  status: RunStatus
  className?: string
}

export const RunStatusBadge = ({ status, className }: RunStatusBadgeProps) => {
  const styles: Record<RunStatus, string> = {
    running: "bg-blue-50 text-blue-600 border-blue-100",
    done: "bg-emerald-50 text-emerald-600 border-emerald-100",
    failed: "bg-rose-50 text-rose-600 border-rose-100"
  }

  return (
    <span className={cn(
      "px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest border", 
      styles[status], 
      status === "running" && "animate-pulse",
      className
    )}>
      {status}
    </span>
  )
}

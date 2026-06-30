"use client"

import { useEffect, useState } from "react"
import { CheckCircle2, Circle, AlertCircle, Loader2 } from "lucide-react"
import { RunTrace } from "@/lib/types"
import { api } from "@/lib/api"
import { cn } from "@/lib/utils"

interface RunTraceTimelineProps {
  runId: number
}

export const RunTraceTimeline = ({ runId }: RunTraceTimelineProps) => {
  const [traces, setTraces] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchTrace = async () => {
      try {
        const res = await api.runTrace(runId)
        const data = await res.json()
        setTraces(data)
      } catch (err) {
        console.error("Failed to fetch run trace:", err)
      } finally {
        setIsLoading(false)
      }
    }

    fetchTrace()
    const interval = setInterval(fetchTrace, 5000)
    return () => clearInterval(interval)
  }, [runId])

  if (isLoading && traces.length === 0) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="animate-spin text-slate-400" />
      </div>
    )
  }

  if (!isLoading && traces.length === 0) {
    return (
      <div className="bg-white rounded-2xl border border-slate-100 p-6 shadow-sm text-center">
        <p className="text-xs font-bold text-slate-400 uppercase tracking-widest">No trace data available for this run</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-2xl border border-slate-100 p-6 shadow-sm">
      <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider mb-6">Agent Execution Graph</h3>
      <div className="space-y-4">
        {traces.map((trace, i) => (
          <div key={trace.id || `trace-${i}`} className="relative pl-8 pb-4 last:pb-0">
            {i !== traces.length - 1 && (
              <div className="absolute left-[11px] top-6 bottom-0 w-[2px] bg-slate-100" />
            )}
            <div className="absolute left-0 top-0.5">
              {(trace.status || trace.Status) === 'success' ? (
                <CheckCircle2 size={24} className="text-emerald-500 bg-white" />
              ) : (trace.status || trace.Status) === 'error' ? (
                <AlertCircle size={24} className="text-rose-500 bg-white" />
              ) : (trace.status || trace.Status) === 'running' ? (
                <Loader2 size={24} className="text-indigo-500 animate-spin bg-white" />
              ) : (
                <Circle size={24} className="text-slate-300 bg-white" />
              )}
            </div>
            <div className="flex flex-col">
              <div className="flex items-center gap-3">
                <span className="text-xs font-black uppercase tracking-widest text-slate-400">{trace.phase || trace.Phase}</span>
                <span className="text-sm font-bold text-slate-900">{trace.agent_name || trace.AgentName}</span>
                {(trace.duration_ms ?? trace.DurationMS) !== null && (trace.duration_ms ?? trace.DurationMS) !== undefined && (
                  <span className="text-[10px] font-bold text-slate-400 bg-slate-50 px-2 py-0.5 rounded-full">
                    {trace.duration_ms ?? trace.DurationMS}ms
                  </span>
                )}
              </div>
              {(trace.error_text || trace.ErrorText) && (
                <p className="mt-2 text-xs text-rose-600 bg-rose-50 p-2 rounded-lg border border-rose-100">
                  {trace.error_text || trace.ErrorText}
                </p>
              )}
              {(trace.started_at || trace.StartedAt) && (
                <span className="text-[10px] text-slate-400 mt-1 font-medium">
                  {new Date(trace.started_at || trace.StartedAt).toLocaleTimeString()}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

import React, { useState, useEffect } from 'react'
import { Play, X, Loader2 } from 'lucide-react'
import { usePrompts } from '@/hooks/usePrompts'
import { api } from '@/lib/api'
import { Provider, LiveRunResult } from '@/lib/types'
import { providerColour } from '@/lib/utils'

interface LiveRunButtonProps {
  brand: string
}

export const LiveRunButton: React.FC<LiveRunButtonProps> = ({ brand }) => {
  const [isOpen, setIsOpen] = useState(false)
  const [selectedPromptId, setSelectedPromptId] = useState<number | null>(null)
  const [results, setResults] = useState<LiveRunResult[]>([])
  const [isRunning, setIsRunning] = useState(false)
  const { data: prompts } = usePrompts()

  const startRun = () => {
    if (!selectedPromptId) return
    
    setIsRunning(true)
    setResults([])
    
    const providers: Provider[] = ['claude', 'chatgpt', 'perplexity', 'gemini']
    const eventSource = api.liveRun(selectedPromptId, providers)

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data)
      if (data.done) {
        setIsRunning(false)
        eventSource.close()
        return
      }
      setResults(prev => {
        const index = prev.findIndex(r => r.provider === data.provider)
        if (index > -1) {
          const newResults = [...prev]
          newResults[index] = data
          return newResults
        }
        return [...prev, data]
      })
    }

    eventSource.onerror = () => {
      setIsRunning(false)
      eventSource.close()
    }
  }

  return (
    <>
      <button 
        onClick={() => setIsOpen(true)}
        className="fixed bottom-8 right-8 bg-indigo-600 text-white p-4 rounded-full shadow-2xl hover:bg-indigo-700 transition-transform hover:scale-110 active:scale-95 flex items-center gap-2 group"
      >
        <Play size={20} fill="currentColor" />
        <span className="font-bold pr-2 max-w-0 overflow-hidden group-hover:max-w-xs transition-all duration-500 whitespace-nowrap">Run Now</span>
      </button>

      {isOpen && (
        <div className="fixed inset-0 bg-slate-900/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-[2rem] w-full max-w-2xl shadow-2xl overflow-hidden">
            <div className="p-8 border-b border-slate-100 flex justify-between items-center">
              <div>
                <h2 className="text-2xl font-black text-slate-900">Live Run</h2>
                <p className="text-sm text-slate-500">Probe AI models in real-time</p>
              </div>
              <button onClick={() => setIsOpen(false)} className="p-2 hover:bg-slate-100 rounded-full">
                <X size={24} />
              </button>
            </div>

            <div className="p-8">
              <div className="mb-8">
                <label className="block text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-3">Select Prompt</label>
                <select 
                  className="w-full bg-slate-50 border border-slate-200 rounded-2xl p-4 text-sm font-medium focus:ring-2 focus:ring-indigo-500 outline-none"
                  value={selectedPromptId || ''}
                  onChange={(e) => setSelectedPromptId(Number(e.target.value))}
                  disabled={isRunning}
                >
                  <option value="">Choose a prompt...</option>
                  {prompts?.map(p => (
                    <option key={p.id} value={p.id}>{p.text}</option>
                  ))}
                </select>
              </div>

              <div className="space-y-4 mb-8">
                {['claude', 'chatgpt', 'perplexity', 'gemini'].map(p => {
                  const res = results.find(r => r.provider === p)
                  return (
                    <div key={p} className="flex items-center justify-between p-4 bg-slate-50 rounded-2xl border border-slate-200">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 rounded-full" style={{ backgroundColor: providerColour[p as Provider] }} />
                        <span className="text-sm font-bold capitalize">{p}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        {!res ? (
                          isRunning ? <Loader2 size={16} className="animate-spin text-slate-300" /> : <span className="text-xs text-slate-300 italic">Waiting...</span>
                        ) : (
                          <div className={`px-3 py-1 rounded-full text-[10px] font-black uppercase ${res.brand_mentioned ? 'bg-green-100 text-green-700' : 'bg-slate-200 text-slate-500'}`}>
                            {res.brand_mentioned ? 'Mentioned' : 'Miss'}
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>

              <button
                onClick={startRun}
                disabled={!selectedPromptId || isRunning}
                className="w-full bg-indigo-600 text-white font-black py-4 rounded-2xl hover:bg-indigo-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2"
              >
                {isRunning ? (
                  <>
                    <Loader2 className="animate-spin" />
                    Running...
                  </>
                ) : 'Start Real-time Probe'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

"use client"

import { useState, useRef, useEffect } from "react"
import { Send, Bot, User, Database, ChevronRight, Loader2, Sparkles } from "lucide-react"
import { ChatEvent } from "@/lib/types"
import { api } from "@/lib/api"
import { cn } from "@/lib/utils"

interface Message {
  role: 'user' | 'assistant'
  content: string
  events?: ChatEvent[]
}

export const StrategyChat = ({ brand }: { brand: string }) => {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState("")
  const [isTyping, setIsTyping] = useState(false)
  const [sessionId] = useState(() => `${brand.toLowerCase().replace(/\s+/g, '-')}-${Math.random().toString(36).substring(7)}`)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, isTyping])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isTyping) return

    const userMsg = input.trim()
    setInput("")
    setMessages(prev => [...prev, { role: 'user', content: userMsg }])
    setIsTyping(true)

    try {
      const response = await api.strategyChat(brand, userMsg, sessionId)
      
      if (!response.body) throw new Error("No response body")
      
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let assistantMsg = ""
      let assistantEvents: ChatEvent[] = []

      while (true) {
        const { value, done } = await reader.read()
        if (done) break
        
        const chunk = decoder.decode(value)
        const lines = chunk.split("\n")
        
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            try {
              const event: ChatEvent = JSON.parse(line.substring(6))
              
              if (event.type === 'chunk' && event.text) {
                assistantMsg += event.text
                setMessages(prev => {
                  const last = prev[prev.length - 1]
                  if (last?.role === 'assistant') {
                    return [...prev.slice(0, -1), { ...last, content: assistantMsg }]
                  }
                  return [...prev, { role: 'assistant', content: assistantMsg, events: assistantEvents }]
                })
              } else if (event.type === 'tool_call' || event.type === 'tool_result') {
                assistantEvents.push(event)
                setMessages(prev => {
                  const last = prev[prev.length - 1]
                  if (last?.role === 'assistant') {
                    return [...prev.slice(0, -1), { ...last, events: [...assistantEvents] }]
                  }
                  return [...prev, { role: 'assistant', content: assistantMsg, events: [...assistantEvents] }]
                })
              } else if (event.type === 'error') {
                console.error("Agent error:", event.error)
              }
            } catch (e) {
              // Ignore partial JSON parse errors
            }
          }
        }
      }
    } catch (err) {
      console.error("Chat error:", err)
    } finally {
      setIsTyping(false)
    }
  }

  return (
    <div className="flex flex-col h-[calc(100vh-12rem)] bg-white rounded-3xl border border-slate-100 shadow-xl shadow-slate-200/50 overflow-hidden">
      <div className="flex-1 overflow-y-auto p-6 space-y-8">
        {messages.length === 0 && (
          <div className="h-full flex flex-col items-center justify-center text-center max-w-sm mx-auto space-y-4">
            <div className="w-16 h-16 bg-indigo-50 rounded-2xl flex items-center justify-center text-indigo-600 mb-2">
              <Sparkles size={32} />
            </div>
            <h3 className="text-lg font-bold text-slate-900 uppercase tracking-tight">Strategy Assistant</h3>
            <p className="text-sm text-slate-500 font-medium leading-relaxed">
              Ask me about {brand}'s visibility trends, competitor gaps, or past recommendations. I have access to your live data tools.
            </p>
          </div>
        )}
        
        {messages.map((msg, i) => (
          <div key={i} className={cn("flex gap-4", msg.role === 'user' ? "flex-row-reverse" : "flex-row")}>
            <div className={cn(
              "w-10 h-10 rounded-xl flex items-center justify-center shrink-0 shadow-sm",
              msg.role === 'user' ? "bg-slate-900 text-white" : "bg-indigo-600 text-white"
            )}>
              {msg.role === 'user' ? <User size={20} /> : <Bot size={20} />}
            </div>
            <div className={cn("flex flex-col gap-3 max-w-[80%]", msg.role === 'user' ? "items-end" : "items-start")}>
              {msg.events && msg.events.length > 0 && (
                <div className="flex flex-col gap-2 w-full">
                  {msg.events.filter(e => e.type === 'tool_call').map((ev, j) => (
                    <div key={j} className="flex items-center gap-2 px-3 py-2 bg-slate-50 border border-slate-100 rounded-xl text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                      <Database size={14} className="text-indigo-500" />
                      Calling {ev.tool}...
                      <ChevronRight size={12} className="ml-auto text-slate-300" />
                    </div>
                  ))}
                </div>
              )}
              {msg.content && (
                <div className={cn(
                  "px-5 py-4 rounded-2xl text-sm leading-relaxed font-medium shadow-sm",
                  msg.role === 'user' 
                    ? "bg-slate-900 text-white rounded-tr-none" 
                    : "bg-slate-50 text-slate-800 border border-slate-100 rounded-tl-none"
                )}>
                  {msg.content}
                </div>
              )}
            </div>
          </div>
        ))}
        {isTyping && (
          <div className="flex gap-4">
            <div className="w-10 h-10 rounded-xl flex items-center justify-center bg-indigo-600 text-white shadow-sm">
              <Bot size={20} />
            </div>
            <div className="flex items-center gap-1.5 px-5 py-4 bg-slate-50 border border-slate-100 rounded-2xl rounded-tl-none">
              <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" />
              <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce [animation-delay:0.2s]" />
              <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce [animation-delay:0.4s]" />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSubmit} className="p-6 bg-slate-50 border-t border-slate-100">
        <div className="relative group">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask about strategy..."
            className="w-full pl-6 pr-14 py-4 bg-white border border-slate-200 rounded-2xl text-sm font-semibold focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all shadow-inner"
          />
          <button
            type="submit"
            disabled={!input.trim() || isTyping}
            className="absolute right-2 top-2 bottom-2 px-4 bg-slate-900 text-white rounded-xl hover:bg-indigo-600 disabled:opacity-50 disabled:hover:bg-slate-900 transition-all flex items-center justify-center shadow-lg shadow-slate-200"
          >
            <Send size={18} />
          </button>
        </div>
      </form>
    </div>
  )
}

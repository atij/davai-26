"use client"

import { usePathname, useRouter, useSearchParams } from "next/navigation"
import { cn } from "@/lib/utils"
import { useSummary } from "@/hooks/useSummary"

interface TopbarProps {
  title: string
  leftContent?: React.ReactNode
}

export const Topbar = ({ title, leftContent }: TopbarProps) => {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const currentBrand = searchParams.get("brand") || "adoreme"

  const { data: summary } = useSummary(currentBrand === "adoreme" ? "Adore Me" : "Victoria's Secret")

  const setBrand = (brand: string) => {
    const params = new URLSearchParams(searchParams.toString())
    params.set("brand", brand)
    router.push(`${pathname}?${params.toString()}`)
  }

  const isDashboard = pathname === "/dashboard"

  return (
    <header className="h-20 border-b border-slate-100 bg-white/80 backdrop-blur-md flex items-center justify-between px-8 sticky top-0 z-30">
      <div className="flex items-center">
        {leftContent}
        <h2 className="text-lg font-black text-slate-900 tracking-tight">{title}</h2>
      </div>

      <div className="flex items-center gap-8">
        {summary?.run_at && (
            <div className="hidden md:flex flex-col items-end">
                <span className="text-[8px] font-black text-slate-300 uppercase tracking-[0.2em]">Latest Data</span>
                <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">
                    {new Date(summary.run_at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                </span>
            </div>
        )}

        {isDashboard && (
          <div className="flex bg-slate-100 p-1.5 rounded-2xl gap-1 border border-slate-200/50 shadow-inner">
            <button
              onClick={() => setBrand("adoreme")}
              className={cn(
                "px-5 py-2 text-[10px] font-black uppercase tracking-[0.2em] rounded-xl transition-all duration-300",
                currentBrand === "adoreme" 
                  ? "bg-white text-indigo-600 shadow-md ring-1 ring-slate-200" 
                  : "text-slate-400 hover:text-slate-600"
              )}
            >
              Adore Me
            </button>
            <button
              onClick={() => setBrand("vs")}
              className={cn(
                "px-5 py-2 text-[10px] font-black uppercase tracking-[0.2em] rounded-xl transition-all duration-300",
                currentBrand === "vs" 
                  ? "bg-white text-emerald-600 shadow-md ring-1 ring-slate-200" 
                  : "text-slate-400 hover:text-slate-600"
              )}
            >
              Victoria&apos;s Secret
            </button>
          </div>
        )}
      </div>
    </header>
  )
}

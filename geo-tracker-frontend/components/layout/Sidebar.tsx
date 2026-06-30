"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { LayoutDashboard, BarChart2, MessageSquare, History, Lightbulb, ShipWheel, MessageCircle } from "lucide-react"

export const Sidebar = () => {
  const pathname = usePathname()

  const navItems = [
    { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
    { label: "Strategy Chat", href: "/strategy", icon: MessageCircle },
    { label: "Compare", href: "/compare", icon: BarChart2 },
    { label: "Prompts", href: "/prompts", icon: MessageSquare },
    { label: "Runs", href: "/runs", icon: History },
    { label: "Recommendations", href: "/recommendations", icon: Lightbulb },
  ]

  return (
    <aside className="w-64 border-r border-slate-100 bg-slate-50/50 flex flex-col h-screen sticky top-0">
      <div className="h-20 flex items-center px-6 border-b border-slate-100 bg-white">
        <div className="flex flex-col">
          <h1 className="text-sm font-black tracking-[0.2em] text-slate-900 flex items-center gap-2 uppercase">
            <ShipWheel size={20} className="text-indigo-600" strokeWidth={2.5} />
            Lighthouse
          </h1>
          <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-1 ml-7">
            AI Discovery Observatory
          </span>
        </div>
      </div>
      <nav className="flex-1 p-4 space-y-1.5">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = pathname.startsWith(item.href)
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-4 py-2.5 text-sm font-semibold rounded-xl transition-all duration-200",
                isActive 
                  ? "bg-indigo-600 text-white shadow-lg shadow-indigo-200" 
                  : "text-slate-500 hover:bg-white hover:text-slate-900 hover:shadow-sm"
              )}
            >
              <Icon size={18} className={cn(isActive ? "text-white" : "text-slate-400")} />
              {item.label}
            </Link>
          )
        })}
      </nav>
      <div className="p-6 border-t border-slate-100 bg-white">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-bold text-slate-900 uppercase tracking-widest">Adore Me Tech</span>
          <span className="text-[10px] text-slate-400 font-medium tracking-wide">2026</span>
        </div>
      </div>
    </aside>
  )
}

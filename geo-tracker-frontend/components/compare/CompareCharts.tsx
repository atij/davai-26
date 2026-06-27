"use client"

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts"
import { TrendChart } from "@/components/charts/TrendChart"
import { BrandSummary, TrendPoint } from "@/lib/types"
import { capitalize } from "@/lib/utils"

interface CompareChartsProps {
  trendData: any[]
  sovData: any[]
}

export const CompareCharts = ({ trendData, sovData }: CompareChartsProps) => {
  const brands = [
    { key: "adoreme", name: "Adore Me", color: "#7F77DD" },
    { key: "vs", name: "Victoria's Secret", color: "#1D9E75" },
  ]

  return (
    <div className="grid grid-cols-1 gap-12">
      <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-[0_1px_3px_rgba(0,0,0,0.05),0_10px_20px_-5px_rgba(0,0,0,0.02)] transition-all hover:shadow-md">
        <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8">Comparative Trend</h3>
        <TrendChart data={trendData} brands={brands} />
      </section>

      <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-[0_1px_3px_rgba(0,0,0,0.05),0_10px_20px_-5px_rgba(0,0,0,0.02)] transition-all hover:shadow-md">
        <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-8">Share of Voice Comparison</h3>
        <div className="h-[400px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={sovData}
              margin={{ top: 20, right: 30, left: 0, bottom: 0 }}
            >
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f0f0f0" />
              <XAxis 
                dataKey="provider" 
                axisLine={false} 
                tickLine={false} 
                tick={{ fontSize: 12, fill: "#666" }}
                tickFormatter={(val: string) => capitalize(val)}
              />
              <YAxis 
                axisLine={false} 
                tickLine={false} 
                tick={{ fontSize: 12, fill: "#666" }}
                tickFormatter={(val) => `${val}%`}
              />
              <Tooltip 
                cursor={{ fill: "rgba(0,0,0,0.05)" }}
                contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
              />
              <Legend />
              <Bar dataKey="adoreme" name="Adore Me" fill="#7F77DD" radius={[4, 4, 0, 0]} />
              <Bar dataKey="vs" name="Victoria's Secret" fill="#1D9E75" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>
    </div>
  )
}

"use client"

import React from 'react'
import { 
  LineChart, 
  Line, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  Legend
} from 'recharts'
import { TrendDatum } from '@/lib/types'
import { brandColour, formatDate } from '@/lib/utils'

interface TrendChartProps {
  data: TrendDatum[]
  brands: string[]
  height?: number
}

export const TrendChart: React.FC<TrendChartProps> = ({ data, brands, height = 300 }) => {
  if (!data || data.length < 2) {
    return (
      <div className="bg-white rounded-3xl border border-slate-200 p-8 flex flex-col items-center justify-center h-64 gap-3">
        <p className="text-sm font-bold text-slate-900">Mention Trend</p>
        <p className="text-xs text-slate-400 text-center max-w-48">
          Trend appears after 2+ runs. Run the pipeline again tomorrow to see movement.
        </p>
        {data?.length === 1 && brands[0] && (
          <p className="text-2xl font-black text-indigo-600">{(data[0][brands[0]] as number).toFixed(1)}%</p>
        )}
      </div>
    )
  }

  return (
    <div style={{ width: '100%', height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={data}
          margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
          <XAxis 
            dataKey="run_at" 
            tickFormatter={formatDate}
            axisLine={false}
            tickLine={false}
            tick={{ fill: '#94a3b8', fontSize: 12 }}
          />
          <YAxis 
            axisLine={false}
            tickLine={false}
            tick={{ fill: '#94a3b8', fontSize: 12 }}
            tickFormatter={(val) => `${val}%`}
          />
          <Tooltip 
            contentStyle={{ 
              borderRadius: '12px', 
              border: 'none', 
              boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1)' 
            }}
            labelFormatter={(label) => formatDate(label as string)}
            formatter={(value: number) => [`${value.toFixed(1)}%`, 'Mention Rate']}
          />
          <Legend 
            verticalAlign="top" 
            align="right" 
            iconType="circle"
            wrapperStyle={{ paddingBottom: '20px', fontSize: '12px' }}
          />
          {brands.map((brand) => (
            <Line
              key={brand}
              type="monotone"
              dataKey={brand}
              stroke={brandColour[brand] || '#7F77DD'}
              strokeWidth={3}
              dot={{ r: 4, strokeWidth: 2, fill: '#fff' }}
              activeDot={{ r: 6 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}

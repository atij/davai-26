import useSWR from 'swr'
import { api } from '@/lib/api'
import { TrendPoint } from '@/lib/types'

export function useTrend(brand: string, n = 10) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/trend?runs=${n}` : null,
    () => api.trend(brand, n).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { trend: data as TrendPoint[] | undefined, isLoading, error }
}

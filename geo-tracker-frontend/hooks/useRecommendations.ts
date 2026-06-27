import useSWR from 'swr'
import { api } from '@/lib/api'
import { Recommendation } from '@/lib/types'

export function useRecommendations(brand: string) {
  const { data, isLoading, error, mutate } = useSWR(
    brand ? `/api/recommendations?brand=${brand}` : null,
    () => api.recommendations(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { 
    data: (Array.isArray(data) ? data : []) as Recommendation[], 
    isLoading, 
    error, 
    mutate 
  }
}

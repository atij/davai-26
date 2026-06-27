import useSWR from 'swr'
import { api } from '@/lib/api'
import { StabilityScore } from '@/lib/types'

export function useStability(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/stability` : null,
    () => api.stability(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { 
    data: (Array.isArray(data) ? data : []) as StabilityScore[], 
    isLoading, 
    error 
  }
}

import useSWR from 'swr'
import { api } from '@/lib/api'
import { CompareResponse } from '@/lib/types'

export function useCompare(brandA: string, brandB: string) {
  const { data, isLoading, error } = useSWR(
    (brandA && brandB) ? `/api/compare/organic?brands=${brandA},${brandB}` : null,
    () => api.compareOrganic(brandA, brandB).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { comparison: data as CompareResponse | undefined, isLoading, error }
}

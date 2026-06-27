import useSWR from 'swr'
import { api } from '@/lib/api'
import { Explanation } from '@/lib/types'

export function useExplain(runId: number, brand: string) {
  const { data, isLoading, error } = useSWR(
    runId && brand ? `/api/explain/${runId}?brand=${brand}` : null,
    () => api.explain(runId, brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { data: data as Explanation | undefined, isLoading, error }
}

import useSWR from 'swr'
import { api } from '@/lib/api'
import { PromptResult } from '@/lib/types'

export function useRunDetail(id: number) {
  const { data, isLoading, error } = useSWR(
    id ? `/api/runs/${id}/results` : null,
    () => api.runDetail(id).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { 
    results: (Array.isArray(data) ? data : []) as PromptResult[], 
    isLoading, 
    error 
  }
}

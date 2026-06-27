import useSWR from 'swr'
import { api } from '@/lib/api'
import { Run } from '@/lib/types'

export function useRuns(page = 1) {
  const { data, isLoading, error } = useSWR(
    `/api/runs?page=${page}`,
    () => api.runs(page).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { 
    runs: (Array.isArray(data) ? data : []) as Run[], 
    isLoading, 
    error 
  }
}

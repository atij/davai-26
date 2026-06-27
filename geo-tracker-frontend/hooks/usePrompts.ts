import useSWR from 'swr'
import { api } from '@/lib/api'
import { Prompt } from '@/lib/types'

export function usePrompts() {
  const { data, isLoading, error } = useSWR(
    '/api/prompts',
    () => api.prompts().then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { 
    prompts: (Array.isArray(data) ? data : []) as Prompt[], 
    isLoading, 
    error 
  }
}

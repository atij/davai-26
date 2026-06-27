import useSWR from 'swr'
import { api } from '@/lib/api'
import { Competitor } from '@/lib/types'

export function useCompetitors(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/competitors?brand=${brand}` : null,
    () => api.competitors(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { competitors: data as Competitor[] | undefined, isLoading, error }
}

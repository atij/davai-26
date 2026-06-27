import useSWR from 'swr'
import { api } from '@/lib/api'
import { BrandSummary } from '@/lib/types'

export function useSummary(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/summary` : null,
    () => api.summary(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { data: data as BrandSummary | undefined, isLoading, error }
}

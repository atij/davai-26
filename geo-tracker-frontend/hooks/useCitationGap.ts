import useSWR from 'swr'
import { api } from '@/lib/api'
import { CitationGapEntry } from '@/lib/types'

export function useCitationGap(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/citation-gap` : null,
    () => api.citationGap(brand).then(r => r.json()),
    { refreshInterval: 30_000 }
  )
  return { data: data?.gaps as CitationGapEntry[] | undefined, isLoading, error }
}

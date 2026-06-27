import useSWR from 'swr'
import { api } from '@/lib/api'
import { ComparisonSummary } from '@/lib/types'

export function useComparisonSummary(brand: string) {
  const { data, isLoading, error } = useSWR(
    brand ? `/api/brands/${brand}/comparison-summary` : null,
    () => api.comparisonSummary(brand).then(async (r) => {
        const body = await r.json();
        if (!r.ok) {
            const err = new Error(body.error || 'Failed to fetch comparison summary');
            (err as any).code = body.code;
            throw err;
        }
        return body;
    }),
    { refreshInterval: 30_000 }
  )

  const noData = (error as any)?.code === 'NO_COMPARISON_DATA'
  
  return {
    data: noData ? null : data as ComparisonSummary | null,
    isLoading: noData ? false : isLoading,
    error: noData ? null : error,
  }
}

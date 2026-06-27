"use client"

import { useEffect } from "react"
import { PageShell } from "@/components/layout/PageShell"
import { EmptyState } from "@/components/ui/EmptyState"
import { AlertTriangle } from "lucide-react"

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error(error)
  }, [error])

  return (
    <PageShell>
      <div className="flex items-center justify-center min-h-[60vh]">
        <EmptyState
          icon={<AlertTriangle size={48} className="text-red-500" />}
          title="Something went wrong!"
          message="An unexpected error occurred while rendering this page."
          action={
            <button
              onClick={() => reset()}
              className="px-6 py-2 bg-gray-900 text-white rounded-lg font-semibold transition-colors hover:bg-gray-800"
            >
              Try again
            </button>
          }
        />
      </div>
    </PageShell>
  )
}

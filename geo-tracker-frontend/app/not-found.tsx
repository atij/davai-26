"use client"

import Link from "next/link"
import { PageShell } from "@/components/layout/PageShell"
import { EmptyState } from "@/components/ui/EmptyState"
import { Search } from "lucide-react"

export default function NotFound() {
  return (
    <PageShell>
      <div className="flex items-center justify-center min-h-[60vh]">
        <EmptyState
          icon={<Search size={48} />}
          title="404 - Page Not Found"
          message="The page you are looking for doesn't exist or has been moved."
          action={
            <Link 
              href="/dashboard"
              className="px-6 py-2 bg-brand-adoreme text-white rounded-lg font-semibold transition-colors hover:bg-brand-adoreme/90"
            >
              Back to Dashboard
            </Link>
          }
        />
      </div>
    </PageShell>
  )
}

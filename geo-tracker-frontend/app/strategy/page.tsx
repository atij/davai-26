"use client"

import { useSearchParams } from "next/navigation"
import { PageShell } from "@/components/layout/PageShell"
import { StrategyChat } from "@/components/dashboard/StrategyChat"

export default function StrategyPage() {
  const searchParams = useSearchParams()
  const brand = searchParams.get("brand") === "victorias-secret" ? "Victoria's Secret" : "Adore Me"

  return (
    <PageShell title="GEO Strategy Assistant">
      <div className="max-w-4xl mx-auto">
        <StrategyChat brand={brand} />
      </div>
    </PageShell>
  )
}

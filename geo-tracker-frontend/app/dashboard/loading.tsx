import { PageShell } from "@/components/layout/PageShell"
import { Topbar } from "@/components/layout/Topbar"
import { MetricsRow } from "@/components/dashboard/MetricsRow"
import { Skeleton } from "@/components/ui/Skeleton"

export default function DashboardLoading() {
  return (
    <>
      <Topbar title="Dashboard" showBrandSwitcher={false} />
      <PageShell>
        <MetricsRow>
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </MetricsRow>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          <Skeleton className="h-[400px]" />
          <Skeleton className="h-[400px]" />
          <Skeleton className="h-[400px]" />
          <Skeleton className="h-[400px]" />
        </div>
        <Skeleton className="h-64" />
      </PageShell>
    </>
  )
}

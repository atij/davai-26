import { PageShell } from "@/components/layout/PageShell"
import { Topbar } from "@/components/layout/Topbar"
import { Skeleton } from "@/components/ui/Skeleton"

export default function RunsLoading() {
  return (
    <>
      <Topbar title="Run History" showBrandSwitcher={false} />
      <PageShell>
        <div className="space-y-4">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      </PageShell>
    </>
  )
}

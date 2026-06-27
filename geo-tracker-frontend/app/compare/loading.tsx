import { PageShell } from "@/components/layout/PageShell"
import { Topbar } from "@/components/layout/Topbar"
import { Skeleton } from "@/components/ui/Skeleton"

export default function CompareLoading() {
  return (
    <>
      <Topbar title="Compare Brands" showBrandSwitcher={false} />
      <PageShell>
        <div className="space-y-8">
          <div className="space-y-4">
            <Skeleton className="h-4 w-32" />
            <div className="grid grid-cols-4 gap-6">
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
            </div>
          </div>
          <div className="space-y-4">
            <Skeleton className="h-4 w-32" />
            <div className="grid grid-cols-4 gap-6">
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
            </div>
          </div>
          <Skeleton className="h-[400px] w-full" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </PageShell>
    </>
  )
}

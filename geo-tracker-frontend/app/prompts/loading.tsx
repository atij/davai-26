import { PageShell } from "@/components/layout/PageShell"
import { Topbar } from "@/components/layout/Topbar"
import { Skeleton } from "@/components/ui/Skeleton"

export default function PromptsLoading() {
  return (
    <>
      <Topbar title="Prompt Library" showBrandSwitcher={false} />
      <PageShell>
        <div className="flex gap-2 mb-8 overflow-x-auto">
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
        </div>
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

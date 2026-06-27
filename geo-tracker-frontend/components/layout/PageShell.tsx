import { ReactNode } from "react"

interface PageShellProps {
  children: ReactNode
}

export const PageShell = ({ children }: PageShellProps) => {
  return (
    <main className="flex-1 overflow-y-auto p-8 max-w-7xl mx-auto w-full">
      {children}
    </main>
  )
}

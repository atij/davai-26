import type { Metadata } from "next"
import { Inter } from "next/font/google"
import "@/styles/globals.css"
import { Sidebar } from "@/components/layout/Sidebar"
import { PageShell } from "@/components/layout/PageShell"

const inter = Inter({ subsets: ["latin"] })

export const metadata: Metadata = {
  title: "GEO Tracker",
  description: "Visualizing AI GEO visibility data",
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className={`${inter.className} bg-gray-50 flex min-h-screen`}>
        <Sidebar />
        <div className="flex-1 flex flex-col min-h-screen relative">
          {children}
        </div>
      </body>
    </html>
  )
}

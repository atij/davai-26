import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import { Sentiment, PromptCategory, Provider, RunStatus } from "./types"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const providerColour: Record<Provider, string> = {
  claude: '#AFA9EC', chatgpt: '#5DCAA5', perplexity: '#EF9F27', gemini: '#F0997B',
}

export const brandColour: Record<string, string> = {
  'Adore Me':           '#7F77DD',
  "Victoria's Secret":  '#1D9E75',
}

export const sentimentClass: Record<Sentiment, string> = {
  positive:      'bg-[#E1F5EE] text-[#085041]',
  neutral:       'bg-[#F1EFE8] text-[#444441]',
  negative:      'bg-[#FCEBEB] text-[#791F1F]',
  not_mentioned: 'bg-gray-100 text-gray-400',
}

export const categoryClass: Record<PromptCategory, string> = {
  purchase:   'bg-[#EEEDFE] text-[#3C3489]',
  discovery:  'bg-[#E1F5EE] text-[#085041]',
  comparison: 'bg-[#FAEEDA] text-[#633806]',
  fit:        'bg-[#FAECE7] text-[#712B13]',
  gifting:    'bg-[#E6F1FB] text-[#0C447C]',
}

export const statusClass: Record<RunStatus, string> = {
  running: 'bg-blue-100 text-blue-700',
  done:    'bg-green-100 text-green-700',
  failed:  'bg-red-100 text-red-700',
}

export const formatPercent  = (n: number) => `${(n || 0).toFixed(1)}%`
export const formatScore    = (n: number) => (n || 0).toFixed(1)
export const formatDate     = (s: string) => s ? new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) : 'N/A'
export const formatCost     = (n: number) => `$${(n || 0).toFixed(4)}`

export const capitalize = (str: string) => {
  if (!str) return ""
  return str.charAt(0).toUpperCase() + str.slice(1)
}


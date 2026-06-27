import React from 'react'
import { Provider } from '@/lib/types'
import { providerColour, cn } from '@/lib/utils'

interface ProviderDotProps {
  provider: Provider
  hit: boolean
}

const abbreviations: Record<Provider, string> = {
  claude: 'Cl',
  chatgpt: 'GP',
  perplexity: 'Px',
  gemini: 'Gm'
}

export const ProviderDot: React.FC<ProviderDotProps> = ({ provider, hit }) => {
  return (
    <span 
      className={cn(
        "inline-flex items-center justify-center w-6 h-6 rounded-full text-[10px] font-bold text-white",
        hit ? "" : "bg-gray-200 text-gray-400"
      )}
      style={hit ? { backgroundColor: providerColour[provider] } : {}}
    >
      {abbreviations[provider]}
    </span>
  )
}

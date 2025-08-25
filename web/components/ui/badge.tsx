import React from 'react'

type BadgeProps = {
  children: React.ReactNode
  color?: 'gold' | 'orange' | 'muted'
  className?: string
}

export function Badge({ children, color = 'gold', className = '' }: BadgeProps) {
  const map: Record<string, string> = {
    gold: 'bg-[#fbbf24]/20 text-[#fbbf24]',
    orange: 'bg-[#f97316]/10 text-[#f97316]',
    muted: 'bg-gray-800 text-gray-300',
  }

  return (
    <span className={`inline-block text-xs px-2 py-0.5 rounded ${map[color] || map.muted} ${className}`}>
      {children}
    </span>
  )
}

export default Badge

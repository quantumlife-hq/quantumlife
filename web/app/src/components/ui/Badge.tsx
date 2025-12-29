import { type HTMLAttributes, type ReactNode } from 'react'
import { clsx } from 'clsx'

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: 'primary' | 'success' | 'warning' | 'error' | 'neutral' | 'info'
  size?: 'sm' | 'md'
  dot?: boolean
  icon?: ReactNode
}

const variantStyles = {
  primary: 'bg-primary-500/20 text-primary-300 border-primary-500/30',
  success: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
  warning: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
  error: 'bg-red-500/20 text-red-300 border-red-500/30',
  neutral: 'bg-gray-500/20 text-gray-300 border-gray-500/30',
  info: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
}

const sizeStyles = {
  sm: 'px-1.5 py-0.5 text-xs',
  md: 'px-2.5 py-0.5 text-sm',
}

const dotColors = {
  primary: 'bg-primary-400',
  success: 'bg-emerald-400',
  warning: 'bg-amber-400',
  error: 'bg-red-400',
  neutral: 'bg-gray-400',
  info: 'bg-blue-400',
}

export function Badge({
  variant = 'neutral',
  size = 'md',
  dot = false,
  icon,
  className,
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full font-medium border',
        variantStyles[variant],
        sizeStyles[size],
        className
      )}
      {...props}
    >
      {dot && (
        <span className={clsx('w-1.5 h-1.5 rounded-full', dotColors[variant])} />
      )}
      {icon}
      {children}
    </span>
  )
}

// Status Badge with predefined statuses
export type StatusType = 'online' | 'offline' | 'busy' | 'away' | 'connected' | 'disconnected' | 'pending' | 'error'

const statusConfig: Record<StatusType, { label: string; variant: BadgeProps['variant']; dot: boolean }> = {
  online: { label: 'Online', variant: 'success', dot: true },
  offline: { label: 'Offline', variant: 'neutral', dot: true },
  busy: { label: 'Busy', variant: 'error', dot: true },
  away: { label: 'Away', variant: 'warning', dot: true },
  connected: { label: 'Connected', variant: 'success', dot: true },
  disconnected: { label: 'Disconnected', variant: 'neutral', dot: true },
  pending: { label: 'Pending', variant: 'warning', dot: true },
  error: { label: 'Error', variant: 'error', dot: true },
}

export interface StatusBadgeProps extends Omit<BadgeProps, 'variant' | 'dot'> {
  status: StatusType
  showLabel?: boolean
}

export function StatusBadge({ status, showLabel = true, ...props }: StatusBadgeProps) {
  const config = statusConfig[status]
  return (
    <Badge variant={config.variant} dot={config.dot} {...props}>
      {showLabel && config.label}
    </Badge>
  )
}

// Count Badge (for notifications, etc.)
export interface CountBadgeProps {
  count: number
  max?: number
  size?: 'sm' | 'md'
  variant?: BadgeProps['variant']
}

export function CountBadge({ count, max = 99, size = 'sm', variant = 'primary' }: CountBadgeProps) {
  if (count === 0) return null

  const displayCount = count > max ? `${max}+` : count.toString()

  return (
    <span
      className={clsx(
        'inline-flex items-center justify-center rounded-full font-semibold min-w-[1.25rem]',
        size === 'sm' ? 'h-5 px-1.5 text-xs' : 'h-6 px-2 text-sm',
        variant === 'primary' && 'bg-primary-500 text-white',
        variant === 'error' && 'bg-red-500 text-white',
        variant === 'success' && 'bg-emerald-500 text-white'
      )}
    >
      {displayCount}
    </span>
  )
}

// Provider Badge (for service connections)
export interface ProviderBadgeProps {
  name: string
  icon?: string
  isConnected?: boolean
  count?: number
}

export function ProviderBadge({ name, icon, isConnected, count }: ProviderBadgeProps) {
  return (
    <div className="flex items-center gap-2">
      {icon && <span className="text-lg">{icon}</span>}
      <span className="text-sm font-medium text-white">{name}</span>
      {typeof count === 'number' && count > 0 && (
        <CountBadge count={count} variant="primary" />
      )}
      {isConnected !== undefined && (
        <StatusBadge status={isConnected ? 'connected' : 'disconnected'} showLabel={false} />
      )}
    </div>
  )
}

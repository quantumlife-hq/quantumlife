import { type HTMLAttributes, forwardRef } from 'react'
import { clsx } from 'clsx'

export interface AvatarProps extends HTMLAttributes<HTMLDivElement> {
  src?: string | null
  alt?: string
  name?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  status?: 'online' | 'offline' | 'busy' | 'away'
  ring?: boolean
  ringColor?: 'primary' | 'success' | 'warning' | 'error'
}

const sizeStyles = {
  xs: 'w-6 h-6 text-xs',
  sm: 'w-8 h-8 text-sm',
  md: 'w-10 h-10 text-base',
  lg: 'w-12 h-12 text-lg',
  xl: 'w-16 h-16 text-xl',
}

const statusSizeStyles = {
  xs: 'w-1.5 h-1.5',
  sm: 'w-2 h-2',
  md: 'w-2.5 h-2.5',
  lg: 'w-3 h-3',
  xl: 'w-4 h-4',
}

const statusColors = {
  online: 'bg-emerald-500',
  offline: 'bg-gray-500',
  busy: 'bg-red-500',
  away: 'bg-amber-500',
}

const ringColors = {
  primary: 'ring-primary-500',
  success: 'ring-emerald-500',
  warning: 'ring-amber-500',
  error: 'ring-red-500',
}

// Generate consistent color from string
function stringToColor(str: string): string {
  const colors = [
    'bg-violet-600',
    'bg-blue-600',
    'bg-emerald-600',
    'bg-amber-600',
    'bg-rose-600',
    'bg-cyan-600',
    'bg-fuchsia-600',
    'bg-orange-600',
  ]
  if (!str) return colors[0]
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length] || colors[0]
}

// Get initials from name
function getInitials(name: string): string {
  if (!name) return 'U'
  const parts = name.trim().split(/\s+/)
  if (parts.length === 0 || !parts[0]) return 'U'
  if (parts.length === 1) {
    return parts[0].substring(0, 2).toUpperCase()
  }
  const first = parts[0]?.[0] || ''
  const last = parts[parts.length - 1]?.[0] || ''
  return (first + last).toUpperCase() || 'U'
}

export const Avatar = forwardRef<HTMLDivElement, AvatarProps>(
  (
    {
      src,
      alt,
      name,
      size = 'md',
      status,
      ring = false,
      ringColor = 'primary',
      className,
      ...props
    },
    ref
  ) => {
    const displayName = name ?? alt ?? 'User'
    const initials = getInitials(displayName)
    const bgColor = stringToColor(displayName)

    return (
      <div
        ref={ref}
        className={clsx('relative inline-flex flex-shrink-0', className)}
        {...props}
      >
        {src ? (
          <img
            src={src}
            alt={alt || displayName}
            className={clsx(
              'rounded-full object-cover',
              sizeStyles[size],
              ring && `ring-2 ring-offset-2 ring-offset-dark-900 ${ringColors[ringColor]}`
            )}
            onError={(e) => {
              // Hide broken image and show fallback
              e.currentTarget.style.display = 'none'
            }}
          />
        ) : (
          <div
            className={clsx(
              'rounded-full flex items-center justify-center font-semibold text-white',
              sizeStyles[size],
              bgColor,
              ring && `ring-2 ring-offset-2 ring-offset-dark-900 ${ringColors[ringColor]}`
            )}
            title={displayName}
          >
            {initials}
          </div>
        )}

        {/* Status indicator */}
        {status && (
          <span
            className={clsx(
              'absolute bottom-0 right-0 rounded-full ring-2 ring-dark-900',
              statusSizeStyles[size],
              statusColors[status]
            )}
          />
        )}
      </div>
    )
  }
)

Avatar.displayName = 'Avatar'

// Avatar Group Component
export interface AvatarGroupProps {
  children: React.ReactNode
  max?: number
  size?: AvatarProps['size']
  spacing?: 'tight' | 'normal' | 'loose'
}

const spacingStyles = {
  tight: '-space-x-3',
  normal: '-space-x-2',
  loose: '-space-x-1',
}

export function AvatarGroup({
  children,
  max = 4,
  size = 'md',
  spacing = 'normal',
}: AvatarGroupProps) {
  const childArray = Array.isArray(children) ? children : [children]
  const visibleChildren = childArray.slice(0, max)
  const remainingCount = childArray.length - max

  return (
    <div className={clsx('flex items-center', spacingStyles[spacing])}>
      {visibleChildren}
      {remainingCount > 0 && (
        <div
          className={clsx(
            'rounded-full flex items-center justify-center font-semibold bg-surface-lighter text-gray-300 ring-2 ring-dark-900',
            sizeStyles[size]
          )}
        >
          +{remainingCount}
        </div>
      )}
    </div>
  )
}

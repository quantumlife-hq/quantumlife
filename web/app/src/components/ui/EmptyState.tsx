import { type ReactNode } from 'react'
import { clsx } from 'clsx'
import { motion } from 'framer-motion'

export interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: ReactNode
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeStyles = {
  sm: {
    container: 'py-8',
    icon: 'w-12 h-12',
    title: 'text-base',
    description: 'text-sm',
  },
  md: {
    container: 'py-12',
    icon: 'w-16 h-16',
    title: 'text-lg',
    description: 'text-sm',
  },
  lg: {
    container: 'py-16',
    icon: 'w-20 h-20',
    title: 'text-xl',
    description: 'text-base',
  },
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  size = 'md',
  className,
}: EmptyStateProps) {
  const styles = sizeStyles[size]

  return (
    <motion.div
      className={clsx(
        'flex flex-col items-center justify-center text-center px-4',
        styles.container,
        className
      )}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      {icon && (
        <div
          className={clsx(
            'flex items-center justify-center rounded-full bg-surface-lighter text-gray-500 mb-4',
            styles.icon
          )}
        >
          {icon}
        </div>
      )}
      <h3 className={clsx('font-semibold text-white', styles.title)}>{title}</h3>
      {description && (
        <p className={clsx('text-gray-400 mt-2 max-w-sm', styles.description)}>
          {description}
        </p>
      )}
      {action && <div className="mt-6">{action}</div>}
    </motion.div>
  )
}

// Pre-built empty states for common scenarios
export function NoResultsState({ onClear }: { onClear?: () => void }) {
  return (
    <EmptyState
      icon={
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      }
      title="No results found"
      description="Try adjusting your search or filters to find what you're looking for."
      action={
        onClear && (
          <button
            onClick={onClear}
            className="text-primary-400 hover:text-primary-300 text-sm font-medium"
          >
            Clear filters
          </button>
        )
      }
    />
  )
}

export function NoConnectionsState({ onConnect }: { onConnect?: () => void }) {
  return (
    <EmptyState
      icon={
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
      }
      title="No connections yet"
      description="Connect your services to unlock powerful AI-assisted workflows."
      action={
        onConnect && (
          <button
            onClick={onConnect}
            className="btn btn-primary"
          >
            Connect a service
          </button>
        )
      }
    />
  )
}

export function NoItemsState({ itemType = 'items' }: { itemType?: string }) {
  return (
    <EmptyState
      icon={
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
      }
      title={`No ${itemType}`}
      description={`${itemType.charAt(0).toUpperCase() + itemType.slice(1)} will appear here once they're created.`}
    />
  )
}

export function ErrorState({ onRetry, message }: { onRetry?: () => void; message?: string }) {
  return (
    <EmptyState
      icon={
        <svg className="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      }
      title="Something went wrong"
      description={message || "We couldn't load the data. Please try again."}
      action={
        onRetry && (
          <button
            onClick={onRetry}
            className="btn btn-primary"
          >
            Try again
          </button>
        )
      }
    />
  )
}

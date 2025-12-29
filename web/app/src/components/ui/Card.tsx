import { forwardRef, type HTMLAttributes, type ReactNode } from 'react'
import { clsx } from 'clsx'
import { motion, type HTMLMotionProps } from 'framer-motion'

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'elevated' | 'outline' | 'solid'
  padding?: 'none' | 'sm' | 'md' | 'lg'
  hover?: boolean
  animate?: boolean
  glow?: boolean
}

const variantStyles = {
  default: 'glass',
  elevated: 'glass shadow-glass',
  outline: 'bg-transparent border border-surface-border',
  solid: 'bg-dark-800 border border-surface-border',
}

const paddingStyles = {
  none: 'p-0',
  sm: 'p-4',
  md: 'p-6',
  lg: 'p-8',
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  (
    {
      variant = 'default',
      padding = 'md',
      hover = false,
      animate = false,
      glow = false,
      className,
      children,
      ...props
    },
    ref
  ) => {
    const baseStyles = clsx(
      'rounded-xl',
      variantStyles[variant],
      paddingStyles[padding],
      hover && 'transition-all duration-300 hover:border-primary-500/30 hover:shadow-glow-sm cursor-pointer',
      glow && 'shadow-glow-sm',
      className
    )

    if (animate) {
      return (
        <motion.div
          ref={ref}
          className={baseStyles}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
          whileHover={hover ? { scale: 1.02, y: -4 } : undefined}
          {...(props as HTMLMotionProps<'div'>)}
        >
          {children}
        </motion.div>
      )
    }

    return (
      <div ref={ref} className={baseStyles} {...props}>
        {children}
      </div>
    )
  }
)

Card.displayName = 'Card'

// Card subcomponents
export interface CardHeaderProps extends HTMLAttributes<HTMLDivElement> {
  title?: string
  subtitle?: string
  action?: ReactNode
}

export const CardHeader = forwardRef<HTMLDivElement, CardHeaderProps>(
  ({ title, subtitle, action, className, children, ...props }, ref) => (
    <div
      ref={ref}
      className={clsx('flex items-start justify-between gap-4 mb-4', className)}
      {...props}
    >
      {(title || subtitle) ? (
        <div>
          {title && <h3 className="text-lg font-semibold text-white">{title}</h3>}
          {subtitle && <p className="text-sm text-gray-400 mt-0.5">{subtitle}</p>}
        </div>
      ) : (
        children
      )}
      {action}
    </div>
  )
)

CardHeader.displayName = 'CardHeader'

export const CardContent = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={clsx('', className)} {...props} />
  )
)

CardContent.displayName = 'CardContent'

export const CardFooter = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={clsx('flex items-center gap-3 mt-4 pt-4 border-t border-surface-border', className)}
      {...props}
    />
  )
)

CardFooter.displayName = 'CardFooter'

// Stat Card component
export interface StatCardProps extends CardProps {
  label: string
  value: string | number
  icon?: ReactNode
  trend?: {
    value: number
    isPositive: boolean
  }
  description?: string
}

export const StatCard = forwardRef<HTMLDivElement, StatCardProps>(
  ({ label, value, icon, trend, description, className, ...props }, ref) => (
    <Card ref={ref} className={clsx('flex flex-col', className)} {...props}>
      <div className="flex items-start justify-between">
        <span className="text-sm text-gray-400">{label}</span>
        {icon && <span className="text-primary-400">{icon}</span>}
      </div>
      <div className="mt-2 flex items-end gap-2">
        <span className="text-3xl font-bold text-white">{value}</span>
        {trend && (
          <span
            className={clsx(
              'text-sm font-medium',
              trend.isPositive ? 'text-emerald-400' : 'text-red-400'
            )}
          >
            {trend.isPositive ? '+' : ''}{trend.value}%
          </span>
        )}
      </div>
      {description && (
        <p className="mt-1 text-xs text-gray-500">{description}</p>
      )}
    </Card>
  )
)

StatCard.displayName = 'StatCard'

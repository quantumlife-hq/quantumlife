import { useState, useRef, useEffect, type ReactNode, type ReactElement, cloneElement } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { clsx } from 'clsx'

export interface TooltipProps {
  content: ReactNode
  children: ReactElement
  position?: 'top' | 'bottom' | 'left' | 'right'
  delay?: number
  disabled?: boolean
  className?: string
}

const positionStyles = {
  top: {
    container: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    arrow: 'top-full left-1/2 -translate-x-1/2 border-t-surface-lighter border-x-transparent border-b-transparent',
    initial: { opacity: 0, y: 4, scale: 0.95 },
    animate: { opacity: 1, y: 0, scale: 1 },
    exit: { opacity: 0, y: 4, scale: 0.95 },
  },
  bottom: {
    container: 'top-full left-1/2 -translate-x-1/2 mt-2',
    arrow: 'bottom-full left-1/2 -translate-x-1/2 border-b-surface-lighter border-x-transparent border-t-transparent',
    initial: { opacity: 0, y: -4, scale: 0.95 },
    animate: { opacity: 1, y: 0, scale: 1 },
    exit: { opacity: 0, y: -4, scale: 0.95 },
  },
  left: {
    container: 'right-full top-1/2 -translate-y-1/2 mr-2',
    arrow: 'left-full top-1/2 -translate-y-1/2 border-l-surface-lighter border-y-transparent border-r-transparent',
    initial: { opacity: 0, x: 4, scale: 0.95 },
    animate: { opacity: 1, x: 0, scale: 1 },
    exit: { opacity: 0, x: 4, scale: 0.95 },
  },
  right: {
    container: 'left-full top-1/2 -translate-y-1/2 ml-2',
    arrow: 'right-full top-1/2 -translate-y-1/2 border-r-surface-lighter border-y-transparent border-l-transparent',
    initial: { opacity: 0, x: -4, scale: 0.95 },
    animate: { opacity: 1, x: 0, scale: 1 },
    exit: { opacity: 0, x: -4, scale: 0.95 },
  },
}

export function Tooltip({
  content,
  children,
  position = 'top',
  delay = 200,
  disabled = false,
  className,
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const showTooltip = () => {
    if (disabled) return
    timeoutRef.current = setTimeout(() => {
      setIsVisible(true)
    }, delay)
  }

  const hideTooltip = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
    setIsVisible(false)
  }

  const positionConfig = positionStyles[position]

  return (
    <div className="relative inline-flex">
      {cloneElement(children, {
        onMouseEnter: (e: React.MouseEvent) => {
          showTooltip()
          children.props.onMouseEnter?.(e)
        },
        onMouseLeave: (e: React.MouseEvent) => {
          hideTooltip()
          children.props.onMouseLeave?.(e)
        },
        onFocus: (e: React.FocusEvent) => {
          showTooltip()
          children.props.onFocus?.(e)
        },
        onBlur: (e: React.FocusEvent) => {
          hideTooltip()
          children.props.onBlur?.(e)
        },
      })}

      <AnimatePresence>
        {isVisible && (
          <motion.div
            className={clsx(
              'absolute z-50 px-3 py-1.5 text-sm text-white bg-surface-lighter rounded-lg shadow-lg whitespace-nowrap',
              positionConfig.container,
              className
            )}
            initial={positionConfig.initial}
            animate={positionConfig.animate}
            exit={positionConfig.exit}
            transition={{ duration: 0.15, ease: 'easeOut' }}
          >
            {content}
            {/* Arrow */}
            <span
              className={clsx(
                'absolute w-0 h-0 border-4',
                positionConfig.arrow
              )}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

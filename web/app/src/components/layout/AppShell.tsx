import { type ReactNode } from 'react'
import { clsx } from 'clsx'
import { motion } from 'framer-motion'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { useAppStore, useIsOnboarded } from '@/stores/app'

interface AppShellProps {
  children: ReactNode
}

export function AppShell({ children }: AppShellProps) {
  const { sidebarOpen } = useAppStore()
  const isOnboarded = useIsOnboarded()

  // If not onboarded, render without sidebar
  if (!isOnboarded) {
    return (
      <div className="min-h-screen gradient-bg">
        {children}
      </div>
    )
  }

  return (
    <div className="min-h-screen gradient-bg">
      {/* Sidebar */}
      <Sidebar />

      {/* Main Content Area */}
      <motion.div
        className="flex flex-col min-h-screen"
        animate={{
          marginLeft: sidebarOpen ? 256 : 72,
        }}
        transition={{ duration: 0.3, ease: 'easeOut' }}
      >
        {/* Header */}
        <Header />

        {/* Page Content */}
        <main className="flex-1 p-6 overflow-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            {children}
          </motion.div>
        </main>
      </motion.div>
    </div>
  )
}

// Content wrapper with max width constraint
export function PageContent({
  children,
  className,
  maxWidth = '7xl',
}: {
  children: ReactNode
  className?: string
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl' | '6xl' | '7xl' | 'full'
}) {
  const maxWidthStyles = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
    '3xl': 'max-w-3xl',
    '4xl': 'max-w-4xl',
    '5xl': 'max-w-5xl',
    '6xl': 'max-w-6xl',
    '7xl': 'max-w-7xl',
    full: 'max-w-full',
  }

  return (
    <div className={clsx('mx-auto w-full', maxWidthStyles[maxWidth], className)}>
      {children}
    </div>
  )
}

// Section wrapper with consistent spacing
export function PageSection({
  children,
  title,
  description,
  action,
  className,
}: {
  children: ReactNode
  title?: string
  description?: string
  action?: ReactNode
  className?: string
}) {
  return (
    <section className={clsx('mb-8', className)}>
      {(title || description || action) && (
        <div className="flex items-start justify-between mb-4">
          <div>
            {title && (
              <h2 className="text-lg font-semibold text-white">{title}</h2>
            )}
            {description && (
              <p className="text-sm text-gray-400 mt-1">{description}</p>
            )}
          </div>
          {action}
        </div>
      )}
      {children}
    </section>
  )
}

// Grid layouts
export function CardGrid({
  children,
  columns = 3,
  className,
}: {
  children: ReactNode
  columns?: 1 | 2 | 3 | 4
  className?: string
}) {
  const columnStyles = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4',
  }

  return (
    <div className={clsx('grid gap-4', columnStyles[columns], className)}>
      {children}
    </div>
  )
}

// Two-column layout (sidebar + main content)
export function TwoColumnLayout({
  sidebar,
  children,
  sidebarPosition = 'left',
  sidebarWidth = 'md',
}: {
  sidebar: ReactNode
  children: ReactNode
  sidebarPosition?: 'left' | 'right'
  sidebarWidth?: 'sm' | 'md' | 'lg'
}) {
  const widthStyles = {
    sm: 'w-64',
    md: 'w-80',
    lg: 'w-96',
  }

  return (
    <div className="flex gap-6">
      {sidebarPosition === 'left' && (
        <aside className={clsx('flex-shrink-0', widthStyles[sidebarWidth])}>
          {sidebar}
        </aside>
      )}
      <main className="flex-1 min-w-0">{children}</main>
      {sidebarPosition === 'right' && (
        <aside className={clsx('flex-shrink-0', widthStyles[sidebarWidth])}>
          {sidebar}
        </aside>
      )}
    </div>
  )
}

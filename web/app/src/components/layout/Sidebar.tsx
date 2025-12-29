import { NavLink, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import { motion, AnimatePresence } from 'framer-motion'
import {
  HomeIcon,
  LinkIcon,
  Cog6ToothIcon,
  SparklesIcon,
  BookOpenIcon,
  ShieldCheckIcon,
  BanknotesIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline'
import { useAppStore, useIsOnboarded, useUnreadCount } from '@/stores/app'
import { Avatar } from '@/components/ui'

interface NavItem {
  name: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
}

const mainNavItems: NavItem[] = [
  { name: 'Command Center', href: '/', icon: HomeIcon },
  { name: 'Connections', href: '/connections', icon: LinkIcon },
  { name: 'Learning', href: '/learning', icon: BookOpenIcon },
  { name: 'Proactive', href: '/proactive', icon: SparklesIcon },
]

const bottomNavItems: NavItem[] = [
  { name: 'Trust', href: '/trust', icon: ShieldCheckIcon },
  { name: 'Ledger', href: '/ledger', icon: BanknotesIcon },
  { name: 'Settings', href: '/settings', icon: Cog6ToothIcon },
]

export function Sidebar() {
  const location = useLocation()
  const { sidebarOpen, setSidebarOpen, identity } = useAppStore()
  const isOnboarded = useIsOnboarded()
  const unreadCount = useUnreadCount()

  // Add badge to Proactive nav item if there are unread notifications
  const navItemsWithBadges = mainNavItems.map((item) => ({
    ...item,
    badge: item.name === 'Proactive' ? unreadCount : undefined,
  }))

  if (!isOnboarded) {
    return null
  }

  return (
    <motion.aside
      className={clsx(
        'fixed left-0 top-0 h-screen z-40 flex flex-col',
        'bg-dark-950/95 backdrop-blur-xl border-r border-surface-border',
        'transition-[width] duration-300 ease-out'
      )}
      animate={{ width: sidebarOpen ? 256 : 72 }}
    >
      {/* Logo */}
      <div className="h-16 flex items-center justify-between px-4 border-b border-surface-border">
        <AnimatePresence mode="wait">
          {sidebarOpen ? (
            <motion.div
              key="full-logo"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="flex items-center gap-3"
            >
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-violet-600 flex items-center justify-center">
                <SparklesIcon className="w-5 h-5 text-white" />
              </div>
              <span className="font-bold text-white text-lg">QuantumLife</span>
            </motion.div>
          ) : (
            <motion.div
              key="icon-logo"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="w-10 h-10 rounded-lg bg-gradient-to-br from-primary-500 to-violet-600 flex items-center justify-center mx-auto"
            >
              <SparklesIcon className="w-6 h-6 text-white" />
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-4 px-3 overflow-y-auto">
        <ul className="space-y-1">
          {navItemsWithBadges.map((item) => (
            <NavItem
              key={item.href}
              item={item}
              isCollapsed={!sidebarOpen}
              isActive={location.pathname === item.href}
            />
          ))}
        </ul>
      </nav>

      {/* Bottom Navigation */}
      <div className="py-4 px-3 border-t border-surface-border">
        <ul className="space-y-1">
          {bottomNavItems.map((item) => (
            <NavItem
              key={item.href}
              item={item}
              isCollapsed={!sidebarOpen}
              isActive={location.pathname === item.href}
            />
          ))}
        </ul>
      </div>

      {/* User Profile */}
      <div className="p-3 border-t border-surface-border">
        <div
          className={clsx(
            'flex items-center gap-3 p-2 rounded-lg',
            'hover:bg-surface-light/30 transition-colors cursor-pointer'
          )}
        >
          <Avatar
            name={identity?.name}
            size="sm"
            status="online"
          />
          <AnimatePresence mode="wait">
            {sidebarOpen && (
              <motion.div
                initial={{ opacity: 0, width: 0 }}
                animate={{ opacity: 1, width: 'auto' }}
                exit={{ opacity: 0, width: 0 }}
                className="flex-1 min-w-0 overflow-hidden"
              >
                <p className="text-sm font-medium text-white truncate">
                  {identity?.name || 'User'}
                </p>
                <p className="text-xs text-gray-500 truncate">
                  {identity?.email || 'No email set'}
                </p>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      {/* Collapse Toggle */}
      <button
        onClick={() => setSidebarOpen(!sidebarOpen)}
        className={clsx(
          'absolute -right-3 top-20 w-6 h-6 rounded-full',
          'bg-surface-lighter border border-surface-border',
          'flex items-center justify-center',
          'text-gray-400 hover:text-white hover:bg-primary-500/20',
          'transition-all duration-200'
        )}
      >
        {sidebarOpen ? (
          <ChevronLeftIcon className="w-4 h-4" />
        ) : (
          <ChevronRightIcon className="w-4 h-4" />
        )}
      </button>
    </motion.aside>
  )
}

// Individual Nav Item Component
function NavItem({
  item,
  isCollapsed,
  isActive,
}: {
  item: NavItem
  isCollapsed: boolean
  isActive: boolean
}) {
  return (
    <li>
      <NavLink
        to={item.href}
        className={clsx(
          'flex items-center gap-3 px-3 py-2.5 rounded-lg',
          'transition-all duration-200 group relative',
          isActive
            ? 'bg-primary-500/20 text-primary-300'
            : 'text-gray-400 hover:text-white hover:bg-surface-light/30'
        )}
      >
        {/* Active indicator */}
        {isActive && (
          <motion.div
            layoutId="activeIndicator"
            className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary-500 rounded-r-full"
          />
        )}

        <item.icon
          className={clsx(
            'w-5 h-5 flex-shrink-0',
            isActive ? 'text-primary-400' : 'text-gray-500 group-hover:text-gray-300'
          )}
        />

        <AnimatePresence mode="wait">
          {!isCollapsed && (
            <motion.span
              initial={{ opacity: 0, width: 0 }}
              animate={{ opacity: 1, width: 'auto' }}
              exit={{ opacity: 0, width: 0 }}
              className="text-sm font-medium whitespace-nowrap overflow-hidden"
            >
              {item.name}
            </motion.span>
          )}
        </AnimatePresence>

        {/* Badge */}
        {item.badge !== undefined && item.badge > 0 && (
          <motion.span
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            className={clsx(
              'flex items-center justify-center min-w-[1.25rem] h-5 px-1.5',
              'bg-primary-500 text-white text-xs font-semibold rounded-full',
              isCollapsed ? 'absolute top-1 right-1' : 'ml-auto'
            )}
          >
            {item.badge > 99 ? '99+' : item.badge}
          </motion.span>
        )}

        {/* Tooltip for collapsed state */}
        {isCollapsed && (
          <div
            className={clsx(
              'absolute left-full ml-3 px-3 py-1.5',
              'bg-surface-lighter text-white text-sm rounded-lg',
              'opacity-0 invisible group-hover:opacity-100 group-hover:visible',
              'transition-all duration-200 whitespace-nowrap z-50',
              'shadow-lg'
            )}
          >
            {item.name}
          </div>
        )}
      </NavLink>
    </li>
  )
}

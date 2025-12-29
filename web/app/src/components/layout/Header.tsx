import { useState } from 'react'
import { useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import { motion, AnimatePresence } from 'framer-motion'
import {
  BellIcon,
  MagnifyingGlassIcon,
  QuestionMarkCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline'
import { useAppStore, useUnreadCount, useCurrentHat } from '@/stores/app'
import { Badge, Avatar, SearchInput } from '@/components/ui'
import type { Hat } from '@/types'

// Page title mapping
const pageTitles: Record<string, string> = {
  '/': 'Command Center',
  '/connections': 'Connections',
  '/learning': 'Learning Hub',
  '/proactive': 'Proactive Intelligence',
  '/trust': 'Trust Management',
  '/ledger': 'Action Ledger',
  '/settings': 'Settings',
  '/onboarding': 'Welcome',
}

export function Header() {
  const location = useLocation()
  const [showSearch, setShowSearch] = useState(false)
  const [showNotifications, setShowNotifications] = useState(false)
  const { notifications, markNotificationRead, identity } = useAppStore()
  const unreadCount = useUnreadCount()
  const currentHat = useCurrentHat()

  const pageTitle = pageTitles[location.pathname] || 'QuantumLife'

  return (
    <header className="h-16 flex items-center justify-between px-6 bg-dark-950/50 backdrop-blur-sm border-b border-surface-border">
      {/* Left: Page Title & Hat Context */}
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-semibold text-white">{pageTitle}</h1>

        {/* Current Hat Badge */}
        {currentHat && (
          <HatBadge hat={currentHat} />
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-3">
        {/* Search */}
        <AnimatePresence>
          {showSearch ? (
            <motion.div
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 280, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              className="overflow-hidden"
            >
              <SearchInput
                placeholder="Search everything..."
                autoFocus
                onBlur={() => setShowSearch(false)}
                className="w-full"
              />
            </motion.div>
          ) : (
            <button
              onClick={() => setShowSearch(true)}
              className="p-2 text-gray-400 hover:text-white hover:bg-surface-light/30 rounded-lg transition-colors"
            >
              <MagnifyingGlassIcon className="w-5 h-5" />
            </button>
          )}
        </AnimatePresence>

        {/* Help */}
        <button className="p-2 text-gray-400 hover:text-white hover:bg-surface-light/30 rounded-lg transition-colors">
          <QuestionMarkCircleIcon className="w-5 h-5" />
        </button>

        {/* Notifications */}
        <div className="relative">
          <button
            onClick={() => setShowNotifications(!showNotifications)}
            className={clsx(
              'p-2 rounded-lg transition-colors relative',
              showNotifications
                ? 'text-primary-400 bg-primary-500/20'
                : 'text-gray-400 hover:text-white hover:bg-surface-light/30'
            )}
          >
            <BellIcon className="w-5 h-5" />
            {unreadCount > 0 && (
              <span className="absolute -top-0.5 -right-0.5 w-4 h-4 bg-primary-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
                {unreadCount > 9 ? '9+' : unreadCount}
              </span>
            )}
          </button>

          {/* Notifications Dropdown */}
          <AnimatePresence>
            {showNotifications && (
              <NotificationsDropdown
                notifications={notifications}
                onMarkRead={markNotificationRead}
                onClose={() => setShowNotifications(false)}
              />
            )}
          </AnimatePresence>
        </div>

        {/* User Avatar */}
        <Avatar
          name={identity?.name}
          size="sm"
          className="cursor-pointer"
        />
      </div>
    </header>
  )
}

// Hat Badge Component
function HatBadge({ hat }: { hat: Hat }) {
  return (
    <Badge variant="primary" size="sm" icon={<span>{hat.icon}</span>}>
      {hat.name}
    </Badge>
  )
}

// Notifications Dropdown
interface NotificationsDropdownProps {
  notifications: Array<{
    id: string
    title: string
    message: string
    timestamp?: string
    created_at?: string
    read?: boolean
    is_read?: boolean
    type: string
  }>
  onMarkRead: (id: string) => void
  onClose: () => void
}

function NotificationsDropdown({
  notifications,
  onMarkRead,
  onClose,
}: NotificationsDropdownProps) {
  const unreadNotifications = notifications.filter((n) => !n.read)
  const recentNotifications = notifications.slice(0, 5)

  return (
    <motion.div
      initial={{ opacity: 0, y: 10, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 10, scale: 0.95 }}
      className="absolute right-0 top-full mt-2 w-80 glass-dark rounded-xl shadow-xl border border-surface-border overflow-hidden z-50"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-surface-border">
        <h3 className="font-semibold text-white">Notifications</h3>
        <div className="flex items-center gap-2">
          {unreadNotifications.length > 0 && (
            <span className="text-xs text-gray-400">
              {unreadNotifications.length} unread
            </span>
          )}
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white"
          >
            <XMarkIcon className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Notifications List */}
      <div className="max-h-80 overflow-y-auto">
        {recentNotifications.length === 0 ? (
          <div className="py-8 text-center text-gray-500">
            <BellIcon className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No notifications yet</p>
          </div>
        ) : (
          <ul>
            {recentNotifications.map((notification) => (
              <li
                key={notification.id}
                className={clsx(
                  'px-4 py-3 border-b border-surface-border last:border-b-0',
                  'hover:bg-surface-light/20 transition-colors cursor-pointer',
                  !notification.read && 'bg-primary-500/5'
                )}
                onClick={() => onMarkRead(notification.id)}
              >
                <div className="flex items-start gap-3">
                  <div
                    className={clsx(
                      'w-2 h-2 rounded-full mt-2 flex-shrink-0',
                      (notification.read || notification.is_read) ? 'bg-gray-600' : 'bg-primary-500'
                    )}
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-white truncate">
                      {notification.title}
                    </p>
                    <p className="text-xs text-gray-400 mt-0.5 line-clamp-2">
                      {notification.message}
                    </p>
                    <p className="text-xs text-gray-500 mt-1">
                      {formatTimestamp(notification.timestamp || notification.created_at || '')}
                    </p>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Footer */}
      {notifications.length > 5 && (
        <div className="px-4 py-2 border-t border-surface-border">
          <button className="text-sm text-primary-400 hover:text-primary-300">
            View all notifications
          </button>
        </div>
      )}
    </motion.div>
  )
}

// Helper function to format timestamp
function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`

  return date.toLocaleDateString()
}

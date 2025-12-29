/**
 * Global App Store using Zustand
 * Manages global application state
 */

import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import type { Identity, Settings, Notification, Hat } from '@/types'

interface AppState {
  // User
  identity: Identity | null
  settings: Settings | null
  isAuthenticated: boolean

  // UI State
  sidebarOpen: boolean
  currentHat: string | null
  notifications: Notification[]
  unreadCount: number

  // Loading states
  isLoading: boolean
  loadingMessage: string | null

  // Actions
  setIdentity: (identity: Identity | null) => void
  setSettings: (settings: Settings | null) => void
  setAuthenticated: (isAuthenticated: boolean) => void
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
  setCurrentHat: (hatId: string | null) => void
  setNotifications: (notifications: Notification[]) => void
  addNotification: (notification: Notification) => void
  markNotificationRead: (id: string) => void
  setUnreadCount: (count: number) => void
  setLoading: (isLoading: boolean, message?: string) => void
  reset: () => void
}

const initialState = {
  identity: null,
  settings: null,
  isAuthenticated: false,
  sidebarOpen: true,
  currentHat: null,
  notifications: [],
  unreadCount: 0,
  isLoading: false,
  loadingMessage: null,
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      ...initialState,

      setIdentity: (identity) => set({ identity, isAuthenticated: !!identity }),

      setSettings: (settings) => set({ settings }),

      setAuthenticated: (isAuthenticated) => set({ isAuthenticated }),

      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      setSidebarOpen: (open) => set({ sidebarOpen: open }),

      setCurrentHat: (hatId) => set({ currentHat: hatId }),

      setNotifications: (notifications) => set({
        notifications,
        unreadCount: notifications.filter(n => !n.is_read).length
      }),

      addNotification: (notification) => set((state) => ({
        notifications: [notification, ...state.notifications],
        unreadCount: state.unreadCount + (notification.is_read ? 0 : 1)
      })),

      markNotificationRead: (id) => set((state) => ({
        notifications: state.notifications.map(n =>
          n.id === id ? { ...n, is_read: true } : n
        ),
        unreadCount: Math.max(0, state.unreadCount - 1)
      })),

      setUnreadCount: (count) => set({ unreadCount: count }),

      setLoading: (isLoading, message) => set({
        isLoading,
        loadingMessage: message ?? null
      }),

      reset: () => set(initialState),
    }),
    {
      name: 'quantumlife-app-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
        currentHat: state.currentHat,
      }),
    }
  )
)

// Selector hooks for performance
export const useIdentity = () => useAppStore((state) => state.identity)
export const useSettings = () => useAppStore((state) => state.settings)
export const useIsAuthenticated = () => useAppStore((state) => state.isAuthenticated)
export const useSidebarOpen = () => useAppStore((state) => state.sidebarOpen)
export const useCurrentHatId = () => useAppStore((state) => state.currentHat)
export const useNotifications = () => useAppStore((state) => state.notifications)
export const useUnreadCount = () => useAppStore((state) => state.unreadCount)
export const useIsLoading = () => useAppStore((state) => state.isLoading)

// Check if user has completed onboarding
export const useIsOnboarded = () => useAppStore((state) =>
  state.settings?.onboarding_completed || state.settings?.onboarded || false
)

// Get current hat object (needs to be fetched separately)
// This just returns the hatId, components should fetch the Hat data from the hats query
export const useCurrentHat = (): Hat | null => {
  // This is a simplified version - in production, you'd use a query to get the hat data
  // For now, return null and let components fetch from the hats query
  return null
}

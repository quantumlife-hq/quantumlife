import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { useEffect, useState } from 'react'
import { AppShell } from '@/components/layout'
import {
  DashboardPage,
  ConnectionsPage,
  LearningPage,
  ProactivePage,
  TrustPage,
  LedgerPage,
  SettingsPage,
  OnboardingPage,
} from '@/pages'
import { ToastProvider } from '@/components/ui'
import { useAppStore, useIsOnboarded } from '@/stores/app'
import { api } from '@/services/api'

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 30, // 30 minutes (previously cacheTime)
      retry: 2,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 1,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </ToastProvider>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}

function AppContent() {
  const { setIdentity, setSettings } = useAppStore()
  const isOnboarded = useIsOnboarded()
  const [isInitializing, setIsInitializing] = useState(true)

  // Load initial data
  useEffect(() => {
    const loadInitialData = async () => {
      try {
        const [identity, settings] = await Promise.all([
          api.identity.get().catch(() => null),
          api.settings.get().catch(() => null),
        ])

        if (identity) {
          setIdentity(identity)
        }
        if (settings) {
          setSettings(settings)
        }
      } catch (error) {
        console.error('Failed to load initial data:', error)
      } finally {
        setIsInitializing(false)
      }
    }

    loadInitialData()
  }, [setIdentity, setSettings])

  // Show loading state during initialization to prevent redirect flicker
  if (isInitializing) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-dark-900">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-primary-500/20 flex items-center justify-center">
            <svg className="w-8 h-8 text-primary-400 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
          <p className="text-gray-400">Loading QuantumLife...</p>
        </div>
      </div>
    )
  }

  return (
    <AppShell>
      <Routes>
        {/* Onboarding */}
        <Route
          path="/onboarding"
          element={isOnboarded ? <Navigate to="/" replace /> : <OnboardingPage />}
        />

        {/* Protected routes */}
        <Route
          path="/"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <DashboardPage />}
        />
        <Route
          path="/connections"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <ConnectionsPage />}
        />
        <Route
          path="/learning"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <LearningPage />}
        />
        <Route
          path="/proactive"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <ProactivePage />}
        />
        <Route
          path="/trust"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <TrustPage />}
        />
        <Route
          path="/ledger"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <LedgerPage />}
        />
        <Route
          path="/settings"
          element={!isOnboarded ? <Navigate to="/onboarding" replace /> : <SettingsPage />}
        />

        {/* OAuth callback */}
        <Route
          path="/connections/callback"
          element={<OAuthCallback />}
        />

        {/* Catch all - redirect to dashboard */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppShell>
  )
}

// OAuth callback handler component
function OAuthCallback() {
  useEffect(() => {
    // The backend handles the OAuth callback and redirects
    // This component is just a loading state
    const params = new URLSearchParams(window.location.search)
    const success = params.get('success')
    const error = params.get('error')

    if (success === 'true') {
      // Redirect to connections page
      window.location.href = '/connections?connected=true'
    } else if (error) {
      // Redirect with error
      window.location.href = `/connections?error=${encodeURIComponent(error)}`
    }
  }, [])

  return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="text-center">
        <div className="w-12 h-12 mx-auto mb-4 rounded-xl bg-primary-500/20 flex items-center justify-center animate-pulse">
          <svg className="w-6 h-6 text-primary-400 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        </div>
        <p className="text-gray-400">Completing connection...</p>
      </div>
    </div>
  )
}

export default App

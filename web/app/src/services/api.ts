/**
 * API Service for QuantumLife
 * Handles all HTTP communication with the backend
 */

import type {
  Identity,
  Hat,
  Item,
  Space,
  Connection,
  Provider,
  Settings,
  Stats,
  LearningStats,
  BehavioralPattern,
  Recommendation,
  Nudge,
  Notification,
  TrustScore,
  LedgerEntry,
  SetupStatus,
  AgentStatus,
} from '@/types'

const API_BASE = '/api/v1'

// Generic fetch wrapper with error handling
async function fetchApi<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE}${endpoint}`

  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  return response.json()
}

// Identity
export async function getIdentity(): Promise<Identity> {
  return fetchApi<Identity>('/identity')
}

export async function createIdentity(name: string, passphrase: string): Promise<Identity> {
  return fetchApi<Identity>('/setup/identity', {
    method: 'POST',
    body: JSON.stringify({ name, passphrase }),
  })
}

// Hats
export async function getHats(): Promise<Hat[]> {
  return fetchApi<Hat[]>('/hats')
}

export async function getHat(id: string): Promise<Hat> {
  return fetchApi<Hat>(`/hats/${id}`)
}

export async function updateHat(id: string, data: Partial<Hat>): Promise<Hat> {
  return fetchApi<Hat>(`/hats/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

// Items
export async function getItems(params?: {
  hat?: string
  status?: string
  type?: string
  limit?: number
  offset?: number
}): Promise<{ items: Item[]; total: number }> {
  const query = new URLSearchParams()
  if (params?.hat) query.set('hat', params.hat)
  if (params?.status) query.set('status', params.status)
  if (params?.type) query.set('type', params.type)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))

  const queryString = query.toString()
  return fetchApi<{ items: Item[]; total: number }>(`/items${queryString ? `?${queryString}` : ''}`)
}

export async function getItem(id: string): Promise<Item> {
  return fetchApi<Item>(`/items/${id}`)
}

export async function createItem(data: Partial<Item>): Promise<Item> {
  return fetchApi<Item>('/items', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function updateItem(id: string, data: Partial<Item>): Promise<Item> {
  return fetchApi<Item>(`/items/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

// Spaces
export async function getSpaces(): Promise<Space[]> {
  return fetchApi<Space[]>('/spaces')
}

export async function syncSpace(spaceId: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/spaces/${spaceId}/sync`, {
    method: 'POST',
  })
}

// Connections (Nango-based)
export async function getConnections(): Promise<{ connections: Connection[]; count: number }> {
  return fetchApi<{ connections: Connection[]; count: number }>('/connections/')
}

export async function getProviders(): Promise<{ providers: Provider[]; categories: string[] }> {
  return fetchApi<{ providers: Provider[]; categories: string[] }>('/connections/providers')
}

export async function getProvidersByCategory(category: string): Promise<{ category: string; providers: Provider[] }> {
  return fetchApi<{ category: string; providers: Provider[] }>(`/connections/providers/${category}`)
}

export async function initiateConnect(provider: string, name?: string): Promise<{
  space_id: string
  provider: string
  auth_url: string
  session_token?: string
  expires_at?: string
  message: string
}> {
  return fetchApi<{
    space_id: string
    provider: string
    auth_url: string
    session_token?: string
    expires_at?: string
    message: string
  }>('/connections/connect', {
    method: 'POST',
    body: JSON.stringify({ provider, name }),
  })
}

export async function disconnectService(spaceId: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/connections/${spaceId}`, {
    method: 'DELETE',
  })
}

export async function getConnectionStatus(spaceId: string): Promise<Connection> {
  return fetchApi<Connection>(`/connections/${spaceId}/status`)
}

// Agent
export async function getAgentStatus(): Promise<AgentStatus> {
  return fetchApi<AgentStatus>('/agent/status')
}

export async function chatWithAgent(message: string): Promise<{ response: string; actions?: unknown[] }> {
  return fetchApi<{ response: string; actions?: unknown[] }>('/agent/chat', {
    method: 'POST',
    body: JSON.stringify({ message }),
  })
}

// Stats
export async function getStats(): Promise<Stats> {
  return fetchApi<Stats>('/stats')
}

// Settings
export async function getSettings(): Promise<Settings> {
  return fetchApi<Settings>('/settings')
}

export async function updateSettings(data: Partial<Settings>): Promise<Settings> {
  return fetchApi<Settings>('/settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function updateOnboardingStep(step: number): Promise<{ message: string }> {
  return fetchApi<{ message: string }>('/settings/onboarding', {
    method: 'POST',
    body: JSON.stringify({ step }),
  })
}

export async function exportData(): Promise<Blob> {
  const response = await fetch(`${API_BASE}/settings/export`)
  if (!response.ok) throw new Error('Export failed')
  return response.blob()
}

export async function deleteAccount(): Promise<{ message: string }> {
  return fetchApi<{ message: string }>('/settings/account', {
    method: 'DELETE',
  })
}

// Setup
export async function getSetupStatus(): Promise<SetupStatus> {
  return fetchApi<SetupStatus>('/setup/status')
}

export async function updateSetupProgress(step: string, connected: boolean): Promise<{ message: string }> {
  return fetchApi<{ message: string }>('/setup/progress', {
    method: 'POST',
    body: JSON.stringify({ step, connected }),
  })
}

export async function completeSetup(): Promise<{ message: string; completed_at: string }> {
  return fetchApi<{ message: string; completed_at: string }>('/setup/complete', {
    method: 'POST',
  })
}

export async function getOAuthUrl(provider: string): Promise<{ oauth_url: string; state: string; provider: string }> {
  return fetchApi<{ oauth_url: string; state: string; provider: string }>(`/oauth/${provider}/url`)
}

// Learning
export async function getLearningUnderstanding(): Promise<LearningStats> {
  return fetchApi<LearningStats>('/learning/understanding')
}

export async function getLearningPatterns(): Promise<BehavioralPattern[]> {
  const response = await fetchApi<{ patterns: BehavioralPattern[] }>('/learning/patterns')
  return response.patterns || []
}

export async function getLearningStats(): Promise<{ total_signals: number; total_patterns: number }> {
  return fetchApi<{ total_signals: number; total_patterns: number }>('/learning/stats')
}

// Proactive
export async function getRecommendations(): Promise<Recommendation[]> {
  return fetchApi<Recommendation[]>('/proactive/recommendations')
}

export async function getNudges(): Promise<Nudge[]> {
  return fetchApi<Nudge[]>('/proactive/nudges')
}

// Notifications
export async function getNotifications(): Promise<{ notifications: Notification[] }> {
  return fetchApi<{ notifications: Notification[] }>('/notifications')
}

export async function getUnreadCount(): Promise<{ count: number }> {
  return fetchApi<{ count: number }>('/notifications/unread-count')
}

export async function markNotificationRead(id: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/notifications/${id}/read`, {
    method: 'POST',
  })
}

export async function dismissNotification(id: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/notifications/${id}/dismiss`, {
    method: 'POST',
  })
}

export async function markAllNotificationsRead(): Promise<{ message: string }> {
  return fetchApi<{ message: string }>('/notifications/read-all', {
    method: 'POST',
  })
}

// Trust
export async function getTrustScores(): Promise<{ scores: TrustScore[] }> {
  return fetchApi<{ scores: TrustScore[] }>('/trust/scores')
}

export async function getTrustScore(domain: string): Promise<TrustScore> {
  return fetchApi<TrustScore>(`/trust/scores/${domain}`)
}

// Ledger
export async function getLedgerEntries(params?: {
  type?: string
  actor?: string
  limit?: number
  offset?: number
}): Promise<{ entries: LedgerEntry[]; total: number }> {
  const query = new URLSearchParams()
  if (params?.type) query.set('type', params.type)
  if (params?.actor) query.set('actor', params.actor)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))

  const queryString = query.toString()
  return fetchApi<{ entries: LedgerEntry[]; total: number }>(`/ledger${queryString ? `?${queryString}` : ''}`)
}

export async function verifyLedgerChain(): Promise<{ valid: boolean; message: string }> {
  return fetchApi<{ valid: boolean; message: string }>('/ledger/verify')
}

// Waitlist
export async function joinWaitlist(email: string, source?: string): Promise<{ message: string; position: number }> {
  return fetchApi<{ message: string; position: number }>('/waitlist', {
    method: 'POST',
    body: JSON.stringify({ email, source }),
  })
}

export async function getWaitlistCount(): Promise<{ count: number }> {
  return fetchApi<{ count: number }>('/waitlist/count')
}

// Memories
export async function getMemories(): Promise<{ memories: unknown[] }> {
  return fetchApi<{ memories: unknown[] }>('/memories')
}

export async function searchMemories(query: string): Promise<{ results: unknown[] }> {
  return fetchApi<{ results: unknown[] }>('/memories/search', {
    method: 'POST',
    body: JSON.stringify({ query }),
  })
}

export async function createMemory(content: string, type?: string): Promise<{ id: string }> {
  return fetchApi<{ id: string }>('/memories', {
    method: 'POST',
    body: JSON.stringify({ content, type }),
  })
}

// API namespace for convenient access
export const api = {
  identity: {
    get: getIdentity,
    create: createIdentity,
    update: async (data: Partial<Identity>) => fetchApi<Identity>('/identity', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  },
  hats: {
    list: getHats,
    get: getHat,
    update: updateHat,
    create: async (data: { name: string; icon: string; description: string }) =>
      fetchApi<Hat>('/hats', { method: 'POST', body: JSON.stringify(data) }),
    delete: async (id: string) =>
      fetchApi<{ message: string }>(`/hats/${id}`, { method: 'DELETE' }),
  },
  items: {
    list: async (params?: { hat?: string; status?: string; type?: string; limit?: number; offset?: number }) => {
      const result = await getItems(params)
      return result.items
    },
    get: getItem,
    create: createItem,
    update: updateItem,
  },
  spaces: {
    list: getSpaces,
    sync: syncSpace,
  },
  connections: {
    list: async () => {
      const result = await getConnections()
      return result.connections
    },
    providers: async () => {
      const result = await getProviders()
      return result.providers
    },
    connect: initiateConnect,
    disconnect: disconnectService,
    status: getConnectionStatus,
  },
  agent: {
    status: getAgentStatus,
    chat: chatWithAgent,
  },
  stats: {
    get: getStats,
  },
  settings: {
    get: getSettings,
    update: updateSettings,
  },
  setup: {
    status: getSetupStatus,
    progress: updateSetupProgress,
    complete: completeSetup,
    oauthUrl: getOAuthUrl,
  },
  learning: {
    getPatterns: getLearningPatterns,
    getPreferences: async () => {
      try {
        return await fetchApi<Array<{ id: string; category: string; key: string; value: string; confidence: number }>>('/learning/preferences')
      } catch {
        return [] // Endpoint may not exist yet
      }
    },
    deletePattern: async (id: string) => fetchApi<{ message: string }>(`/learning/patterns/${id}`, { method: 'DELETE' }),
    getStats: getLearningStats,
  },
  proactive: {
    getSuggestions: async () => {
      const recs = await getRecommendations()
      return recs.map(r => ({
        ...r,
        priority: r.confidence > 0.8 ? 'high' as const : r.confidence > 0.5 ? 'medium' as const : 'low' as const,
        source: r.hat_id || 'general',
        actions: [],
        createdAt: r.created_at,
      }))
    },
    actOnSuggestion: async (id: string, action: string) =>
      fetchApi<{ message: string }>(`/proactive/recommendations/${id}/${action}`, { method: 'POST' }),
  },
  notifications: {
    list: async () => {
      const result = await getNotifications()
      return result.notifications
    },
    unreadCount: getUnreadCount,
    markRead: markNotificationRead,
    dismiss: dismissNotification,
    markAllRead: markAllNotificationsRead,
  },
  trust: {
    getCapital: async () => {
      const result = await getTrustScores()
      // Calculate totals from scores
      const scores = result.scores || []
      const totalScore = scores.reduce((sum, s) => sum + s.score, 0)
      const maxScore = scores.length * 1000 || 10000
      return {
        totalScore,
        maxScore,
        currentLevel: Math.min(5, Math.floor(totalScore / 2000) + 1),
        level: Math.min(100, Math.round((totalScore / maxScore) * 100)),
        approvedActions: scores.reduce((sum, s) => sum + s.action_count, 0),
        rejectedActions: 0,
        autoApproved: 0,
      }
    },
    getHistory: async () => {
      const ledger = await getLedgerEntries({ type: 'trust', limit: 20 })
      return ledger.entries.map(e => ({
        id: e.id,
        type: e.result === 'success' ? 'approval' as const : 'rejection' as const,
        description: e.action,
        points: e.result === 'success' ? 10 : -5,
        timestamp: e.timestamp,
        source: e.domain,
      }))
    },
  },
  ledger: {
    getEntries: async (params?: { status?: string; type?: string }) => {
      const result = await getLedgerEntries(params)
      return result.entries.map(e => ({
        ...e,
        status: e.result === 'success' ? 'completed' as const : 'rejected' as const,
        trustImpact: e.result === 'success' ? 5 : -2,
        type: e.actor as 'manual' | 'auto' | 'suggestion',
        details: e.details || {},
      }))
    },
    verify: verifyLedgerChain,
  },
  waitlist: {
    join: joinWaitlist,
    count: getWaitlistCount,
  },
  memories: {
    list: getMemories,
    search: searchMemories,
    create: createMemory,
  },
}

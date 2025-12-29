// Core types for QuantumLife

// Identity
export interface Identity {
  id: string
  name: string
  email?: string
  bio?: string
  created_at: string
}

// Hat (life context/role)
export interface Hat {
  id: string
  name: string
  description: string
  icon: string
  color: string
  priority: number
  item_count?: number
  is_active: boolean
  created_at: string
  updated_at: string
}

// Item (email, event, task, etc.)
export interface Item {
  id: string
  type: ItemType
  status: ItemStatus
  hat_id: string
  subject: string
  title?: string // alias for subject
  sender?: string
  source?: string // alias for sender
  body?: string
  priority: number | 'high' | 'medium' | 'low'
  confidence: number
  space_id?: string
  timestamp?: string // alias for created_at
  created_at: string
  updated_at: string
}

export type ItemType = 'email' | 'event' | 'task' | 'note' | 'reminder' | 'message'
export type ItemStatus = 'pending' | 'routed' | 'handled' | 'archived' | 'snoozed'

// Space (connected service)
export interface Space {
  id: string
  type: SpaceType
  provider: string
  name: string
  is_connected: boolean
  last_sync_at?: string
  sync_status: string
  auth_source: AuthSource
  created_at: string
  updated_at: string
}

export type SpaceType = 'email' | 'calendar' | 'files' | 'chat' | 'finance' | 'custom'
export type AuthSource = 'custom' | 'nango'

// Provider (available service to connect)
export interface Provider {
  key: string
  name: string
  category: ProviderCategory
  icon: string
  auth_mode: 'oauth2' | 'oauth1' | 'api_key' | 'basic'
  is_connected: boolean
  connections: number
}

export type ProviderCategory =
  | 'email'
  | 'calendar'
  | 'communication'
  | 'productivity'
  | 'development'
  | 'finance'
  | 'health'
  | 'social'
  | 'storage'

// Connection (a user's connection to a service)
export interface Connection {
  id: string
  provider: string
  name: string
  type: SpaceType
  auth_source: AuthSource
  is_connected: boolean
  last_sync_at?: string
  sync_status: string
  status?: 'connected' | 'disconnected' | 'error'
  icon?: string
  email?: string
  itemCount?: number
  created_at: string
}

// Agent
export interface AgentStatus {
  is_running: boolean
  current_hat?: string
  last_action?: string
  items_processed: number
  uptime_seconds: number
}

// Learning
export interface LearningStats {
  confidence: number
  signals_count: number
  patterns_count: number
  sender_profiles: Record<string, SenderProfile>
}

export interface SenderProfile {
  email: string
  name?: string
  priority: number
  response_time_avg: number
  interaction_count: number
}

export interface BehavioralPattern {
  id: string
  type: string
  pattern: string
  confidence: number
  hat_id?: string
  created_at: string
}

// Settings
export interface Settings {
  display_name?: string
  email?: string
  timezone?: string
  autonomy_mode?: AutonomyMode
  notification_preferences?: NotificationPreferences
  onboarding_completed?: boolean
  onboarding_step?: number
  // Aliases for convenience
  autonomyMode?: AutonomyMode
  notifications?: boolean
  onboarded?: boolean
}

export type AutonomyMode = 'suggest' | 'supervised' | 'autonomous' | 'full_autonomous' | 'ask' | 'auto'

export interface NotificationPreferences {
  email_enabled: boolean
  push_enabled: boolean
  digest_frequency: 'realtime' | 'hourly' | 'daily' | 'weekly'
}

// Notifications
export interface Notification {
  id: string
  type: NotificationType
  title: string
  message: string
  is_read: boolean
  read?: boolean // alias
  timestamp?: string // alias for created_at
  created_at: string
  action_url?: string
  metadata?: Record<string, unknown>
}

export type NotificationType =
  | 'info'
  | 'success'
  | 'warning'
  | 'error'
  | 'action_required'
  | 'recommendation'

// Stats
export interface Stats {
  total_items: number
  total_memories: number
  active_hats: number
  connected_spaces: number
  agent_status: 'active' | 'idle' | 'sleeping'
  items_by_status: Record<ItemStatus, number>
}

// Proactive
export interface Recommendation {
  id: string
  type: string
  title: string
  description: string
  confidence: number
  hat_id?: string
  action?: RecommendedAction
  created_at: string
}

export interface RecommendedAction {
  type: string
  label: string
  params?: Record<string, unknown>
}

export interface Nudge {
  id: string
  message: string
  priority: number
  created_at: string
  expires_at?: string
}

// Trust
export interface TrustScore {
  domain: TrustDomain
  score: number
  state: TrustState
  action_count: number
  last_action_at?: string
}

export type TrustDomain =
  | 'email'
  | 'calendar'
  | 'tasks'
  | 'finance'
  | 'communication'
  | 'health'
  | 'mesh'
  | 'general'

export type TrustState =
  | 'probation'
  | 'learning'
  | 'trusted'
  | 'verified'
  | 'restricted'

// Ledger (audit trail)
export interface LedgerEntry {
  id: string
  timestamp: string
  type: string
  domain: string
  action: string
  actor: 'agent' | 'user' | 'system'
  result: 'success' | 'failure'
  details?: Record<string, unknown>
  hash: string
}

// Setup
export interface SetupStatus {
  identity_created: boolean
  gmail_connected: boolean
  calendar_connected: boolean
  finance_connected: boolean
  completed_at?: string
  current_step: number
  total_steps: number
}

// API Response types
export interface ApiResponse<T> {
  data?: T
  error?: string
  message?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
  has_more: boolean
}

// WebSocket message types
export interface WSMessage {
  type: string
  data: unknown
  timestamp: string
}

// Form state
export interface FormState<T> {
  values: T
  errors: Partial<Record<keyof T, string>>
  touched: Partial<Record<keyof T, boolean>>
  isSubmitting: boolean
  isValid: boolean
}

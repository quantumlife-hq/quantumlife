import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { clsx } from 'clsx'
import {
  MagnifyingGlassIcon,
  PlusIcon,
  ArrowPathIcon,
  TrashIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  FunnelIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection, CardGrid } from '@/components/layout'
import {
  Card,
  Button,
  IconButton,
  SearchInput,
  Badge,
  StatusBadge,
  Modal,
  ModalBody,
  ModalFooter,
  Spinner,
  SkeletonCard,
  EmptyState,
  NoConnectionsState,
  useToast,
  ConfirmModal,
} from '@/components/ui'
import type { Provider, Connection } from '@/types'

// Provider categories for filtering
const categories = [
  { id: 'all', label: 'All Services', icon: '🌐' },
  { id: 'email', label: 'Email', icon: '📧' },
  { id: 'calendar', label: 'Calendar', icon: '📅' },
  { id: 'communication', label: 'Communication', icon: '💬' },
  { id: 'productivity', label: 'Productivity', icon: '📝' },
  { id: 'finance', label: 'Finance', icon: '💰' },
  { id: 'development', label: 'Development', icon: '💻' },
  { id: 'social', label: 'Social', icon: '🌍' },
  { id: 'storage', label: 'Storage', icon: '☁️' },
  { id: 'crm', label: 'CRM', icon: '👥' },
]

export function ConnectionsPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('all')
  const [showConnectModal, setShowConnectModal] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null)
  const [disconnectTarget, setDisconnectTarget] = useState<Connection | null>(null)

  const toast = useToast()
  const queryClient = useQueryClient()

  // Fetch connections
  const { data: connections, isLoading: connectionsLoading } = useQuery({
    queryKey: ['connections'],
    queryFn: api.connections.list,
  })

  // Fetch available providers
  const { data: providers, isLoading: providersLoading } = useQuery({
    queryKey: ['providers'],
    queryFn: api.connections.providers,
  })

  // Connect mutation
  const connectMutation = useMutation({
    mutationFn: (providerKey: string) => api.connections.connect(providerKey),
    onSuccess: (data) => {
      // Redirect to OAuth flow
      if (data.auth_url) {
        window.location.href = data.auth_url
      }
    },
    onError: (error: Error) => {
      toast.error('Connection failed', error.message)
    },
  })

  // Disconnect mutation
  const disconnectMutation = useMutation({
    mutationFn: (spaceId: string) => api.connections.disconnect(spaceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['connections'] })
      toast.success('Disconnected', 'Service has been disconnected successfully')
      setDisconnectTarget(null)
    },
    onError: (error: Error) => {
      toast.error('Disconnect failed', error.message)
    },
  })

  // Filter providers
  const filteredProviders = useMemo(() => {
    if (!providers) return []

    return providers.filter((provider) => {
      const matchesSearch =
        provider.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        provider.category.toLowerCase().includes(searchQuery.toLowerCase())

      const matchesCategory =
        selectedCategory === 'all' || provider.category === selectedCategory

      return matchesSearch && matchesCategory
    })
  }, [providers, searchQuery, selectedCategory])

  // Group connections by provider
  const connectionsByProvider = useMemo(() => {
    if (!connections) return new Map<string, Connection[]>()

    return connections.reduce((acc, conn) => {
      const existing = acc.get(conn.provider) || []
      acc.set(conn.provider, [...existing, conn])
      return acc
    }, new Map<string, Connection[]>())
  }, [connections])

  const handleConnect = (provider: Provider) => {
    setSelectedProvider(provider)
    setShowConnectModal(true)
  }

  const confirmConnect = () => {
    if (selectedProvider) {
      connectMutation.mutate(selectedProvider.key)
      setShowConnectModal(false)
    }
  }

  return (
    <PageContent>
      {/* Header Stats */}
      <PageSection>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <StatsCard
            label="Connected Services"
            value={connections?.length || 0}
            icon="🔗"
            variant="primary"
          />
          <StatsCard
            label="Available Integrations"
            value={providers?.length || 0}
            icon="🌐"
            variant="info"
          />
          <StatsCard
            label="Categories"
            value={categories.length - 1}
            icon="📂"
            variant="success"
          />
        </div>
      </PageSection>

      {/* Connected Services */}
      <PageSection
        title="Connected Services"
        description="Your active service integrations"
      >
        {connectionsLoading ? (
          <CardGrid columns={3}>
            {[1, 2, 3].map((i) => (
              <SkeletonCard key={i} showAvatar showAction />
            ))}
          </CardGrid>
        ) : !connections?.length ? (
          <Card padding="lg">
            <NoConnectionsState onConnect={() => setSelectedCategory('all')} />
          </Card>
        ) : (
          <CardGrid columns={3}>
            <AnimatePresence mode="popLayout">
              {connections.map((connection) => (
                <ConnectionCard
                  key={connection.id}
                  connection={connection}
                  onDisconnect={() => setDisconnectTarget(connection)}
                />
              ))}
            </AnimatePresence>
          </CardGrid>
        )}
      </PageSection>

      {/* Available Providers */}
      <PageSection
        title="Add New Connection"
        description="Browse and connect to 500+ services"
      >
        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          <SearchInput
            placeholder="Search services..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="sm:w-80"
          />
          <div className="flex items-center gap-2 overflow-x-auto pb-2 sm:pb-0">
            <FunnelIcon className="w-4 h-4 text-gray-500 flex-shrink-0" />
            {categories.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setSelectedCategory(cat.id)}
                className={clsx(
                  'px-3 py-1.5 rounded-full text-sm font-medium whitespace-nowrap transition-all',
                  selectedCategory === cat.id
                    ? 'bg-primary-500/20 text-primary-300 border border-primary-500/30'
                    : 'bg-surface-light/20 text-gray-400 border border-transparent hover:border-surface-border'
                )}
              >
                <span className="mr-1.5">{cat.icon}</span>
                {cat.label}
              </button>
            ))}
          </div>
        </div>

        {/* Provider Grid */}
        {providersLoading ? (
          <CardGrid columns={4}>
            {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
              <SkeletonCard key={i} lines={2} />
            ))}
          </CardGrid>
        ) : filteredProviders.length === 0 ? (
          <Card padding="lg">
            <EmptyState
              icon={<MagnifyingGlassIcon className="w-8 h-8" />}
              title="No services found"
              description="Try adjusting your search or filter criteria"
              action={
                <Button
                  variant="ghost"
                  onClick={() => {
                    setSearchQuery('')
                    setSelectedCategory('all')
                  }}
                >
                  Clear filters
                </Button>
              }
            />
          </Card>
        ) : (
          <CardGrid columns={4}>
            <AnimatePresence mode="popLayout">
              {filteredProviders.map((provider) => (
                <ProviderCard
                  key={provider.key}
                  provider={provider}
                  isConnected={connectionsByProvider.has(provider.key)}
                  connectionCount={connectionsByProvider.get(provider.key)?.length || 0}
                  onConnect={() => handleConnect(provider)}
                />
              ))}
            </AnimatePresence>
          </CardGrid>
        )}
      </PageSection>

      {/* Connect Modal */}
      <Modal
        isOpen={showConnectModal}
        onClose={() => setShowConnectModal(false)}
        title={`Connect ${selectedProvider?.name}`}
        size="sm"
      >
        <ModalBody>
          <div className="text-center py-4">
            <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-surface-lighter flex items-center justify-center text-3xl">
              {selectedProvider?.icon}
            </div>
            <p className="text-gray-300 mb-4">
              You'll be redirected to {selectedProvider?.name} to authorize access.
            </p>
            <div className="text-sm text-gray-500 space-y-2">
              <p>QuantumLife will be able to:</p>
              <ul className="text-left pl-4 space-y-1">
                <li className="flex items-center gap-2">
                  <CheckCircleIcon className="w-4 h-4 text-emerald-400" />
                  <span>Read and manage your {selectedProvider?.category} data</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircleIcon className="w-4 h-4 text-emerald-400" />
                  <span>Perform actions on your behalf (with approval)</span>
                </li>
              </ul>
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setShowConnectModal(false)}>
            Cancel
          </Button>
          <Button
            onClick={confirmConnect}
            isLoading={connectMutation.isPending}
          >
            <PlusIcon className="w-4 h-4 mr-2" />
            Connect
          </Button>
        </ModalFooter>
      </Modal>

      {/* Disconnect Confirmation */}
      <ConfirmModal
        isOpen={!!disconnectTarget}
        onClose={() => setDisconnectTarget(null)}
        onConfirm={() => disconnectTarget && disconnectMutation.mutate(disconnectTarget.id)}
        title="Disconnect Service"
        message={`Are you sure you want to disconnect ${disconnectTarget?.name}? This will revoke access and remove any associated data.`}
        confirmText="Disconnect"
        variant="danger"
        isLoading={disconnectMutation.isPending}
      />
    </PageContent>
  )
}

// Stats Card Component
function StatsCard({
  label,
  value,
  icon,
  variant,
}: {
  label: string
  value: number
  icon: string
  variant: 'primary' | 'info' | 'success'
}) {
  const variants = {
    primary: 'from-primary-500/20 to-violet-500/20 border-primary-500/30',
    info: 'from-blue-500/20 to-cyan-500/20 border-blue-500/30',
    success: 'from-emerald-500/20 to-teal-500/20 border-emerald-500/30',
  }

  return (
    <Card
      className={clsx('bg-gradient-to-br border', variants[variant])}
      padding="md"
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-400">{label}</p>
          <p className="text-2xl font-bold text-white mt-1">{value}</p>
        </div>
        <span className="text-3xl">{icon}</span>
      </div>
    </Card>
  )
}

// Connected Service Card
function ConnectionCard({
  connection,
  onDisconnect,
}: {
  connection: Connection
  onDisconnect: () => void
}) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
    >
      <Card hover padding="md">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-xl bg-surface-lighter flex items-center justify-center text-2xl">
              {connection.icon || '🔗'}
            </div>
            <div>
              <h3 className="font-medium text-white">{connection.name}</h3>
              <p className="text-sm text-gray-500">{connection.email || connection.provider}</p>
            </div>
          </div>
          <StatusBadge status={connection.status === 'connected' ? 'connected' : 'disconnected'} />
        </div>

        <div className="mt-4 pt-4 border-t border-surface-border flex items-center justify-between">
          <div className="text-sm text-gray-500">
            {connection.itemCount || 0} items synced
          </div>
          <div className="flex items-center gap-2">
            <IconButton
              icon={<ArrowPathIcon className="w-4 h-4" />}
              aria-label="Refresh"
              variant="ghost"
              size="sm"
            />
            <IconButton
              icon={<TrashIcon className="w-4 h-4" />}
              aria-label="Disconnect"
              variant="ghost"
              size="sm"
              onClick={onDisconnect}
            />
          </div>
        </div>
      </Card>
    </motion.div>
  )
}

// Provider Card
function ProviderCard({
  provider,
  isConnected,
  connectionCount,
  onConnect,
}: {
  provider: Provider
  isConnected: boolean
  connectionCount: number
  onConnect: () => void
}) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
    >
      <Card
        hover
        padding="md"
        className={clsx(
          isConnected && 'border-emerald-500/30 bg-emerald-500/5'
        )}
      >
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-surface-lighter flex items-center justify-center text-xl">
            {provider.icon}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-medium text-white truncate">{provider.name}</h3>
              {isConnected && (
                <CheckCircleIcon className="w-4 h-4 text-emerald-400 flex-shrink-0" />
              )}
            </div>
            <Badge variant="neutral" size="sm" className="mt-1">
              {provider.category}
            </Badge>
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between">
          {isConnected ? (
            <span className="text-xs text-emerald-400">
              {connectionCount} connected
            </span>
          ) : (
            <span className="text-xs text-gray-500">Not connected</span>
          )}
          <Button
            size="sm"
            variant={isConnected ? 'secondary' : 'primary'}
            onClick={onConnect}
          >
            {isConnected ? 'Add another' : 'Connect'}
          </Button>
        </div>
      </Card>
    </motion.div>
  )
}

export default ConnectionsPage

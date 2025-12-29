import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { clsx } from 'clsx'
import {
  SparklesIcon,
  BoltIcon,
  CheckIcon,
  XMarkIcon,
  ClockIcon,
  ChevronRightIcon,
  AdjustmentsHorizontalIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection, CardGrid } from '@/components/layout'
import {
  Card,
  CardHeader,
  CardContent,
  Button,
  IconButton,
  Badge,
  Modal,
  ModalBody,
  ModalFooter,
  Spinner,
  SkeletonCard,
  EmptyState,
  useToast,
} from '@/components/ui'

interface Suggestion {
  id: string
  type: 'action' | 'reminder' | 'insight' | string
  title: string
  description: string
  source: string
  priority: 'high' | 'medium' | 'low'
  actions: Array<{
    id: string
    label: string
    type: 'approve' | 'reject' | 'defer'
  }>
  context?: Record<string, unknown>
  createdAt: string
  created_at?: string
}

export function ProactivePage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const [selectedSuggestion, setSelectedSuggestion] = useState<Suggestion | null>(null)

  // Fetch suggestions
  const { data: suggestions, isLoading } = useQuery({
    queryKey: ['proactive', 'suggestions'],
    queryFn: api.proactive.getSuggestions,
  })

  // Act on suggestion mutation
  const actMutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: string }) =>
      api.proactive.actOnSuggestion(id, action),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proactive'] })
      toast.success('Action completed', 'The suggestion has been processed')
      setSelectedSuggestion(null)
    },
    onError: () => {
      toast.error('Action failed', 'Please try again')
    },
  })

  // Group suggestions by priority
  const highPriority = suggestions?.filter(s => s.priority === 'high') || []
  const mediumPriority = suggestions?.filter(s => s.priority === 'medium') || []
  const lowPriority = suggestions?.filter(s => s.priority === 'low') || []

  return (
    <PageContent>
      {/* Header */}
      <PageSection>
        <Card className="bg-gradient-to-r from-violet-500/20 via-purple-500/20 to-fuchsia-500/20 border-violet-500/30">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-violet-500/30 flex items-center justify-center">
                <SparklesIcon className="w-6 h-6 text-violet-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Proactive Intelligence</h2>
                <p className="text-gray-400">
                  AI-powered suggestions and automated actions
                </p>
              </div>
            </div>
            <Button variant="secondary" size="sm">
              <AdjustmentsHorizontalIcon className="w-4 h-4 mr-2" />
              Configure
            </Button>
          </div>
        </Card>
      </PageSection>

      {/* Stats */}
      <PageSection>
        <CardGrid columns={3}>
          <StatCard
            label="Pending Suggestions"
            value={suggestions?.length || 0}
            icon="💡"
            color="violet"
          />
          <StatCard
            label="Auto-completed Today"
            value={12}
            icon="✓"
            color="emerald"
          />
          <StatCard
            label="Time Saved"
            value="2.5h"
            icon="⏱️"
            color="blue"
          />
        </CardGrid>
      </PageSection>

      {/* High Priority */}
      {highPriority.length > 0 && (
        <PageSection
          title="Requires Attention"
          description="High priority items that need your input"
        >
          <CardGrid columns={1}>
            <AnimatePresence mode="popLayout">
              {highPriority.map((suggestion) => (
                <SuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  onView={() => setSelectedSuggestion(suggestion)}
                  onAct={(action) => actMutation.mutate({ id: suggestion.id, action })}
                />
              ))}
            </AnimatePresence>
          </CardGrid>
        </PageSection>
      )}

      {/* Medium & Low Priority */}
      <PageSection
        title="Suggestions"
        description="Recommendations from your AI assistant"
      >
        {isLoading ? (
          <CardGrid columns={2}>
            {[1, 2, 3, 4].map((i) => (
              <SkeletonCard key={i} lines={3} showAction />
            ))}
          </CardGrid>
        ) : !suggestions?.length ? (
          <Card padding="lg">
            <EmptyState
              icon={<SparklesIcon className="w-8 h-8" />}
              title="No suggestions"
              description="Your AI assistant will provide suggestions as it learns your patterns"
            />
          </Card>
        ) : (
          <CardGrid columns={2}>
            <AnimatePresence mode="popLayout">
              {[...mediumPriority, ...lowPriority].map((suggestion) => (
                <SuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  onView={() => setSelectedSuggestion(suggestion)}
                  onAct={(action) => actMutation.mutate({ id: suggestion.id, action })}
                  compact
                />
              ))}
            </AnimatePresence>
          </CardGrid>
        )}
      </PageSection>

      {/* Detail Modal */}
      <Modal
        isOpen={!!selectedSuggestion}
        onClose={() => setSelectedSuggestion(null)}
        title={selectedSuggestion?.title}
        size="md"
      >
        {selectedSuggestion && (
          <>
            <ModalBody>
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      selectedSuggestion.priority === 'high'
                        ? 'error'
                        : selectedSuggestion.priority === 'medium'
                        ? 'warning'
                        : 'neutral'
                    }
                  >
                    {selectedSuggestion.priority} priority
                  </Badge>
                  <Badge variant="info">{selectedSuggestion.type}</Badge>
                </div>

                <p className="text-gray-300">{selectedSuggestion.description}</p>

                <div className="p-4 rounded-lg bg-surface-light/20 border border-surface-border">
                  <div className="text-sm text-gray-400 mb-2">Source</div>
                  <div className="text-white">{selectedSuggestion.source}</div>
                </div>

                {selectedSuggestion.context && (
                  <div className="p-4 rounded-lg bg-surface-light/20 border border-surface-border">
                    <div className="text-sm text-gray-400 mb-2">Context</div>
                    <pre className="text-sm text-gray-300 overflow-x-auto">
                      {JSON.stringify(selectedSuggestion.context, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            </ModalBody>
            <ModalFooter>
              <Button
                variant="ghost"
                onClick={() => {
                  actMutation.mutate({ id: selectedSuggestion.id, action: 'defer' })
                }}
              >
                <ClockIcon className="w-4 h-4 mr-2" />
                Defer
              </Button>
              <Button
                variant="danger"
                onClick={() => {
                  actMutation.mutate({ id: selectedSuggestion.id, action: 'reject' })
                }}
              >
                <XMarkIcon className="w-4 h-4 mr-2" />
                Dismiss
              </Button>
              <Button
                onClick={() => {
                  actMutation.mutate({ id: selectedSuggestion.id, action: 'approve' })
                }}
                isLoading={actMutation.isPending}
              >
                <CheckIcon className="w-4 h-4 mr-2" />
                Approve
              </Button>
            </ModalFooter>
          </>
        )}
      </Modal>
    </PageContent>
  )
}

// Stat Card
function StatCard({
  label,
  value,
  icon,
  color,
}: {
  label: string
  value: string | number
  icon: string
  color: 'violet' | 'emerald' | 'blue'
}) {
  const colorStyles = {
    violet: 'from-violet-500/20 to-purple-500/20 border-violet-500/30',
    emerald: 'from-emerald-500/20 to-teal-500/20 border-emerald-500/30',
    blue: 'from-blue-500/20 to-cyan-500/20 border-blue-500/30',
  }

  return (
    <Card className={clsx('bg-gradient-to-br border', colorStyles[color])}>
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

// Suggestion Card
function SuggestionCard({
  suggestion,
  onView,
  onAct,
  compact = false,
}: {
  suggestion: Suggestion
  onView: () => void
  onAct: (action: string) => void
  compact?: boolean
}) {
  const typeIcons: Record<string, string> = {
    action: '⚡',
    reminder: '🔔',
    insight: '💡',
  }

  const priorityColors = {
    high: 'border-red-500/30 bg-red-500/5',
    medium: 'border-amber-500/30 bg-amber-500/5',
    low: 'border-surface-border',
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
    >
      <Card className={clsx('border', priorityColors[suggestion.priority])}>
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 rounded-lg bg-surface-lighter flex items-center justify-center text-xl flex-shrink-0">
            {typeIcons[suggestion.type]}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <h3 className="font-medium text-white">{suggestion.title}</h3>
              <Badge
                variant={
                  suggestion.priority === 'high'
                    ? 'error'
                    : suggestion.priority === 'medium'
                    ? 'warning'
                    : 'neutral'
                }
                size="sm"
              >
                {suggestion.priority}
              </Badge>
            </div>
            {!compact && (
              <p className="text-sm text-gray-400 mt-1 line-clamp-2">
                {suggestion.description}
              </p>
            )}
            <div className="flex items-center gap-2 mt-3">
              <span className="text-xs text-gray-500">{suggestion.source}</span>
              <span className="text-gray-600">•</span>
              <span className="text-xs text-gray-500">
                {formatTimeAgo(suggestion.createdAt)}
              </span>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between mt-4 pt-4 border-t border-surface-border">
          <Button variant="ghost" size="sm" onClick={onView}>
            View details
            <ChevronRightIcon className="w-4 h-4 ml-1" />
          </Button>
          <div className="flex items-center gap-2">
            <IconButton
              icon={<XMarkIcon className="w-4 h-4" />}
              aria-label="Dismiss"
              variant="ghost"
              size="sm"
              onClick={() => onAct('reject')}
            />
            <IconButton
              icon={<ClockIcon className="w-4 h-4" />}
              aria-label="Defer"
              variant="ghost"
              size="sm"
              onClick={() => onAct('defer')}
            />
            <Button size="sm" onClick={() => onAct('approve')}>
              <CheckIcon className="w-4 h-4 mr-1" />
              Approve
            </Button>
          </div>
        </div>
      </Card>
    </motion.div>
  )
}

function formatTimeAgo(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`

  return date.toLocaleDateString()
}

export default ProactivePage

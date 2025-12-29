import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { clsx } from 'clsx'
import {
  BookOpenIcon,
  AcademicCapIcon,
  LightBulbIcon,
  TrashIcon,
  PlusIcon,
  SparklesIcon,
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
  Progress,
  Spinner,
  SkeletonCard,
  EmptyState,
  useToast,
} from '@/components/ui'

export function LearningPage() {
  const toast = useToast()
  const queryClient = useQueryClient()

  // Fetch learning data
  const { data: patterns, isLoading: patternsLoading } = useQuery({
    queryKey: ['learning', 'patterns'],
    queryFn: api.learning.getPatterns,
  })

  const { data: preferences, isLoading: preferencesLoading } = useQuery({
    queryKey: ['learning', 'preferences'],
    queryFn: api.learning.getPreferences,
  })

  // Delete pattern mutation
  const deletePatternMutation = useMutation({
    mutationFn: (patternId: string) => api.learning.deletePattern(patternId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['learning'] })
      toast.success('Pattern removed', 'The learned pattern has been deleted')
    },
    onError: () => {
      toast.error('Error', 'Failed to delete pattern')
    },
  })

  const isLoading = patternsLoading || preferencesLoading

  return (
    <PageContent>
      {/* Header */}
      <PageSection>
        <Card className="bg-gradient-to-r from-blue-500/20 via-cyan-500/20 to-teal-500/20 border-blue-500/30">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-blue-500/30 flex items-center justify-center">
                <AcademicCapIcon className="w-6 h-6 text-blue-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Learning Hub</h2>
                <p className="text-gray-400">
                  See what your AI assistant has learned about you
                </p>
              </div>
            </div>
            <Badge variant="info" size="md">
              <SparklesIcon className="w-4 h-4 mr-1" />
              {patterns?.length || 0} patterns learned
            </Badge>
          </div>
        </Card>
      </PageSection>

      {/* Stats */}
      <PageSection>
        <CardGrid columns={3}>
          <StatCard
            label="Behavioral Patterns"
            value={patterns?.filter(p => p.type === 'behavioral').length || 0}
            icon="🧠"
            color="blue"
          />
          <StatCard
            label="Preferences"
            value={preferences?.length || 0}
            icon="❤️"
            color="rose"
          />
          <StatCard
            label="Learning Accuracy"
            value="94%"
            icon="🎯"
            color="emerald"
          />
        </CardGrid>
      </PageSection>

      {/* Learned Patterns */}
      <PageSection
        title="Learned Patterns"
        description="Behaviors and patterns your AI has observed"
      >
        {isLoading ? (
          <CardGrid columns={2}>
            {[1, 2, 3, 4].map((i) => (
              <SkeletonCard key={i} lines={3} showAction />
            ))}
          </CardGrid>
        ) : !patterns?.length ? (
          <Card padding="lg">
            <EmptyState
              icon={<BookOpenIcon className="w-8 h-8" />}
              title="No patterns yet"
              description="Your AI assistant will learn from your interactions over time"
            />
          </Card>
        ) : (
          <CardGrid columns={2}>
            <AnimatePresence mode="popLayout">
              {patterns.map((pattern: Pattern) => (
                <PatternCard
                  key={pattern.id}
                  pattern={{
                    ...pattern,
                    description: pattern.description || pattern.pattern || 'Pattern detected',
                    occurrences: pattern.occurrences || 1,
                    lastSeen: pattern.lastSeen || pattern.created_at || new Date().toISOString(),
                  }}
                  onDelete={() => deletePatternMutation.mutate(pattern.id)}
                />
              ))}
            </AnimatePresence>
          </CardGrid>
        )}
      </PageSection>

      {/* Preferences */}
      <PageSection
        title="Learned Preferences"
        description="Your preferences across different contexts"
      >
        {preferencesLoading ? (
          <Card>
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-4 animate-pulse">
                  <div className="w-10 h-10 rounded-lg bg-surface-lighter" />
                  <div className="flex-1">
                    <div className="h-4 bg-surface-lighter rounded w-1/2 mb-2" />
                    <div className="h-3 bg-surface-lighter rounded w-3/4" />
                  </div>
                </div>
              ))}
            </div>
          </Card>
        ) : !preferences?.length ? (
          <Card padding="lg">
            <EmptyState
              icon={<LightBulbIcon className="w-8 h-8" />}
              title="No preferences learned"
              description="Preferences will be discovered as you use QuantumLife"
            />
          </Card>
        ) : (
          <Card>
            <ul className="divide-y divide-surface-border">
              {preferences.map((pref) => (
                <PreferenceItem key={pref.id} preference={pref} />
              ))}
            </ul>
          </Card>
        )}
      </PageSection>
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
  color: 'blue' | 'rose' | 'emerald'
}) {
  const colorStyles = {
    blue: 'from-blue-500/20 to-cyan-500/20 border-blue-500/30',
    rose: 'from-rose-500/20 to-pink-500/20 border-rose-500/30',
    emerald: 'from-emerald-500/20 to-teal-500/20 border-emerald-500/30',
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

// Pattern Card
interface Pattern {
  id: string
  type: string
  description?: string
  pattern?: string
  confidence: number
  occurrences?: number
  lastSeen?: string
  created_at?: string
  context?: string
  hat_id?: string
}

function PatternCard({
  pattern,
  onDelete,
}: {
  pattern: Pattern
  onDelete: () => void
}) {
  const typeIcons: Record<string, string> = {
    behavioral: '🧠',
    temporal: '⏰',
    contextual: '📍',
    preference: '❤️',
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
    >
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <span className="text-2xl">{typeIcons[pattern.type] || '📊'}</span>
              <div>
                <Badge variant="info" size="sm">
                  {pattern.type}
                </Badge>
              </div>
            </div>
            <IconButton
              icon={<TrashIcon className="w-4 h-4" />}
              aria-label="Delete pattern"
              variant="ghost"
              size="sm"
              onClick={onDelete}
            />
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-white mb-3">{pattern.description}</p>

          <div className="space-y-3">
            <div>
              <div className="flex items-center justify-between text-sm mb-1">
                <span className="text-gray-400">Confidence</span>
                <span className="text-white">{pattern.confidence}%</span>
              </div>
              <Progress value={pattern.confidence} size="sm" variant="primary" />
            </div>

            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-400">Occurrences</span>
              <span className="text-white">{pattern.occurrences}</span>
            </div>

            {pattern.context && (
              <div className="text-sm">
                <span className="text-gray-400">Context: </span>
                <span className="text-gray-300">{pattern.context}</span>
              </div>
            )}

            <div className="text-xs text-gray-500">
              Last seen: {pattern.lastSeen ? new Date(pattern.lastSeen).toLocaleDateString() : 'N/A'}
            </div>
          </div>
        </CardContent>
      </Card>
    </motion.div>
  )
}

// Preference Item
interface Preference {
  id: string
  category: string
  key: string
  value: string
  confidence: number
}

function PreferenceItem({ preference }: { preference: Preference }) {
  return (
    <li className="flex items-center gap-4 py-4 px-2">
      <div className="w-10 h-10 rounded-lg bg-surface-lighter flex items-center justify-center">
        <LightBulbIcon className="w-5 h-5 text-amber-400" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium text-white">{preference.key}</h4>
          <Badge variant="neutral" size="sm">{preference.category}</Badge>
        </div>
        <p className="text-sm text-gray-400 mt-0.5">{preference.value}</p>
      </div>
      <div className="text-right">
        <div className="text-sm text-white">{preference.confidence}%</div>
        <div className="text-xs text-gray-500">confidence</div>
      </div>
    </li>
  )
}

export default LearningPage

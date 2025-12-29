import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { clsx } from 'clsx'
import {
  ShieldCheckIcon,
  ArrowTrendingUpIcon,
  ClockIcon,
  CheckCircleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection, CardGrid, TwoColumnLayout } from '@/components/layout'
import {
  Card,
  CardHeader,
  CardContent,
  CardFooter,
  Badge,
  Progress,
  ProgressRing,
  Spinner,
  SkeletonCard,
  EmptyState,
} from '@/components/ui'

export function TrustPage() {
  // Fetch trust capital data
  const { data: trustCapital, isLoading: capitalLoading } = useQuery({
    queryKey: ['trust', 'capital'],
    queryFn: api.trust.getCapital,
  })

  // Fetch trust history
  const { data: trustHistory, isLoading: historyLoading } = useQuery({
    queryKey: ['trust', 'history'],
    queryFn: api.trust.getHistory,
  })

  const isLoading = capitalLoading || historyLoading

  return (
    <PageContent>
      {/* Header */}
      <PageSection>
        <Card className="bg-gradient-to-r from-emerald-500/20 via-teal-500/20 to-cyan-500/20 border-emerald-500/30">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-emerald-500/30 flex items-center justify-center">
                <ShieldCheckIcon className="w-6 h-6 text-emerald-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Trust Management</h2>
                <p className="text-gray-400">
                  Monitor and control AI autonomy levels
                </p>
              </div>
            </div>
            <div className="text-right">
              <div className="text-sm text-gray-400">Current Level</div>
              <div className="text-2xl font-bold text-emerald-400">
                Level {trustCapital?.currentLevel || 0}
              </div>
            </div>
          </div>
        </Card>
      </PageSection>

      <TwoColumnLayout
        sidebar={<TrustSidebar trustCapital={trustCapital} isLoading={capitalLoading} />}
        sidebarWidth="md"
      >
        {/* Trust Capital Overview */}
        <PageSection title="Trust Capital" description="Your accumulated trust score">
          {capitalLoading ? (
            <SkeletonCard lines={4} />
          ) : (
            <Card>
              <div className="flex items-center justify-center py-8">
                <ProgressRing
                  value={trustCapital?.totalScore || 0}
                  max={trustCapital?.maxScore || 10000}
                  size={160}
                  strokeWidth={12}
                  variant="success"
                  label={`of ${trustCapital?.maxScore?.toLocaleString() || '10,000'}`}
                />
              </div>

              <div className="grid grid-cols-3 gap-4 mt-6">
                <TrustMetric
                  label="Approved"
                  value={trustCapital?.approvedActions || 0}
                  icon={<CheckCircleIcon className="w-5 h-5 text-emerald-400" />}
                />
                <TrustMetric
                  label="Rejected"
                  value={trustCapital?.rejectedActions || 0}
                  icon={<XCircleIcon className="w-5 h-5 text-red-400" />}
                />
                <TrustMetric
                  label="Auto-approved"
                  value={trustCapital?.autoApproved || 0}
                  icon={<ClockIcon className="w-5 h-5 text-blue-400" />}
                />
              </div>
            </Card>
          )}
        </PageSection>

        {/* Trust History */}
        <PageSection title="Recent Trust Events" description="Actions that affected your trust score">
          {historyLoading ? (
            <Card>
              <div className="space-y-4">
                {[1, 2, 3, 4, 5].map((i) => (
                  <div key={i} className="flex items-center gap-4 animate-pulse">
                    <div className="w-10 h-10 rounded-lg bg-surface-lighter" />
                    <div className="flex-1">
                      <div className="h-4 bg-surface-lighter rounded w-3/4 mb-2" />
                      <div className="h-3 bg-surface-lighter rounded w-1/2" />
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          ) : !trustHistory?.length ? (
            <Card padding="lg">
              <EmptyState
                icon={<ArrowTrendingUpIcon className="w-8 h-8" />}
                title="No trust history"
                description="Trust events will appear here as you interact with your AI assistant"
              />
            </Card>
          ) : (
            <Card padding="none">
              <ul className="divide-y divide-surface-border">
                {trustHistory.map((event) => (
                  <TrustEvent key={event.id} event={event} />
                ))}
              </ul>
            </Card>
          )}
        </PageSection>
      </TwoColumnLayout>
    </PageContent>
  )
}

// Trust Sidebar
function TrustSidebar({
  trustCapital,
  isLoading,
}: {
  trustCapital: any
  isLoading: boolean
}) {
  const levels = [
    { level: 1, name: 'Observer', threshold: 0, color: 'gray' },
    { level: 2, name: 'Helper', threshold: 1000, color: 'blue' },
    { level: 3, name: 'Assistant', threshold: 3000, color: 'violet' },
    { level: 4, name: 'Partner', threshold: 6000, color: 'emerald' },
    { level: 5, name: 'Autopilot', threshold: 10000, color: 'amber' },
  ]

  const currentLevel = trustCapital?.currentLevel || 1
  const currentScore = trustCapital?.totalScore || 0

  return (
    <div className="space-y-6">
      {/* Trust Levels */}
      <Card>
        <CardHeader title="Trust Levels" subtitle="Unlock more autonomy" />
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 bg-surface-lighter rounded-lg animate-pulse" />
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              {levels.map((level) => (
                <LevelCard
                  key={level.level}
                  level={level}
                  isCurrent={level.level === currentLevel}
                  isUnlocked={currentScore >= level.threshold}
                  progress={
                    level.level === currentLevel
                      ? ((currentScore - level.threshold) /
                          ((levels[level.level]?.threshold || 10000) - level.threshold)) *
                        100
                      : level.level < currentLevel
                      ? 100
                      : 0
                  }
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Trust Tips */}
      <Card>
        <CardHeader title="How to Earn Trust" />
        <CardContent>
          <ul className="space-y-3 text-sm">
            <TrustTip
              icon="✓"
              text="Approve AI suggestions"
              points="+10"
            />
            <TrustTip
              icon="🎯"
              text="Accurate AI predictions"
              points="+5"
            />
            <TrustTip
              icon="⏰"
              text="Consistent usage patterns"
              points="+2"
            />
            <TrustTip
              icon="🔗"
              text="Connect new services"
              points="+25"
            />
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}

// Level Card
function LevelCard({
  level,
  isCurrent,
  isUnlocked,
  progress,
}: {
  level: { level: number; name: string; threshold: number; color: string }
  isCurrent: boolean
  isUnlocked: boolean
  progress: number
}) {
  return (
    <div
      className={clsx(
        'p-3 rounded-lg border transition-all',
        isCurrent
          ? 'bg-primary-500/20 border-primary-500/50'
          : isUnlocked
          ? 'bg-surface-light/20 border-emerald-500/30'
          : 'bg-surface-light/10 border-surface-border opacity-50'
      )}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-white">Level {level.level}</span>
          <span className="text-sm text-gray-400">{level.name}</span>
        </div>
        {isUnlocked && !isCurrent && (
          <CheckCircleIcon className="w-4 h-4 text-emerald-400" />
        )}
        {isCurrent && (
          <Badge variant="primary" size="sm">Current</Badge>
        )}
      </div>
      {isCurrent && (
        <Progress value={progress} size="sm" variant="primary" />
      )}
      <div className="text-xs text-gray-500 mt-1">
        {level.threshold.toLocaleString()} trust points
      </div>
    </div>
  )
}

// Trust Tip
function TrustTip({ icon, text, points }: { icon: string; text: string; points: string }) {
  return (
    <li className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <span>{icon}</span>
        <span className="text-gray-400">{text}</span>
      </div>
      <span className="text-emerald-400 font-medium">{points}</span>
    </li>
  )
}

// Trust Metric
function TrustMetric({
  label,
  value,
  icon,
}: {
  label: string
  value: number
  icon: React.ReactNode
}) {
  return (
    <div className="text-center p-4 rounded-lg bg-surface-light/20">
      <div className="flex justify-center mb-2">{icon}</div>
      <div className="text-xl font-bold text-white">{value.toLocaleString()}</div>
      <div className="text-xs text-gray-500">{label}</div>
    </div>
  )
}

// Trust Event
interface TrustEventData {
  id: string
  type: 'approval' | 'rejection' | 'auto_approve' | 'correction'
  description: string
  points: number
  timestamp: string
  source: string
}

function TrustEvent({ event }: { event: TrustEventData }) {
  const typeConfig = {
    approval: { icon: CheckCircleIcon, color: 'text-emerald-400', bg: 'bg-emerald-500/20' },
    rejection: { icon: XCircleIcon, color: 'text-red-400', bg: 'bg-red-500/20' },
    auto_approve: { icon: ClockIcon, color: 'text-blue-400', bg: 'bg-blue-500/20' },
    correction: { icon: ArrowTrendingUpIcon, color: 'text-amber-400', bg: 'bg-amber-500/20' },
  }

  const config = typeConfig[event.type]
  const Icon = config.icon

  return (
    <li className="flex items-center gap-4 p-4 hover:bg-surface-light/20 transition-colors">
      <div className={clsx('w-10 h-10 rounded-lg flex items-center justify-center', config.bg)}>
        <Icon className={clsx('w-5 h-5', config.color)} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-white">{event.description}</p>
        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-gray-500">{event.source}</span>
          <span className="text-gray-600">•</span>
          <span className="text-xs text-gray-500">
            {new Date(event.timestamp).toLocaleString()}
          </span>
        </div>
      </div>
      <div
        className={clsx(
          'text-sm font-medium',
          event.points >= 0 ? 'text-emerald-400' : 'text-red-400'
        )}
      >
        {event.points >= 0 ? '+' : ''}{event.points}
      </div>
    </li>
  )
}

export default TrustPage

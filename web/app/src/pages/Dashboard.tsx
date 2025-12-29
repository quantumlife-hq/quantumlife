import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { clsx } from 'clsx'
import {
  SparklesIcon,
  ChartBarIcon,
  ClockIcon,
  ArrowTrendingUpIcon,
  ChatBubbleLeftRightIcon,
  BoltIcon,
  ArrowRightIcon,
  CheckIcon,
  EllipsisHorizontalIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection, CardGrid, TwoColumnLayout } from '@/components/layout'
import {
  Card,
  CardHeader,
  CardContent,
  CardFooter,
  StatCard,
  Button,
  Badge,
  StatusBadge,
  Progress,
  ProgressRing,
  Avatar,
  Spinner,
  SkeletonCard,
  EmptyState,
} from '@/components/ui'
import { useAppStore, useCurrentHat } from '@/stores/app'
import type { Hat, Item } from '@/types'

export function DashboardPage() {
  const { identity, setCurrentHat } = useAppStore()
  const currentHat = useCurrentHat()

  // Fetch hats
  const { data: hats, isLoading: hatsLoading } = useQuery({
    queryKey: ['hats'],
    queryFn: api.hats.list,
  })

  // Fetch recent items
  const { data: recentItems, isLoading: itemsLoading } = useQuery({
    queryKey: ['items', 'recent'],
    queryFn: () => api.items.list({ limit: 5 }),
  })

  // Fetch connections stats
  const { data: connections } = useQuery({
    queryKey: ['connections'],
    queryFn: api.connections.list,
  })

  // Fetch trust capital
  const { data: trustCapital } = useQuery({
    queryKey: ['trust', 'capital'],
    queryFn: api.trust.getCapital,
  })

  // Calculate stats
  const totalConnections = connections?.length || 0
  const activeItems = recentItems?.length || 0
  const trustLevel = trustCapital?.level || 0

  return (
    <PageContent>
      {/* Welcome Header */}
      <PageSection>
        <Card className="bg-gradient-to-r from-primary-500/20 via-violet-500/20 to-fuchsia-500/20 border-primary-500/30">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-bold text-white">
                Welcome back, {identity?.name?.split(' ')[0] || 'User'}
              </h2>
              <p className="text-gray-400 mt-1">
                Here's what's happening with your digital life today
              </p>
            </div>
            <div className="hidden md:flex items-center gap-4">
              <QuickStatBadge
                icon={<BoltIcon className="w-4 h-4" />}
                label="Trust Level"
                value={`${trustLevel}%`}
                variant="primary"
              />
              <QuickStatBadge
                icon={<ChartBarIcon className="w-4 h-4" />}
                label="Connections"
                value={totalConnections.toString()}
                variant="info"
              />
            </div>
          </div>
        </Card>
      </PageSection>

      {/* Main Content */}
      <TwoColumnLayout
        sidebar={<Sidebar hats={hats || []} currentHat={currentHat} onSelectHat={setCurrentHat} isLoading={hatsLoading} />}
        sidebarWidth="md"
      >
        {/* Stats Row */}
        <CardGrid columns={3} className="mb-6">
          <StatCard
            label="Active Items"
            value={activeItems}
            icon={<ClockIcon className="w-6 h-6" />}
            trend={{ value: 12, isPositive: true }}
            description="Items pending action"
          />
          <StatCard
            label="Trust Capital"
            value={`${trustLevel}%`}
            icon={<ArrowTrendingUpIcon className="w-6 h-6" />}
            description="Progressive autonomy level"
          />
          <StatCard
            label="Connected Services"
            value={totalConnections}
            icon={<ChartBarIcon className="w-6 h-6" />}
            description="Active integrations"
          />
        </CardGrid>

        {/* Recent Activity */}
        <PageSection
          title="Recent Activity"
          description="Your latest items across all services"
          action={
            <Link to="/items">
              <Button variant="ghost" size="sm">
                View all
                <ArrowRightIcon className="w-4 h-4 ml-1" />
              </Button>
            </Link>
          }
        >
          {itemsLoading ? (
            <Card>
              <div className="space-y-4">
                {[1, 2, 3].map((i) => (
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
          ) : !recentItems?.length ? (
            <Card padding="lg">
              <EmptyState
                icon={<ClockIcon className="w-8 h-8" />}
                title="No recent items"
                description="Items from your connected services will appear here"
                action={
                  <Link to="/connections">
                    <Button>Connect a service</Button>
                  </Link>
                }
              />
            </Card>
          ) : (
            <Card padding="none">
              <ul className="divide-y divide-surface-border">
                {recentItems.map((item) => (
                  <ActivityItem key={item.id} item={item} />
                ))}
              </ul>
            </Card>
          )}
        </PageSection>

        {/* AI Suggestions */}
        <PageSection
          title="AI Suggestions"
          description="Proactive recommendations from your AI assistant"
          action={
            <Link to="/proactive">
              <Button variant="ghost" size="sm">
                Configure
                <ArrowRightIcon className="w-4 h-4 ml-1" />
              </Button>
            </Link>
          }
        >
          <CardGrid columns={2}>
            <SuggestionCard
              icon="📧"
              title="Email Summary"
              description="You have 12 unread emails. 3 require action."
              action="Review emails"
            />
            <SuggestionCard
              icon="📅"
              title="Calendar Optimization"
              description="Tomorrow has 2 back-to-back meetings. Consider adding breaks."
              action="View calendar"
            />
          </CardGrid>
        </PageSection>
      </TwoColumnLayout>
    </PageContent>
  )
}

// Quick Stat Badge
function QuickStatBadge({
  icon,
  label,
  value,
  variant,
}: {
  icon: React.ReactNode
  label: string
  value: string
  variant: 'primary' | 'info'
}) {
  const variants = {
    primary: 'bg-primary-500/20 text-primary-300 border-primary-500/30',
    info: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
  }

  return (
    <div className={clsx('flex items-center gap-2 px-3 py-2 rounded-lg border', variants[variant])}>
      {icon}
      <div>
        <div className="text-xs text-gray-400">{label}</div>
        <div className="font-semibold">{value}</div>
      </div>
    </div>
  )
}

// Sidebar Component
function Sidebar({
  hats,
  currentHat,
  onSelectHat,
  isLoading,
}: {
  hats: Hat[]
  currentHat: Hat | null
  onSelectHat: (hatId: string | null) => void
  isLoading: boolean
}) {
  return (
    <div className="space-y-6">
      {/* Hats Section */}
      <Card>
        <CardHeader title="Your Hats" subtitle="Switch context" />
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-12 bg-surface-lighter rounded-lg animate-pulse" />
              ))}
            </div>
          ) : !hats.length ? (
            <p className="text-sm text-gray-500">No hats configured yet</p>
          ) : (
            <div className="space-y-2">
              {hats.map((hat) => (
                <button
                  key={hat.id}
                  onClick={() => onSelectHat(currentHat?.id === hat.id ? null : hat.id)}
                  className={clsx(
                    'w-full flex items-center gap-3 p-3 rounded-lg transition-all',
                    currentHat?.id === hat.id
                      ? 'bg-primary-500/20 border border-primary-500/30'
                      : 'hover:bg-surface-light/30 border border-transparent'
                  )}
                >
                  <span className="text-xl">{hat.icon}</span>
                  <div className="flex-1 text-left">
                    <div className="font-medium text-white">{hat.name}</div>
                    <div className="text-xs text-gray-500">{hat.description}</div>
                  </div>
                  {currentHat?.id === hat.id && (
                    <CheckIcon className="w-5 h-5 text-primary-400" />
                  )}
                </button>
              ))}
            </div>
          )}
        </CardContent>
        <CardFooter>
          <Link to="/settings" className="text-sm text-primary-400 hover:text-primary-300">
            Manage hats
          </Link>
        </CardFooter>
      </Card>

      {/* Trust Progress */}
      <Card>
        <CardHeader title="Trust Progress" />
        <CardContent>
          <div className="flex justify-center mb-4">
            <ProgressRing value={65} size={100} variant="primary" label="Level 3" />
          </div>
          <div className="space-y-3">
            <TrustItem label="Manual approvals" count={23} />
            <TrustItem label="Auto-approved" count={156} />
            <TrustItem label="Trust earned" count="2.5K" />
          </div>
        </CardContent>
        <CardFooter>
          <Link to="/trust" className="text-sm text-primary-400 hover:text-primary-300">
            View trust details
          </Link>
        </CardFooter>
      </Card>
    </div>
  )
}

function TrustItem({ label, count }: { label: string; count: number | string }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-gray-400">{label}</span>
      <span className="font-medium text-white">{count}</span>
    </div>
  )
}

// Activity Item
function ActivityItem({ item }: { item: Item }) {
  const typeIcons: Record<string, string> = {
    email: '📧',
    calendar: '📅',
    task: '✓',
    note: '📝',
    message: '💬',
  }

  return (
    <li className="flex items-center gap-4 p-4 hover:bg-surface-light/20 transition-colors">
      <div className="w-10 h-10 rounded-lg bg-surface-lighter flex items-center justify-center text-lg">
        {typeIcons[item.type] || '📄'}
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="font-medium text-white truncate">{item.title}</h4>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-xs text-gray-500">{item.source}</span>
          <span className="text-xs text-gray-600">•</span>
          <span className="text-xs text-gray-500">{formatTimeAgo(item.timestamp)}</span>
        </div>
      </div>
      {item.priority && (
        <Badge
          variant={item.priority === 'high' ? 'error' : item.priority === 'medium' ? 'warning' : 'neutral'}
          size="sm"
        >
          {item.priority}
        </Badge>
      )}
    </li>
  )
}

// Suggestion Card
function SuggestionCard({
  icon,
  title,
  description,
  action,
}: {
  icon: string
  title: string
  description: string
  action: string
}) {
  return (
    <Card hover className="cursor-pointer">
      <div className="flex items-start gap-3">
        <span className="text-2xl">{icon}</span>
        <div className="flex-1">
          <h4 className="font-medium text-white">{title}</h4>
          <p className="text-sm text-gray-400 mt-1">{description}</p>
          <button className="text-sm text-primary-400 hover:text-primary-300 mt-3 flex items-center gap-1">
            {action}
            <ArrowRightIcon className="w-4 h-4" />
          </button>
        </div>
      </div>
    </Card>
  )
}

// Helper function
function formatTimeAgo(timestamp: string | undefined): string {
  if (!timestamp) return 'Unknown'
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

export default DashboardPage

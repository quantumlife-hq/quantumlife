import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { clsx } from 'clsx'
import {
  BanknotesIcon,
  FunnelIcon,
  MagnifyingGlassIcon,
  ChevronDownIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection, CardGrid } from '@/components/layout'
import {
  Card,
  CardHeader,
  CardContent,
  Button,
  SearchInput,
  Badge,
  Modal,
  ModalBody,
  Spinner,
  EmptyState,
} from '@/components/ui'

interface LedgerEntry {
  id: string
  timestamp: string
  action: string
  type: 'manual' | 'auto' | 'suggestion'
  status: 'completed' | 'pending' | 'rejected' | 'reverted'
  source?: string
  domain?: string
  target?: string
  details: Record<string, unknown>
  trustImpact: number
}

export function LedgerPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [selectedEntry, setSelectedEntry] = useState<LedgerEntry | null>(null)

  // Fetch ledger entries
  const { data: entries, isLoading, refetch } = useQuery({
    queryKey: ['ledger', 'entries', statusFilter, typeFilter],
    queryFn: () => api.ledger.getEntries({
      status: statusFilter !== 'all' ? statusFilter : undefined,
      type: typeFilter !== 'all' ? typeFilter : undefined,
    }),
  })

  // Filter entries by search
  const filteredEntries = entries?.filter((entry: LedgerEntry) =>
    entry.action.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (entry.source || entry.domain || '').toLowerCase().includes(searchQuery.toLowerCase())
  ) || []

  // Calculate stats
  const stats = {
    total: entries?.length || 0,
    completed: entries?.filter((e: LedgerEntry) => e.status === 'completed').length || 0,
    pending: entries?.filter((e: LedgerEntry) => e.status === 'pending').length || 0,
    reverted: entries?.filter((e: LedgerEntry) => e.status === 'reverted').length || 0,
  }

  return (
    <PageContent>
      {/* Header */}
      <PageSection>
        <Card className="bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-red-500/20 border-amber-500/30">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-amber-500/30 flex items-center justify-center">
                <BanknotesIcon className="w-6 h-6 text-amber-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Action Ledger</h2>
                <p className="text-gray-400">
                  Complete audit trail of all AI actions
                </p>
              </div>
            </div>
            <Button variant="secondary" size="sm" onClick={() => refetch()}>
              <ArrowPathIcon className="w-4 h-4 mr-2" />
              Refresh
            </Button>
          </div>
        </Card>
      </PageSection>

      {/* Stats */}
      <PageSection>
        <CardGrid columns={4}>
          <StatCard label="Total Actions" value={stats.total} color="neutral" />
          <StatCard label="Completed" value={stats.completed} color="success" />
          <StatCard label="Pending" value={stats.pending} color="warning" />
          <StatCard label="Reverted" value={stats.reverted} color="error" />
        </CardGrid>
      </PageSection>

      {/* Filters */}
      <PageSection>
        <div className="flex flex-wrap items-center gap-4">
          <SearchInput
            placeholder="Search actions..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-64"
          />

          <div className="flex items-center gap-2">
            <FunnelIcon className="w-4 h-4 text-gray-500" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="bg-surface-lighter border border-surface-border text-white text-sm rounded-lg px-3 py-2 focus:ring-2 focus:ring-primary-500/50"
            >
              <option value="all">All Statuses</option>
              <option value="completed">Completed</option>
              <option value="pending">Pending</option>
              <option value="rejected">Rejected</option>
              <option value="reverted">Reverted</option>
            </select>

            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="bg-surface-lighter border border-surface-border text-white text-sm rounded-lg px-3 py-2 focus:ring-2 focus:ring-primary-500/50"
            >
              <option value="all">All Types</option>
              <option value="manual">Manual</option>
              <option value="auto">Auto</option>
              <option value="suggestion">Suggestion</option>
            </select>
          </div>

          <div className="ml-auto text-sm text-gray-500">
            {filteredEntries.length} entries
          </div>
        </div>
      </PageSection>

      {/* Ledger Entries */}
      <PageSection>
        {isLoading ? (
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
        ) : filteredEntries.length === 0 ? (
          <Card padding="lg">
            <EmptyState
              icon={<BanknotesIcon className="w-8 h-8" />}
              title="No entries found"
              description={searchQuery ? "Try adjusting your search or filters" : "Actions will appear here as you use QuantumLife"}
              action={
                searchQuery && (
                  <Button variant="ghost" onClick={() => setSearchQuery('')}>
                    Clear search
                  </Button>
                )
              }
            />
          </Card>
        ) : (
          <Card padding="none">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-surface-border">
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Time</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Action</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Type</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Source</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Status</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-400">Trust</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-400">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEntries.map((entry: LedgerEntry) => (
                    <LedgerRow
                      key={entry.id}
                      entry={entry}
                      onClick={() => setSelectedEntry(entry)}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}
      </PageSection>

      {/* Entry Detail Modal */}
      <Modal
        isOpen={!!selectedEntry}
        onClose={() => setSelectedEntry(null)}
        title="Action Details"
        size="md"
      >
        {selectedEntry && (
          <ModalBody>
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <StatusIcon status={selectedEntry.status} />
                <div>
                  <h3 className="font-medium text-white">{selectedEntry.action}</h3>
                  <p className="text-sm text-gray-500">
                    {new Date(selectedEntry.timestamp).toLocaleString()}
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <InfoBlock label="Type" value={selectedEntry.type} />
                <InfoBlock label="Status" value={selectedEntry.status} />
                <InfoBlock label="Source" value={selectedEntry.source || selectedEntry.domain || 'N/A'} />
                <InfoBlock label="Target" value={selectedEntry.target || 'N/A'} />
              </div>

              <div className="p-4 rounded-lg bg-surface-light/20 border border-surface-border">
                <div className="text-sm text-gray-400 mb-2">Details</div>
                <pre className="text-sm text-gray-300 overflow-x-auto whitespace-pre-wrap">
                  {JSON.stringify(selectedEntry.details, null, 2)}
                </pre>
              </div>

              <div className="flex items-center justify-between p-4 rounded-lg bg-surface-light/20 border border-surface-border">
                <span className="text-sm text-gray-400">Trust Impact</span>
                <span
                  className={clsx(
                    'font-medium',
                    selectedEntry.trustImpact >= 0 ? 'text-emerald-400' : 'text-red-400'
                  )}
                >
                  {selectedEntry.trustImpact >= 0 ? '+' : ''}{selectedEntry.trustImpact} points
                </span>
              </div>
            </div>
          </ModalBody>
        )}
      </Modal>
    </PageContent>
  )
}

// Stat Card
function StatCard({
  label,
  value,
  color,
}: {
  label: string
  value: number
  color: 'neutral' | 'success' | 'warning' | 'error'
}) {
  const colorStyles = {
    neutral: 'border-surface-border',
    success: 'border-emerald-500/30 bg-emerald-500/5',
    warning: 'border-amber-500/30 bg-amber-500/5',
    error: 'border-red-500/30 bg-red-500/5',
  }

  return (
    <Card className={clsx('border', colorStyles[color])}>
      <div className="text-center">
        <div className="text-2xl font-bold text-white">{value.toLocaleString()}</div>
        <div className="text-sm text-gray-400 mt-1">{label}</div>
      </div>
    </Card>
  )
}

// Ledger Row
function LedgerRow({
  entry,
  onClick,
}: {
  entry: LedgerEntry
  onClick: () => void
}) {
  const statusConfig = {
    completed: { color: 'text-emerald-400', bg: 'bg-emerald-500/20', label: 'Completed' },
    pending: { color: 'text-amber-400', bg: 'bg-amber-500/20', label: 'Pending' },
    rejected: { color: 'text-red-400', bg: 'bg-red-500/20', label: 'Rejected' },
    reverted: { color: 'text-gray-400', bg: 'bg-gray-500/20', label: 'Reverted' },
  }

  const typeConfig = {
    manual: { label: 'Manual', variant: 'neutral' as const },
    auto: { label: 'Auto', variant: 'primary' as const },
    suggestion: { label: 'Suggestion', variant: 'info' as const },
  }

  const status = statusConfig[entry.status]
  const type = typeConfig[entry.type]

  return (
    <tr
      className="border-b border-surface-border hover:bg-surface-light/20 cursor-pointer transition-colors"
      onClick={onClick}
    >
      <td className="px-4 py-3 text-sm text-gray-400">
        {formatTimestamp(entry.timestamp)}
      </td>
      <td className="px-4 py-3">
        <div className="font-medium text-white truncate max-w-xs">{entry.action}</div>
      </td>
      <td className="px-4 py-3">
        <Badge variant={type.variant} size="sm">{type.label}</Badge>
      </td>
      <td className="px-4 py-3 text-sm text-gray-400">
        {entry.source || entry.domain || 'N/A'}
      </td>
      <td className="px-4 py-3">
        <span className={clsx('inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs', status.bg, status.color)}>
          <StatusDot status={entry.status} />
          {status.label}
        </span>
      </td>
      <td className="px-4 py-3">
        <span
          className={clsx(
            'text-sm font-medium',
            entry.trustImpact >= 0 ? 'text-emerald-400' : 'text-red-400'
          )}
        >
          {entry.trustImpact >= 0 ? '+' : ''}{entry.trustImpact}
        </span>
      </td>
      <td className="px-4 py-3 text-right">
        <ChevronDownIcon className="w-4 h-4 text-gray-500 inline rotate-[-90deg]" />
      </td>
    </tr>
  )
}

// Status Icon
function StatusIcon({ status }: { status: string }) {
  const icons = {
    completed: <CheckCircleIcon className="w-8 h-8 text-emerald-400" />,
    pending: <ClockIcon className="w-8 h-8 text-amber-400" />,
    rejected: <XCircleIcon className="w-8 h-8 text-red-400" />,
    reverted: <ArrowPathIcon className="w-8 h-8 text-gray-400" />,
  }

  return icons[status as keyof typeof icons] || icons.pending
}

// Status Dot
function StatusDot({ status }: { status: string }) {
  const colors = {
    completed: 'bg-emerald-400',
    pending: 'bg-amber-400',
    rejected: 'bg-red-400',
    reverted: 'bg-gray-400',
  }

  return <span className={clsx('w-1.5 h-1.5 rounded-full', colors[status as keyof typeof colors])} />
}

// Info Block
function InfoBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="p-3 rounded-lg bg-surface-light/20">
      <div className="text-xs text-gray-500 mb-1">{label}</div>
      <div className="text-sm text-white capitalize">{value}</div>
    </div>
  )
}

// Helper
function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()

  if (isToday) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

export default LedgerPage

import { useStats, useTrustOverall } from '../api/hooks';
import { cn } from '../lib/utils';

export function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useStats();
  const { data: trust, isLoading: trustLoading } = useTrustOverall();

  if (statsLoading || trustLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome back{stats?.identity ? `, ${stats.identity}` : ''}
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Items"
          value={stats?.total_items ?? 0}
          description="Total items tracked"
        />
        <StatCard
          title="Memories"
          value={stats?.total_memories ?? 0}
          description="Stored memories"
        />
        <StatCard
          title="Spaces"
          value={stats?.total_spaces ?? 0}
          description="Connected integrations"
        />
        <StatCard
          title="Trust Score"
          value={trust?.overall_score?.toFixed(1) ?? '—'}
          description={trust?.interpretation ?? 'Not available'}
          valueClassName={getTrustColor(trust?.overall_state)}
        />
      </div>

      {/* Trust Overview */}
      {trust && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold mb-4">Trust Overview</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Overall State</span>
              <span className={cn(
                "px-2 py-1 rounded-full text-xs font-medium",
                getStateBadgeColor(trust.overall_state)
              )}>
                {trust.overall_state}
              </span>
            </div>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Trust Score</span>
                <span>{trust.overall_score.toFixed(1)}%</span>
              </div>
              <div className="h-2 rounded-full bg-secondary">
                <div
                  className={cn("h-2 rounded-full transition-all", getTrustBarColor(trust.overall_score))}
                  style={{ width: `${trust.overall_score}%` }}
                />
              </div>
            </div>
            <p className="text-sm text-muted-foreground">{trust.interpretation}</p>
          </div>
        </div>
      )}

      {/* Agent Status */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="text-lg font-semibold mb-4">Agent Status</h2>
        <div className="flex items-center gap-2">
          <div className={cn(
            "h-3 w-3 rounded-full",
            stats?.agent_running ? "bg-green-500" : "bg-yellow-500"
          )} />
          <span className="text-sm">
            {stats?.agent_running ? 'Agent is running' : 'Agent is idle'}
          </span>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  description,
  valueClassName,
}: {
  title: string;
  value: string | number;
  description: string;
  valueClassName?: string;
}) {
  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="text-sm font-medium text-muted-foreground">{title}</div>
      <div className={cn("text-2xl font-bold mt-2", valueClassName)}>{value}</div>
      <p className="text-xs text-muted-foreground mt-1">{description}</p>
    </div>
  );
}

function getTrustColor(state?: string) {
  switch (state) {
    case 'verified': return 'text-green-600';
    case 'trusted': return 'text-blue-600';
    case 'learning': return 'text-yellow-600';
    case 'probation': return 'text-orange-600';
    case 'restricted': return 'text-red-600';
    default: return '';
  }
}

function getStateBadgeColor(state?: string) {
  switch (state) {
    case 'verified': return 'bg-green-100 text-green-800';
    case 'trusted': return 'bg-blue-100 text-blue-800';
    case 'learning': return 'bg-yellow-100 text-yellow-800';
    case 'probation': return 'bg-orange-100 text-orange-800';
    case 'restricted': return 'bg-red-100 text-red-800';
    default: return 'bg-gray-100 text-gray-800';
  }
}

function getTrustBarColor(score: number) {
  if (score >= 90) return 'bg-green-500';
  if (score >= 75) return 'bg-blue-500';
  if (score >= 50) return 'bg-yellow-500';
  if (score >= 30) return 'bg-orange-500';
  return 'bg-red-500';
}

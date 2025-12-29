import { useTrustScores, useTrustOverall } from '../api/hooks';
import { cn } from '../lib/utils';
import type { TrustScore, TrustState } from '../types';

export function Trust() {
  const { data: scoresData, isLoading: scoresLoading } = useTrustScores();
  const { data: overall, isLoading: overallLoading } = useTrustOverall();

  if (scoresLoading || overallLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  const scores = scoresData?.scores ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Trust Capital</h1>
        <p className="text-muted-foreground">
          Monitor and manage agent trust across domains
        </p>
      </div>

      {/* Overall Trust */}
      {overall && (
        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Overall Trust</h2>
            <span className={cn(
              "px-3 py-1 rounded-full text-sm font-medium",
              getStateBadgeColor(overall.overall_state)
            )}>
              {formatState(overall.overall_state)}
            </span>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <div className="text-4xl font-bold">{overall.overall_score.toFixed(1)}</div>
              <div className="text-sm text-muted-foreground">Trust Score</div>
            </div>
            <div>
              <div className="text-4xl font-bold">{overall.domain_count}</div>
              <div className="text-sm text-muted-foreground">Active Domains</div>
            </div>
            <div className="md:col-span-1">
              <p className="text-sm">{overall.interpretation}</p>
            </div>
          </div>

          <div className="mt-4">
            <div className="h-3 rounded-full bg-secondary">
              <div
                className={cn("h-3 rounded-full transition-all", getTrustBarColor(overall.overall_score))}
                style={{ width: `${overall.overall_score}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* Domain Scores */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Domain Trust Scores</h2>
        {scores.length === 0 ? (
          <div className="rounded-lg border bg-card p-6 text-center text-muted-foreground">
            No trust data available yet. Trust scores will appear as the agent performs actions.
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {scores.map((score) => (
              <DomainCard key={score.domain} score={score} />
            ))}
          </div>
        )}
      </div>

      {/* Trust State Legend */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="text-lg font-semibold mb-4">Trust States</h2>
        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-5">
          <StateInfo state="probation" description="New agent, suggestions only" />
          <StateInfo state="learning" description="Building trust, supervised mode" />
          <StateInfo state="trusted" description="Autonomous with undo window" />
          <StateInfo state="verified" description="Full autonomy" />
          <StateInfo state="restricted" description="Trust lost, recovery required" />
        </div>
      </div>
    </div>
  );
}

function DomainCard({ score }: { score: TrustScore }) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium capitalize">{score.domain}</h3>
        <span className={cn(
          "px-2 py-0.5 rounded-full text-xs font-medium",
          getStateBadgeColor(score.state)
        )}>
          {formatState(score.state)}
        </span>
      </div>

      <div className="space-y-3">
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted-foreground">Trust Score</span>
            <span className="font-medium">{score.value.toFixed(1)}%</span>
          </div>
          <div className="h-2 rounded-full bg-secondary">
            <div
              className={cn("h-2 rounded-full transition-all", getTrustBarColor(score.value))}
              style={{ width: `${score.value}%` }}
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2 text-xs">
          <FactorBar label="Accuracy" value={score.factors.accuracy} />
          <FactorBar label="Compliance" value={score.factors.compliance} />
          <FactorBar label="Calibration" value={score.factors.calibration} />
          <FactorBar label="Recency" value={score.factors.recency} />
          <FactorBar label="Reversals" value={score.factors.reversals} />
          <div className="text-muted-foreground">
            {score.action_count} actions
          </div>
        </div>
      </div>
    </div>
  );
}

function FactorBar({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <div className="flex justify-between mb-0.5">
        <span className="text-muted-foreground">{label}</span>
        <span>{value.toFixed(0)}</span>
      </div>
      <div className="h-1 rounded-full bg-secondary">
        <div
          className="h-1 rounded-full bg-primary/60"
          style={{ width: `${value}%` }}
        />
      </div>
    </div>
  );
}

function StateInfo({ state, description }: { state: TrustState; description: string }) {
  return (
    <div className="flex items-start gap-2">
      <span className={cn(
        "px-2 py-0.5 rounded-full text-xs font-medium shrink-0",
        getStateBadgeColor(state)
      )}>
        {formatState(state)}
      </span>
      <span className="text-xs text-muted-foreground">{description}</span>
    </div>
  );
}

function formatState(state: string) {
  return state.charAt(0).toUpperCase() + state.slice(1);
}

function getStateBadgeColor(state?: string) {
  switch (state) {
    case 'verified': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
    case 'trusted': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400';
    case 'learning': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
    case 'probation': return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400';
    case 'restricted': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
    default: return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400';
  }
}

function getTrustBarColor(score: number) {
  if (score >= 90) return 'bg-green-500';
  if (score >= 75) return 'bg-blue-500';
  if (score >= 50) return 'bg-yellow-500';
  if (score >= 30) return 'bg-orange-500';
  return 'bg-red-500';
}

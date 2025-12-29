import { useLedgerEntries, useLedgerSummary, useLedgerVerify } from '../api/hooks';
import { cn } from '../lib/utils';
import { CheckCircle, XCircle, Shield, Clock } from 'lucide-react';

export function Ledger() {
  const { data: entriesData, isLoading: entriesLoading } = useLedgerEntries(50, 0);
  const { data: summary, isLoading: summaryLoading } = useLedgerSummary();
  const { data: verification, isLoading: verifyLoading } = useLedgerVerify();

  const isLoading = entriesLoading || summaryLoading || verifyLoading;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  const entries = entriesData?.entries ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Audit Trail</h1>
        <p className="text-muted-foreground">
          Cryptographic hash-chained ledger of all agent actions
        </p>
      </div>

      {/* Chain Verification Status */}
      <div className={cn(
        "rounded-lg border p-4 flex items-center gap-4",
        verification?.valid ? "bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800" : "bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800"
      )}>
        {verification?.valid ? (
          <CheckCircle className="h-8 w-8 text-green-600" />
        ) : (
          <XCircle className="h-8 w-8 text-red-600" />
        )}
        <div>
          <div className="font-medium">
            {verification?.valid ? 'Chain Integrity Verified' : 'Chain Integrity Failed'}
          </div>
          <div className="text-sm text-muted-foreground">
            {verification?.entries_checked} entries verified
            {verification?.error && ` - Error: ${verification.error}`}
          </div>
        </div>
      </div>

      {/* Summary Stats */}
      {summary && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <StatCard
            icon={<Shield className="h-4 w-4" />}
            title="Total Entries"
            value={summary.total_entries}
          />
          <StatCard
            icon={<Clock className="h-4 w-4" />}
            title="First Entry"
            value={formatDate(summary.first_entry)}
          />
          <StatCard
            icon={<Clock className="h-4 w-4" />}
            title="Last Entry"
            value={formatDate(summary.last_entry)}
          />
          <StatCard
            icon={<CheckCircle className="h-4 w-4" />}
            title="Chain Valid"
            value={summary.chain_valid ? 'Yes' : 'No'}
          />
        </div>
      )}

      {/* Actions Breakdown */}
      {summary?.actions_by_type && Object.keys(summary.actions_by_type).length > 0 && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold mb-4">Actions by Type</h2>
          <div className="flex flex-wrap gap-2">
            {Object.entries(summary.actions_by_type).map(([action, count]) => (
              <div key={action} className="px-3 py-1.5 rounded-lg bg-secondary text-sm">
                <span className="font-medium">{action}</span>
                <span className="ml-2 text-muted-foreground">{count}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Ledger Entries */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Recent Entries</h2>
        {entries.length === 0 ? (
          <div className="rounded-lg border bg-card p-6 text-center text-muted-foreground">
            No ledger entries yet. Entries will appear as the agent performs actions.
          </div>
        ) : (
          <div className="rounded-lg border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="px-4 py-3 text-left font-medium">#</th>
                    <th className="px-4 py-3 text-left font-medium">Timestamp</th>
                    <th className="px-4 py-3 text-left font-medium">Action</th>
                    <th className="px-4 py-3 text-left font-medium">Actor</th>
                    <th className="px-4 py-3 text-left font-medium">Entity</th>
                    <th className="px-4 py-3 text-left font-medium">Hash</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {entries.map((entry) => (
                    <tr key={entry.id} className="hover:bg-muted/50">
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                        {entry.sequence}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {formatDateTime(entry.timestamp)}
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-0.5 rounded bg-secondary text-xs font-medium">
                          {entry.action}
                        </span>
                      </td>
                      <td className="px-4 py-3 capitalize">{entry.actor}</td>
                      <td className="px-4 py-3">
                        <span className="text-muted-foreground">{entry.entity_type}:</span>{' '}
                        <span className="font-mono text-xs">{truncate(entry.entity_id, 12)}</span>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                        {truncate(entry.hash, 16)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({
  icon,
  title,
  value,
}: {
  icon: React.ReactNode;
  title: string;
  value: string | number;
}) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-center gap-2 text-muted-foreground mb-2">
        {icon}
        <span className="text-sm">{title}</span>
      </div>
      <div className="text-xl font-semibold">{value}</div>
    </div>
  );
}

function formatDate(dateStr: string) {
  if (!dateStr) return '—';
  return new Date(dateStr).toLocaleDateString();
}

function formatDateTime(dateStr: string) {
  if (!dateStr) return '—';
  const date = new Date(dateStr);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function truncate(str: string, len: number) {
  if (!str) return '—';
  if (str.length <= len) return str;
  return str.slice(0, len) + '...';
}

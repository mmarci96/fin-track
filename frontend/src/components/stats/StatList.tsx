import type { Bucket } from '@/lib/statistics';
import { displayMoney } from '@/lib/format';

/**
 * A ranked list of spending buckets with a proportional bar per row. Dependency
 * free on purpose — any new breakdown just passes a different Bucket[].
 */
export function StatList({
  buckets,
  currency,
  emptyMessage = 'Nothing to show yet.',
}: {
  buckets: Bucket[];
  currency: string;
  emptyMessage?: string;
}) {
  if (buckets.length === 0) {
    return <p className="py-8 text-center text-sm text-muted-foreground">{emptyMessage}</p>;
  }

  const max = buckets[0].total || 1; // buckets are sorted desc, so [0] is the largest

  return (
    <ul className="space-y-3">
      {buckets.map((b) => (
        <li key={b.key} className="space-y-1">
          <div className="flex items-baseline justify-between gap-2 text-sm">
            <span className="truncate font-medium">{b.label}</span>
            <span className="shrink-0 tabular-nums text-muted-foreground">
              {displayMoney(b.total, currency)}
            </span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary"
              style={{ width: `${Math.max(2, (b.total / max) * 100)}%` }}
            />
          </div>
        </li>
      ))}
    </ul>
  );
}

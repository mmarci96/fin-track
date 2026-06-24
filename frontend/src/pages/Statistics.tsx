import { useMemo, useState } from 'react';
import { useReceipts } from '@/api/receipts';
import { Card, CardContent } from '@/components/ui/card';
import { CenteredSpinner } from '@/components/ui/spinner';
import { WarningBanner } from '@/components/WarningBanner';
import { StatList } from '@/components/stats/StatList';
import { displayMoney } from '@/lib/format';
import { availableCurrencies, computeStats } from '@/lib/statistics';
import { cn } from '@/lib/utils';

type Dimension = 'stores' | 'products' | 'categories';

const DIMENSIONS: { id: Dimension; label: string }[] = [
  { id: 'stores', label: 'Stores' },
  { id: 'products', label: 'Products' },
  { id: 'categories', label: 'Categories' },
];

export function Statistics() {
  const { data: receipts, isLoading, isError, error } = useReceipts();
  const [currency, setCurrency] = useState<string | null>(null);
  const [dimension, setDimension] = useState<Dimension>('stores');

  const currencies = useMemo(
    () => availableCurrencies(receipts ?? []),
    [receipts],
  );
  // Default to the most-used currency until the user picks one.
  const activeCurrency = currency ?? currencies[0] ?? 'HUF';
  const stats = useMemo(
    () => computeStats(receipts ?? [], activeCurrency),
    [receipts, activeCurrency],
  );

  if (isLoading) return <CenteredSpinner />;
  if (isError)
    return (
      <div className="p-4">
        <WarningBanner
          tone="destructive"
          title="Could not load statistics"
          messages={[(error as Error).message]}
        />
      </div>
    );

  if (!receipts || receipts.length === 0)
    return (
      <div className="space-y-4 p-4">
        <h1 className="text-xl font-semibold">Statistics</h1>
        <Card>
          <CardContent className="py-12 text-center text-sm text-muted-foreground">
            No spending data yet. Add or scan a receipt to see your stats.
          </CardContent>
        </Card>
      </div>
    );

  return (
    <div className="space-y-4 p-4">
      <header className="flex items-center justify-between gap-2">
        <h1 className="text-xl font-semibold">Statistics</h1>
        {currencies.length > 1 && (
          <select
            aria-label="Currency"
            className="h-9 rounded-md border border-input bg-background px-2 text-sm"
            value={activeCurrency}
            onChange={(e) => setCurrency(e.target.value)}
          >
            {currencies.map((code) => (
              <option key={code} value={code}>
                {code}
              </option>
            ))}
          </select>
        )}
      </header>

      <Card>
        <CardContent className="space-y-1">
          <p className="text-sm text-muted-foreground">Total spend</p>
          <p className="text-3xl font-semibold tabular-nums">
            {displayMoney(stats.totalSpend, activeCurrency)}
          </p>
          <p className="text-sm text-muted-foreground">
            across {stats.receiptCount} receipt
            {stats.receiptCount === 1 ? '' : 's'}
          </p>
        </CardContent>
      </Card>

      <div className="flex rounded-lg border border-border p-1">
        {DIMENSIONS.map((d) => (
          <button
            key={d.id}
            type="button"
            onClick={() => setDimension(d.id)}
            className={cn(
              'flex-1 rounded-md py-2 text-sm font-medium transition-colors',
              dimension === d.id
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {d.label}
          </button>
        ))}
      </div>

      <Card>
        <CardContent>
          {dimension === 'stores' && (
            <StatList buckets={stats.byStore} currency={activeCurrency} />
          )}
          {dimension === 'products' && (
            <StatList buckets={stats.byProduct} currency={activeCurrency} />
          )}
          {dimension === 'categories' && (
            <StatList
              buckets={stats.byCategory}
              currency={activeCurrency}
              emptyMessage="No category data yet."
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

import type { Receipt } from '@/api/receipts';

// Spending aggregations for the Statistics page. Everything here is pure and
// computed from the receipts already in the cache, so it is trivially testable
// and adding a new breakdown (e.g. "by month") is a one-line groupBy call.

export interface Bucket {
  key: string;
  label: string;
  total: number; // integer minor units, in the stats' currency
  count: number;
}

export interface Stats {
  currency: string;
  totalSpend: number;
  receiptCount: number;
  byStore: Bucket[];
  byProduct: Bucket[];
  byCategory: Bucket[]; // empty until receipts carry category data
}

/**
 * Generic bucketed sum. `keyFn` returns null to skip an item. Buckets are
 * returned sorted by total descending.
 */
function groupBy<T>(
  items: T[],
  keyFn: (item: T) => string | null,
  labelFn: (item: T) => string,
  amountFn: (item: T) => number,
): Bucket[] {
  const map = new Map<string, Bucket>();
  for (const item of items) {
    const key = keyFn(item);
    if (key === null) continue;
    const existing = map.get(key);
    if (existing) {
      existing.total += amountFn(item);
      existing.count += 1;
    } else {
      map.set(key, {
        key,
        label: labelFn(item),
        total: amountFn(item),
        count: 1,
      });
    }
  }
  return [...map.values()].sort((a, b) => b.total - a.total);
}

/** Currencies present in the data, most-used first (good default selection). */
export function availableCurrencies(receipts: Receipt[]): string[] {
  const counts = new Map<string, number>();
  for (const r of receipts) {
    counts.set(r.currency, (counts.get(r.currency) ?? 0) + 1);
  }
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([code]) => code);
}

/**
 * Aggregate spending for a single currency. Mixing currencies would be
 * meaningless (their minor units differ), so we filter to one first.
 *
 * Note: a product belonging to N categories contributes its full price to each
 * of them, so category totals can exceed the overall spend — that's the
 * intended "how much went toward groceries vs household" reading.
 */
export function computeStats(receipts: Receipt[], currency: string): Stats {
  const scoped = receipts.filter((r) => r.currency === currency);
  const products = scoped.flatMap((r) => r.products);
  const productCategoryPairs = products.flatMap((p) =>
    p.categories.map((c) => ({ category: c, price: p.price })),
  );

  return {
    currency,
    totalSpend: scoped.reduce((sum, r) => sum + r.total, 0),
    receiptCount: scoped.length,
    byStore: groupBy(
      scoped,
      (r) => r.merchant || 'Unknown',
      (r) => r.merchant || 'Unknown',
      (r) => r.total,
    ),
    byProduct: groupBy(
      products.filter((p) => p.name.trim() !== ''),
      (p) => p.name.trim().toLowerCase(),
      (p) => p.name.trim(),
      (p) => p.price,
    ),
    byCategory: groupBy(
      productCategoryPairs,
      (pair) => String(pair.category.id),
      (pair) => pair.category.name,
      (pair) => pair.price,
    ),
  };
}

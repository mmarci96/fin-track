// Prices and totals are integers in minor units (e.g. cents / fillér).
// These helpers are the single place that converts to/from that representation.

const MINOR_PER_MAJOR = 100;

/** Format an integer minor-unit amount for display, e.g. 1299 -> "12.99". */
export function formatMoney(minor: number): string {
  return (minor / MINOR_PER_MAJOR).toFixed(2);
}

/** Parse a user-typed major-unit string back into integer minor units. */
export function parseMoney(input: string): number {
  const normalized = input.replace(/\s/g, '').replace(',', '.');
  const value = Number(normalized);
  if (!Number.isFinite(value)) return 0;
  return Math.round(value * MINOR_PER_MAJOR);
}

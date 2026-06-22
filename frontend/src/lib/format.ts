// Prices and totals are stored as integers in a currency's minor unit. The
// number of decimals depends on the currency: HUF has none (the integer is
// whole forints, e.g. 5300 -> "5300 Ft"), EUR has two (e.g. 399 -> "3.99 €").
// These helpers are the single place that converts to/from that representation.

interface CurrencyInfo {
  decimals: number;
  symbol: string;
}

const CURRENCIES: Record<string, CurrencyInfo> = {
  HUF: { decimals: 0, symbol: 'Ft' },
  EUR: { decimals: 2, symbol: '€' },
};

const DEFAULT_CURRENCY = 'HUF';

function info(currency: string): CurrencyInfo {
  return CURRENCIES[currency] ?? CURRENCIES[DEFAULT_CURRENCY];
}

/** The currency codes we support, for populating selectors. */
export const CURRENCY_CODES = Object.keys(CURRENCIES);

/**
 * Format an integer minor-unit amount as a plain number string for the given
 * currency, e.g. (5300, "HUF") -> "5300", (399, "EUR") -> "3.99". Used for
 * form input values, so it carries no currency symbol.
 */
export function formatMoney(minor: number, currency = DEFAULT_CURRENCY): string {
  const { decimals } = info(currency);
  return (minor / 10 ** decimals).toFixed(decimals);
}

/** Like formatMoney but with the currency symbol, for read-only display. */
export function displayMoney(minor: number, currency = DEFAULT_CURRENCY): string {
  return `${formatMoney(minor, currency)} ${info(currency).symbol}`;
}

/** Parse a user-typed major-unit string back into integer minor units. */
export function parseMoney(input: string, currency = DEFAULT_CURRENCY): number {
  const { decimals } = info(currency);
  const normalized = input.replace(/\s/g, '').replace(',', '.');
  const value = Number(normalized);
  if (!Number.isFinite(value)) return 0;
  return Math.round(value * 10 ** decimals);
}

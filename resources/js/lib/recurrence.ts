// The recurrence rules of docs/specs/recurrence.md in TypeScript, for the previews of the forms and the
// mobile app. It passes the same vectors as the Go engine (testdata/recurrence-vectors.json). Dates are
// ISO calendar dates, handled as integers: no time zone, no daylight saving. The year is a number all
// the way: a date past MAX_YEAR is never turned into a string and compared, and one walk is capped.

export type Frequency = "daily" | "weekly" | "monthly" | "yearly";

export interface Rule {
  frequency: Frequency;
  interval: number;
  start_date: string;
  end_date: string | null;
  max_count: number | null;
}

/** The engine stops after this year: ISO dates stay four digits. */
export const MAX_YEAR = 9999;

/** One call looks at no more than this many occurrences, whatever the window. */
export const MAX_WALK = 10000;

interface Ymd {
  y: number;
  m: number;
  d: number;
}

function parse(date: string): Ymd | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return null;
  const [y, m, d] = [Number(match[1]), Number(match[2]), Number(match[3])];
  if (y < 1 || m < 1 || m > 12 || d < 1 || d > lastDay(y, m)) return null;
  return { y, m, d };
}

function pad(n: number, width: number): string {
  return String(n).padStart(width, "0");
}

function format({ y, m, d }: Ymd): string {
  return `${pad(y, 4)}-${pad(m, 2)}-${pad(d, 2)}`;
}

/** A UTC midnight in milliseconds; setUTCFullYear avoids the two-digit-year rule of Date.UTC (year 50 is not 1950). */
function utc(y: number, month0: number, d: number): number {
  const t = new Date(0);
  t.setUTCFullYear(y, month0, d);
  return t.getTime();
}

export function lastDay(year: number, month: number): number {
  return new Date(utc(year, month, 0)).getUTCDate();
}

function monthsAdded(p: Ymd, n: number): Ymd {
  const total = p.y * 12 + (p.m - 1) + n;
  const y = Math.floor(total / 12);
  const m = (total % 12) + 1;
  return { y, m, d: Math.min(p.d, lastDay(y, m)) };
}

function daysAdded(p: Ymd, n: number): Ymd {
  const t = new Date(utc(p.y, p.m - 1, p.d + n));
  return { y: t.getUTCFullYear(), m: t.getUTCMonth() + 1, d: t.getUTCDate() };
}

function checked(p: Ymd): string {
  if (p.y < 1 || p.y > MAX_YEAR) throw new RangeError(`year ${p.y} is outside 1 to ${MAX_YEAR}`);
  return format(p);
}

/** Moves a date by `n` months and clamps the day to the last day of the target month (Jan 31 + 1 is Feb 28). */
export function addMonths(date: string, n: number): string {
  const p = parse(date);
  if (!p) throw new RangeError(`bad date ${date}`);
  return checked(monthsAdded(p, n));
}

export function addDays(date: string, n: number): string {
  const p = parse(date);
  if (!p) throw new RangeError(`bad date ${date}`);
  return checked(daysAdded(p, n));
}

/** The k-th occurrence from the start date and k alone (a clamped day never drifts); null past MAX_YEAR. */
function nthYmd(rule: Rule, start: Ymd, k: number): Ymd | null {
  const n = k * rule.interval;
  let p: Ymd;
  switch (rule.frequency) {
    case "daily":
      p = daysAdded(start, n);
      break;
    case "weekly":
      p = daysAdded(start, 7 * n);
      break;
    case "monthly":
      p = monthsAdded(start, n);
      break;
    case "yearly":
      p = monthsAdded(start, 12 * n);
      break;
  }
  return p.y > MAX_YEAR ? null : p;
}

export function nth(rule: Rule, k: number): string {
  const start = parse(rule.start_date);
  if (!start) throw new RangeError(`bad date ${rule.start_date}`);
  const p = nthYmd(rule, start, k);
  if (p === null) throw new RangeError(`the ${k}-th occurrence is past the year ${MAX_YEAR}`);
  return format(p);
}

/** A k that is never later than the first occurrence on or after `from`, so a far window does not walk from 0. */
function firstIndex(rule: Rule, start: Ymd, from: Ymd): number {
  const days = Math.floor((utc(from.y, from.m - 1, from.d) - utc(start.y, start.m - 1, start.d)) / 86_400_000);
  if (days <= 0) return 0;
  let units: number;
  switch (rule.frequency) {
    case "daily":
      units = Math.floor(days / rule.interval);
      break;
    case "weekly":
      units = Math.floor(days / (7 * rule.interval));
      break;
    case "monthly":
      units = Math.floor(((from.y - start.y) * 12 + (from.m - start.m)) / rule.interval);
      break;
    case "yearly":
      units = Math.floor((from.y - start.y) / rule.interval);
      break;
  }
  return Math.max(0, units - 1);
}

/**
 * The occurrences with from <= date <= to, ascending, honoring the end date and the count (counted from the
 * start). At most MAX_WALK occurrences are looked at, so a hostile rule cannot hold the browser.
 */
export function occurrences(rule: Rule, from: string, to: string, limit = 0): string[] {
  const start = parse(rule.start_date);
  const lo = parse(from);
  const hi = parse(to);
  const end = rule.end_date === null ? null : parse(rule.end_date);
  if (!start || !lo || !hi || (rule.end_date !== null && !end)) return [];
  if (!Number.isInteger(rule.interval) || rule.interval < 1 || to < from) return [];
  const out: string[] = [];
  const first = firstIndex(rule, start, lo);
  for (let k = first; k < first + MAX_WALK; k++) {
    if (rule.max_count !== null && k >= rule.max_count) break;
    const p = nthYmd(rule, start, k);
    if (p === null) break;
    const date = format(p);
    if (date > to || (rule.end_date !== null && date > rule.end_date)) break;
    if (date < from) continue;
    out.push(date);
    if (limit > 0 && out.length >= limit) break;
  }
  return out;
}

/** Splits `total` cents into `parts` amounts that add up to it: the first total % parts parts get one extra cent. */
export function splitInstallments(total: number, parts: number): number[] {
  const base = Math.floor(total / parts);
  const extra = total % parts;
  return Array.from({ length: parts }, (_, i) => (i < extra ? base + 1 : base));
}

/** The dates of the parts: the first date plus i months, clamped to month ends. */
export function installmentDates(first: string, parts: number): string[] {
  return Array.from({ length: parts }, (_, i) => addMonths(first, i));
}

/** Reads "12.50" or "12,5" as cents for a preview; anything the server would read differently gives null. */
export function previewCents(text: string): number | null {
  const match = /^(\d{1,13})(?:[.,](\d{1,2}))?$/.exec(text.trim());
  if (!match) return null;
  const cents = Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
  return Number.isSafeInteger(cents) ? cents : null;
}

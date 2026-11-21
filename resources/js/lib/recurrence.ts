// The recurrence rules of docs/specs/recurrence.md in TypeScript, for the previews of the forms and the
// mobile app. It passes the same vectors as the Go engine (testdata/recurrence-vectors.json). Dates are
// ISO calendar dates, handled as integers: no Date object, no time zone, no daylight saving.

export type Frequency = "daily" | "weekly" | "monthly" | "yearly";

export interface Rule {
  frequency: Frequency;
  interval: number;
  start_date: string;
  end_date: string | null;
  max_count: number | null;
}

const MAX_YEAR = 9999;

interface Ymd {
  y: number;
  m: number;
  d: number;
}

function parse(date: string): Ymd | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return null;
  const [y, m, d] = [Number(match[1]), Number(match[2]), Number(match[3])];
  if (m < 1 || m > 12 || d < 1 || d > lastDay(y, m)) return null;
  return { y, m, d };
}

function pad(n: number, width: number): string {
  return String(n).padStart(width, "0");
}

function format({ y, m, d }: Ymd): string {
  return `${pad(y, 4)}-${pad(m, 2)}-${pad(d, 2)}`;
}

export function lastDay(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

/** Moves a date by `n` months and clamps the day to the last day of the target month (Jan 31 + 1 is Feb 28). */
export function addMonths(date: string, n: number): string {
  const p = parse(date);
  if (!p) throw new RangeError(`bad date ${date}`);
  const total = p.y * 12 + (p.m - 1) + n;
  const y = Math.floor(total / 12);
  const m = (total % 12) + 1;
  return format({ y, m, d: Math.min(p.d, lastDay(y, m)) });
}

export function addDays(date: string, n: number): string {
  const p = parse(date);
  if (!p) throw new RangeError(`bad date ${date}`);
  const t = new Date(Date.UTC(p.y, p.m - 1, p.d + n));
  return format({ y: t.getUTCFullYear(), m: t.getUTCMonth() + 1, d: t.getUTCDate() });
}

/** The date of the k-th occurrence, from the start date and k alone, so a clamped day never drifts. */
export function nth(rule: Rule, k: number): string {
  const n = k * rule.interval;
  switch (rule.frequency) {
    case "daily":
      return addDays(rule.start_date, n);
    case "weekly":
      return addDays(rule.start_date, 7 * n);
    case "monthly":
      return addMonths(rule.start_date, n);
    case "yearly":
      return addMonths(rule.start_date, 12 * n);
  }
}

/** The occurrences with from <= date <= to, ascending, honoring the end date and the count (counted from the start). */
export function occurrences(rule: Rule, from: string, to: string, limit = 0): string[] {
  if (!parse(rule.start_date) || !parse(from) || !parse(to) || rule.interval < 1 || to < from) return [];
  if (rule.end_date !== null && !parse(rule.end_date)) return [];
  const out: string[] = [];
  for (let k = 0; ; k++) {
    if (rule.max_count !== null && k >= rule.max_count) break;
    const date = nth(rule, k);
    if (Number(date.slice(0, 4)) > MAX_YEAR || date > to || (rule.end_date !== null && date > rule.end_date)) break;
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

import type { Database } from "../db/database";
import { addDays, addMonths, daysBetween, isIsoDate, monthEnd } from "./dates";
import { LedgerError } from "./ledger";
import { MAX_CENTS } from "./money";

export type PeriodPreset = "this_month" | "last_month" | "last_3_months" | "last_12_months" | "this_year" | "custom";

export interface Period {
  preset: PeriodPreset;
  from: string;
  to: string;
}

/** A period is at most ten years, so a series stays small. */
const MAX_PERIOD_DAYS = 3660;

function badPeriod(): LedgerError {
  return new LedgerError(
    "period",
    "choose a valid period: dates as YYYY-MM-DD, the start not after the end, at most ten years",
  );
}

/**
 * Resolves a preset against today (the workspace-local ISO date). Month
 * presets are whole calendar months: last_3_months and last_12_months are the
 * current month and the ones before it. An empty preset is this_month.
 */
export function resolvePeriod(preset: string, from: string, to: string, today: string): Period {
  if (!isIsoDate(today)) {
    throw badPeriod();
  }
  const monthStart = `${today.slice(0, 7)}-01`;
  let result: Period;

  switch (preset) {
    case "":
    case "this_month":
      result = { preset: "this_month", from: monthStart, to: monthEnd(today) };
      break;
    case "last_month": {
      const start = addMonths(monthStart, -1);
      result = { preset: "last_month", from: start, to: monthEnd(start) };
      break;
    }
    case "last_3_months":
      result = { preset: "last_3_months", from: addMonths(monthStart, -2), to: monthEnd(today) };
      break;
    case "last_12_months":
      result = { preset: "last_12_months", from: addMonths(monthStart, -11), to: monthEnd(today) };
      break;
    case "this_year":
      result = { preset: "this_year", from: `${today.slice(0, 4)}-01-01`, to: `${today.slice(0, 4)}-12-31` };
      break;
    case "custom":
      if (!isIsoDate(from) || !isIsoDate(to)) {
        throw badPeriod();
      }
      result = { preset: "custom", from, to };
      break;
    default:
      throw badPeriod();
  }

  if (result.to < result.from || daysBetween(result.from, result.to) > MAX_PERIOD_DAYS) {
    throw badPeriod();
  }
  return result;
}

/** The "YYYY-MM" of every month the period touches, oldest first. */
export function periodMonths(p: Period): string[] {
  const out: string[] = [];
  for (let m = `${p.from.slice(0, 7)}-01`; m <= p.to; m = addMonths(m, 1)) {
    out.push(m.slice(0, 7));
  }
  return out;
}

function wholeMonths(p: Period): number {
  if (!p.from.endsWith("-01") || p.to !== monthEnd(p.to)) {
    return 0;
  }
  return (
    (Number(p.to.slice(0, 4)) - Number(p.from.slice(0, 4))) * 12 +
    Number(p.to.slice(5, 7)) -
    Number(p.from.slice(5, 7)) +
    1
  );
}

/** The period of the same length right before p (whole months compare with the same number of months). */
export function previousPeriod(p: Period): Period {
  const n = wholeMonths(p);
  if (n > 0) {
    return { preset: "custom", from: addMonths(p.from, -n), to: monthEnd(addMonths(p.from, -1)) };
  }
  const len = daysBetween(p.from, p.to);
  const end = addDays(p.from, -1);
  return { preset: "custom", from: addDays(end, -(len - 1)), to: end };
}

export interface Sum {
  income_cents: number;
  expense_cents: number;
  net_cents: number;
}

export interface MonthFlow extends Sum {
  month: string;
}

export interface CashFlowReport {
  period: Period;
  months: MonthFlow[];
  totals: Sum;
}

function clamp(n: number): number {
  return Math.max(-MAX_CENTS, Math.min(MAX_CENTS, n));
}

function mul(amount: number, n: number): number {
  if (amount <= 0 || n <= 0) {
    return 0;
  }
  return amount > Math.floor(MAX_CENTS / n) ? MAX_CENTS : amount * n;
}

function emptySum(): Sum {
  return { income_cents: 0, expense_cents: 0, net_cents: 0 };
}

function put(sum: Sum, kind: string, cents: number): void {
  if (kind === "income") {
    sum.income_cents = clamp(sum.income_cents + cents);
  } else {
    sum.expense_cents = clamp(sum.expense_cents + cents);
  }
  sum.net_cents = clamp(sum.income_cents - sum.expense_cents);
}

/** Income against expense per calendar month; transfer legs are never counted. */
export async function cashFlow(db: Database, period: Period): Promise<CashFlowReport> {
  const months = periodMonths(period).map((month): MonthFlow => ({ month, ...emptySum() }));
  const index = new Map(months.map((m, i) => [m.month, i]));
  const totals = emptySum();

  const groups = await db.all<{ occurred_on: string; kind: string; amount_cents: number; n: number }>(
    `SELECT occurred_on, kind, amount_cents, COUNT(*) AS n FROM transactions
		 WHERE deleted_at IS NULL AND transfer_uid IS NULL AND occurred_on BETWEEN ? AND ?
		 GROUP BY occurred_on, kind, amount_cents`,
    [period.from, period.to],
  );
  for (const g of groups) {
    const cents = mul(g.amount_cents, g.n);
    const i = index.get(g.occurred_on.slice(0, 7));
    const month = i === undefined ? undefined : months[i];
    if (month !== undefined) {
      put(month, g.kind, cents);
    }
    put(totals, g.kind, cents);
  }
  return { period, months, totals };
}

export interface CategoryRow {
  uid: string;
  name: string;
  uncategorized: boolean;
  cents: number;
  own_cents: number;
  /** Share of the report total in basis points (1/100 of a percent), rounded half up. */
  share_bps: number;
  children: CategoryRow[];
}

export interface CategoryReport {
  period: Period;
  kind: "income" | "expense";
  total_cents: number;
  rows: CategoryRow[];
}

/** part / whole in basis points, half up, integer arithmetic only; 0 when either is not positive. */
export function bps(part: number, whole: number): number {
  if (whole <= 0 || part <= 0) {
    return 0;
  }
  const n = BigInt(part) * 10000n * 2n + BigInt(whole);
  return Number(n / (BigInt(whole) * 2n));
}

function sortRows(rows: CategoryRow[]): void {
  rows.sort((a, b) => {
    if (a.cents !== b.cents) {
      return a.cents > b.cents ? -1 : 1;
    }
    const ka = `${a.name}\u0000${a.uid}`;
    const kb = `${b.name}\u0000${b.uid}`;
    return ka < kb ? -1 : ka > kb ? 1 : 0;
  });
}

/** Totals the transactions of a kind by category, parents rolled up with their children beneath. */
export async function byCategory(db: Database, period: Period, kind: "income" | "expense"): Promise<CategoryReport> {
  const groups = await db.all<{ category_uid: string | null; amount_cents: number; n: number }>(
    `SELECT category_uid, amount_cents, COUNT(*) AS n FROM transactions
		 WHERE deleted_at IS NULL AND transfer_uid IS NULL AND kind = ? AND occurred_on BETWEEN ? AND ?
		 GROUP BY category_uid, amount_cents`,
    [kind, period.from, period.to],
  );
  // A live transaction may still name a deleted category, so trashed ones are read too.
  const cats = new Map(
    (
      await db.all<{ uid: string; name: string; parent_uid: string | null }>(
        "SELECT uid, name, parent_uid FROM categories",
      )
    ).map((c) => [c.uid, c]),
  );

  const out: CategoryReport = { period, kind, total_cents: 0, rows: [] };
  const own = new Map<string, number>();
  let uncategorized = 0;

  for (const g of groups) {
    const cents = mul(g.amount_cents, g.n);
    out.total_cents = clamp(out.total_cents + cents);
    if (g.category_uid === null) {
      uncategorized = clamp(uncategorized + cents);
    } else {
      own.set(g.category_uid, clamp((own.get(g.category_uid) ?? 0) + cents));
    }
  }

  const rows = new Map<string, CategoryRow>();
  const top: string[] = [];
  for (const [uid, cents] of own) {
    const c = cats.get(uid);
    if (c === undefined) {
      continue;
    }
    const root = c.parent_uid !== null && cats.has(c.parent_uid) ? c.parent_uid : uid;
    let row = rows.get(root);
    if (row === undefined) {
      row = {
        uid: root,
        name: cats.get(root)?.name ?? "",
        uncategorized: false,
        cents: 0,
        own_cents: 0,
        share_bps: 0,
        children: [],
      };
      rows.set(root, row);
      top.push(root);
    }
    row.cents = clamp(row.cents + cents);
    if (root === uid) {
      row.own_cents = clamp(row.own_cents + cents);
      continue;
    }
    row.children.push({
      uid,
      name: c.name,
      uncategorized: false,
      cents,
      own_cents: cents,
      share_bps: bps(cents, out.total_cents),
      children: [],
    });
  }

  for (const uid of top) {
    const row = rows.get(uid);
    if (row === undefined) {
      continue;
    }
    row.share_bps = bps(row.cents, out.total_cents);
    sortRows(row.children);
    out.rows.push(row);
  }
  if (uncategorized > 0) {
    out.rows.push({
      uid: "",
      name: "",
      uncategorized: true,
      cents: uncategorized,
      own_cents: uncategorized,
      share_bps: bps(uncategorized, out.total_cents),
      children: [],
    });
  }
  sortRows(out.rows);
  return out;
}

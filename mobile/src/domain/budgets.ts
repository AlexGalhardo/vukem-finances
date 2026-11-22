import type { Database } from "../db/database";
import type { Row } from "../db/entities";
import type { Repository } from "../db/repository";
import { addMonths } from "./dates";
import { uidTime } from "./ids";
import { LedgerError } from "./ledger";
import { MAX_CENTS } from "./money";

export interface BudgetStatus {
  uid: string;
  category_uid: string | null;
  category: string;
  overall: boolean;
  rollover: boolean;
  limit_cents: number;
  carry_cents: number;
  available_cents: number;
  spent_cents: number;
  percent: number;
}

export function validMonth(month: string): boolean {
  return /^\d{4}-(0[1-9]|1[0-2])$/u.test(month);
}

/** "YYYY-MM" before a valid month. */
export function previousMonth(month: string): string {
  return addMonths(`${month}-01`, -1).slice(0, 7);
}

/** Sums saturate at the largest representable amount, as on the server. */
export function satAdd(a: number, b: number): number {
  return Math.min(MAX_CENTS, a + b);
}

function satMul(amount: number, n: number): number {
  if (amount <= 0 || n <= 0) {
    return 0;
  }
  return amount > Math.floor(MAX_CENTS / n) ? MAX_CENTS : amount * n;
}

interface MonthSpending {
  total: number;
  /** A category's own spending plus its subcategories'. */
  tree: Map<string, number>;
}

/**
 * What a month spent: every expense that is not a transfer leg (uncategorized
 * ones too). Soft-deleted rows are out; deleted categories still sit in the
 * tree because a live transaction may name one. Grouped by amount and
 * multiplied with clamping so nothing overflows.
 */
async function spendingOf(db: Database, month: string): Promise<MonthSpending> {
  const groups = await db.all<{ category_uid: string | null; amount_cents: number; n: number }>(
    `SELECT category_uid, amount_cents, COUNT(*) AS n FROM transactions
		 WHERE deleted_at IS NULL AND transfer_uid IS NULL AND kind = 'expense' AND occurred_on BETWEEN ? AND ?
		 GROUP BY category_uid, amount_cents`,
    [`${month}-01`, `${month}-31`],
  );
  const cats = await db.all<{ uid: string; parent_uid: string | null }>("SELECT uid, parent_uid FROM categories");
  const parent = new Map(cats.filter((c) => c.parent_uid !== null).map((c) => [c.uid, c.parent_uid as string]));

  const out: MonthSpending = { total: 0, tree: new Map() };
  for (const g of groups) {
    const cents = satMul(g.amount_cents, g.n);
    out.total = satAdd(out.total, cents);
    if (g.category_uid === null) {
      continue;
    }
    out.tree.set(g.category_uid, satAdd(out.tree.get(g.category_uid) ?? 0, cents));
    const p = parent.get(g.category_uid);
    if (p !== undefined) {
      out.tree.set(p, satAdd(out.tree.get(p) ?? 0, cents));
    }
  }
  return out;
}

function percentOf(spent: number, available: number): number {
  if (available <= 0) {
    return 0;
  }
  // BigInt: spent * 100 can pass 2^53.
  return Number((BigInt(spent) * 100n) / BigInt(available));
}

/**
 * The budgets, the overall one first, with what `month` spent against each.
 * Carry is the unspent part of the limit of the previous month, only for a
 * rollover budget made before this month; it never goes negative and never
 * chains. The creation month comes from the ULID of the budget.
 */
export async function budgetStatuses(db: Database, month: string): Promise<BudgetStatus[]> {
  if (!validMonth(month)) {
    throw new LedgerError("month", "the month must look like 2026-11");
  }

  const budgets = await db.all<{
    uid: string;
    category_uid: string | null;
    limit_cents: number;
    rollover: number;
  }>("SELECT uid, category_uid, limit_cents, rollover FROM budgets WHERE deleted_at IS NULL ORDER BY uid");
  if (budgets.length === 0) {
    return [];
  }

  const names = new Map(
    (await db.all<{ uid: string; name: string }>("SELECT uid, name FROM categories WHERE deleted_at IS NULL")).map(
      (c) => [c.uid, c.name],
    ),
  );
  const cur = await spendingOf(db, month);
  const prev = budgets.some((b) => b.rollover === 1) ? await spendingOf(db, previousMonth(month)) : null;

  const out: BudgetStatus[] = [];
  for (const b of budgets) {
    const spendOf = (m: MonthSpending): number =>
      b.category_uid === null ? m.total : (m.tree.get(b.category_uid) ?? 0);
    const created = uidTime(b.uid);
    const createdMonth = created === null ? "" : new Date(created).toISOString().slice(0, 7);

    const carry =
      b.rollover === 1 && prev !== null && createdMonth < month ? Math.max(0, b.limit_cents - spendOf(prev)) : 0;
    const available = b.limit_cents + carry;
    const status: BudgetStatus = {
      uid: b.uid,
      category_uid: b.category_uid,
      category: b.category_uid === null ? "" : (names.get(b.category_uid) ?? ""),
      overall: b.category_uid === null,
      rollover: b.rollover === 1,
      limit_cents: b.limit_cents,
      carry_cents: carry,
      available_cents: available,
      spent_cents: spendOf(cur),
      percent: 0,
    };
    status.percent = percentOf(status.spent_cents, available);
    if (status.overall) {
      out.unshift(status);
    } else {
      out.push(status);
    }
  }
  return out;
}

export interface BudgetInput {
  /** null is the overall budget. */
  category_uid: string | null;
  limit_cents: number;
  /** undefined leaves the setting of an existing budget alone (off for a new one). */
  rollover?: boolean;
}

/**
 * Creates or updates the budget of an expense category, or the overall budget
 * for a null category. Writing twice leaves one live budget.
 */
export async function saveBudget(db: Database, repo: Repository, input: BudgetInput): Promise<Row<"budget">> {
  if (!Number.isSafeInteger(input.limit_cents) || input.limit_cents <= 0 || input.limit_cents > MAX_CENTS) {
    throw new LedgerError("limit", "a budget limit must be a positive amount");
  }

  return db.transaction(async (tx) => {
    const bound = repo.bind(tx);
    if (input.category_uid !== null) {
      const cat = await bound.get("category", input.category_uid);
      if (cat === null || cat.kind !== "expense") {
        throw new LedgerError("expense", "a budget belongs to an expense category");
      }
    }
    const existing = (await bound.list("budget")).find((b) => b.category_uid === input.category_uid);
    if (existing === undefined) {
      return bound.create("budget", {
        category_uid: input.category_uid,
        limit_cents: input.limit_cents,
        rollover: input.rollover ?? false,
      });
    }
    return bound.update("budget", existing.uid, {
      limit_cents: input.limit_cents,
      ...(input.rollover === undefined ? {} : { rollover: input.rollover }),
    });
  });
}

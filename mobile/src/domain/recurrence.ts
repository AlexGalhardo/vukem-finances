import type { Database } from "../db/database";
import type { Row, TransactionFields } from "../db/entities";
import type { Repository } from "../db/repository";
import { addDays, addMonths, isIsoDate } from "./dates";
import { newUid } from "./ids";
import { LedgerError } from "./ledger";
import { MAX_CENTS } from "./money";

/** docs/specs/recurrence.md: the rule of a repeating transaction. */
export type Frequency = "daily" | "weekly" | "monthly" | "yearly";

export interface Rule {
  frequency: Frequency;
  interval: number;
  start_date: string;
  end_date: string | null;
  max_count: number | null;
}

const MAX_YEAR = 9999;
/** The engine looks at no more than this many occurrences per call. */
const MAX_WALK = 10000;
/** At most this many occurrences are generated per rule per run. */
export const GENERATE_CAP = 500;
/** A stored rule and an installment purchase start and end between these years. */
const MIN_RULE_YEAR = 1900;
const MAX_RULE_YEAR = 2999;

export function validFrequency(f: string): f is Frequency {
  return f === "daily" || f === "weekly" || f === "monthly" || f === "yearly";
}

function year(date: string): number {
  return Number(date.slice(0, 4));
}

/** The date of the k-th occurrence, from the start date and k alone so a clamped day never drifts; null past year 9999. */
function nth(rule: Rule, k: number): string | null {
  if (rule.interval < 1 || k < 0) {
    return null;
  }
  const n = k * rule.interval;
  let d: string;
  switch (rule.frequency) {
    case "daily":
      d = addDays(rule.start_date, n);
      break;
    case "weekly":
      d = addDays(rule.start_date, 7 * n);
      break;
    case "monthly":
      d = addMonths(rule.start_date, n);
      break;
    case "yearly":
      d = addMonths(rule.start_date, 12 * n);
      break;
  }
  return /^\d{4}-/u.test(d) && year(d) <= MAX_YEAR ? d : null;
}

/** An index never later than the first occurrence on or after `from`, so a far window does not walk from k = 0. */
function firstIndex(rule: Rule, from: string): number {
  if (from <= rule.start_date || rule.interval < 1) {
    return 0;
  }
  const days = Math.floor(daysFromEpoch(from) - daysFromEpoch(rule.start_date));
  let units = 0;
  switch (rule.frequency) {
    case "daily":
      units = Math.floor(days / rule.interval);
      break;
    case "weekly":
      units = Math.floor(days / (7 * rule.interval));
      break;
    case "monthly": {
      const months =
        (year(from) - year(rule.start_date)) * 12 + Number(from.slice(5, 7)) - Number(rule.start_date.slice(5, 7));
      units = Math.floor(months / rule.interval);
      break;
    }
    case "yearly":
      units = Math.floor((year(from) - year(rule.start_date)) / rule.interval);
      break;
  }
  return Math.max(0, units - 1);
}

function daysFromEpoch(date: string): number {
  const d = new Date(0);
  d.setUTCFullYear(year(date), Number(date.slice(5, 7)) - 1, Number(date.slice(8, 10)));
  return d.getTime() / 86_400_000;
}

/**
 * The dates of the occurrences with from <= date <= to, ascending, honoring
 * the end date and the count (both count from the start date, not from
 * `from`). `limit` caps the length when above 0. Malformed input yields none.
 */
export function occurrences(rule: Rule, from: string, to: string, limit = 0): string[] {
  if (
    !isIsoDate(rule.start_date) ||
    !validFrequency(rule.frequency) ||
    rule.interval < 1 ||
    !isIsoDate(from) ||
    !isIsoDate(to) ||
    to < from ||
    (rule.end_date !== null && !isIsoDate(rule.end_date))
  ) {
    return [];
  }

  const out: string[] = [];
  const first = firstIndex(rule, from);
  for (let k = first; k < first + MAX_WALK; k++) {
    if (rule.max_count !== null && rule.max_count > 0 && k >= rule.max_count) {
      break;
    }
    const d = nth(rule, k);
    if (d === null || d > to || (rule.end_date !== null && d > rule.end_date)) {
      break;
    }
    if (d < from) {
      continue;
    }
    out.push(d);
    if (limit > 0 && out.length >= limit) {
      break;
    }
  }
  return out;
}

/** A client that writes a rule offline refuses what the server refuses. */
export function validateRule(rule: Rule): void {
  const bad = (message: string): LedgerError => new LedgerError("rule", message);
  if (!validFrequency(rule.frequency)) {
    throw bad("the frequency must be daily, weekly, monthly or yearly");
  }
  if (!Number.isSafeInteger(rule.interval) || rule.interval < 1) {
    throw bad("the interval must be at least 1");
  }
  if (rule.max_count !== null && (!Number.isSafeInteger(rule.max_count) || rule.max_count < 1)) {
    throw bad("the count must be at least 1");
  }
  for (const d of [rule.start_date, rule.end_date]) {
    if (d !== null && (!isIsoDate(d) || year(d) < MIN_RULE_YEAR || year(d) > MAX_RULE_YEAR)) {
      throw bad("dates must be real dates between 1900 and 2999");
    }
  }
}

export interface GenerateResult {
  created: number;
  /** Rules that stopped on an occurrence they could not make; the next run retries it. */
  failed: { recurrence_uid: string; reason: string }[];
}

/**
 * Makes the transactions that fell due for every active rule, from the day
 * after `generated_through` (or the start date) up to `today`, at most 500 per
 * rule. The natural key (recurrence_uid, occurrence_on) holds deleted rows
 * too, so an occurrence the person deleted is never recreated, and a second
 * run creates nothing. Each rule is one database transaction, its watermark
 * included. A rule whose account is archived or gone stops before that
 * occurrence (the watermark stays) without blocking the others.
 */
export async function generateDue(db: Database, repo: Repository, today: string): Promise<GenerateResult> {
  const result: GenerateResult = { created: 0, failed: [] };
  const rules = (await repo.list("recurrence")).filter((r) => r.active);

  for (const rule of rules) {
    await db.transaction(async (tx) => {
      const bound = repo.bind(tx);
      const fresh = await bound.get("recurrence", rule.uid);
      if (fresh === null || !fresh.active) {
        return;
      }
      const from = fresh.generated_through === null ? fresh.start_date : addDays(fresh.generated_through, 1);
      let done: string | null = null;

      for (const date of occurrences(ruleOf(fresh), from, today, GENERATE_CAP)) {
        const reason = await makeOccurrence(tx, bound, fresh, date);
        if (reason !== null && reason !== "held") {
          result.failed.push({ recurrence_uid: fresh.uid, reason });
          break;
        }
        if (reason === null) {
          result.created++;
        }
        done = date;
      }
      if (done !== null) {
        await bound.update("recurrence", fresh.uid, { generated_through: done });
      }
    });
  }
  return result;
}

export function ruleOf(
  r: Pick<Row<"recurrence">, "frequency" | "interval" | "start_date" | "end_date" | "max_count">,
): Rule {
  return {
    frequency: r.frequency as Frequency,
    interval: r.interval,
    start_date: r.start_date,
    end_date: r.end_date,
    max_count: r.max_count,
  };
}

/** Returns null when it made the row, "held" when the natural key is already taken, or the reason it could not. */
async function makeOccurrence(
  db: Database,
  repo: Repository,
  rule: Row<"recurrence">,
  date: string,
): Promise<string | null> {
  const held = await db.all<{ uid: string }>(
    "SELECT uid FROM transactions WHERE recurrence_uid = ? AND occurrence_on = ?",
    [rule.uid, date],
  );
  if (held.length > 0) {
    return "held";
  }
  const account = await repo.get("account", rule.account_uid);
  if (account === null) {
    return "account_deleted";
  }
  if (account.archived_at !== null) {
    return "account_archived";
  }
  const category = rule.category_uid === null ? null : await repo.get("category", rule.category_uid);

  await repo.create("transaction", {
    account_uid: rule.account_uid,
    category_uid: category === null ? null : category.uid,
    kind: rule.kind,
    amount_cents: rule.amount_cents,
    description: rule.description,
    occurred_on: date,
    transfer_uid: null,
    tag_uids: [],
    statement_ref: null,
    recurrence_uid: rule.uid,
    occurrence_on: date,
    installment_group_uid: null,
    installment_no: null,
    installment_total: null,
  });
  return null;
}

const MIN_PARTS = 2;
const MAX_PARTS = 60;

/**
 * Divides total cents into parts that add up to total exactly: the first
 * total % parts parts get one extra cent (100.00 in 3 is 33.34, 33.33, 33.33).
 * Needs parts >= 1 and total >= parts.
 */
export function splitInstallments(total: number, parts: number): number[] {
  const base = Math.floor(total / parts);
  const extra = total % parts;
  return Array.from({ length: parts }, (_, i) => (i < extra ? base + 1 : base));
}

/** Part i (0-based) is on addMonths(first, i), so a purchase on the 31st keeps month ends. */
export function installmentDates(first: string, parts: number): string[] {
  return Array.from({ length: parts }, (_, i) => addMonths(first, i));
}

/** How a part reads in a list: "Description (2/6)". */
export function installmentLabel(description: string, no: number, total: number): string {
  return `${description} (${no}/${total})`;
}

export interface InstallmentInput {
  account_uid: string;
  category_uid: string | null;
  total_cents: number;
  parts: number;
  first_date: string;
  description: string;
  tag_uids?: string[];
}

/** Writes all the parts of an installment purchase in one transaction; any failure writes none. */
export async function createInstallments(
  db: Database,
  repo: Repository,
  input: InstallmentInput,
): Promise<Row<"transaction">[]> {
  if (!Number.isInteger(input.parts) || input.parts < MIN_PARTS || input.parts > MAX_PARTS) {
    throw new LedgerError("parts", "an installment purchase has 2 to 60 parts");
  }
  if (!Number.isSafeInteger(input.total_cents) || input.total_cents <= 0 || input.total_cents > MAX_CENTS) {
    throw new LedgerError("amount", "the total must be a positive amount");
  }
  if (input.total_cents < input.parts) {
    throw new LedgerError("total", "the total is too small: every part must be at least one cent");
  }
  if (!isIsoDate(input.first_date)) {
    throw new LedgerError("date", "the first date must be a real date");
  }
  if (year(input.first_date) < MIN_RULE_YEAR || year(input.first_date) > MAX_RULE_YEAR) {
    throw new LedgerError("rule", "dates must be between 1900 and 2999");
  }

  const amounts = splitInstallments(input.total_cents, input.parts);
  const dates = installmentDates(input.first_date, input.parts);
  const group = newUid();

  return db.transaction(async (tx) => {
    const bound = repo.bind(tx);
    const account = await bound.get("account", input.account_uid);
    if (account === null) {
      throw new LedgerError("not_found", "account not found");
    }
    if (account.archived_at !== null) {
      throw new LedgerError("archived", "an archived account cannot take a purchase");
    }
    const out: Row<"transaction">[] = [];
    for (let i = 0; i < amounts.length; i++) {
      const fields: TransactionFields = {
        account_uid: input.account_uid,
        category_uid: input.category_uid,
        kind: "expense",
        amount_cents: amounts[i] ?? 0,
        description: input.description,
        occurred_on: dates[i] ?? input.first_date,
        transfer_uid: null,
        tag_uids: input.tag_uids ?? [],
        statement_ref: null,
        recurrence_uid: null,
        occurrence_on: null,
        installment_group_uid: group,
        installment_no: i + 1,
        installment_total: input.parts,
      };
      out.push(await bound.create("transaction", fields));
    }
    return out;
  });
}

const UPCOMING_CAP = 200;

export interface UpcomingItem {
  date: string;
  kind: string;
  amount_cents: number;
  description: string;
  source: "transaction" | "installment" | "recurrence";
  account_uid: string;
  /** Set for stored rows. */
  transaction_uid: string | null;
  /** Set for virtual occurrences: computed, never stored. */
  recurrence_uid: string | null;
  installment_no: number | null;
  installment_total: number | null;
}

/**
 * What is due in the next `days` days: stored transactions dated after today
 * (installment parts included) and the occurrences of active rules, from
 * today on, that no row holds yet (a row, even a deleted one, or the
 * watermark of the rule hides the virtual occurrence, so nothing shows twice).
 */
export async function upcoming(db: Database, repo: Repository, today: string, days: number): Promise<UpcomingItem[]> {
  const end = addDays(today, Math.max(days, 1));
  const stored = await db.all<{
    uid: string;
    occurred_on: string;
    kind: string;
    amount_cents: number;
    description: string;
    account_uid: string;
    installment_group_uid: string | null;
    installment_no: number | null;
    installment_total: number | null;
    recurrence_uid: string | null;
  }>(
    `SELECT uid, occurred_on, kind, amount_cents, description, account_uid, installment_group_uid, installment_no,
		        installment_total, recurrence_uid
		 FROM transactions WHERE deleted_at IS NULL AND transfer_uid IS NULL AND occurred_on > ? AND occurred_on <= ?
		 ORDER BY occurred_on, uid LIMIT ?`,
    [today, end, UPCOMING_CAP],
  );

  const out: UpcomingItem[] = stored.map((r) => ({
    date: r.occurred_on,
    kind: r.kind,
    amount_cents: r.amount_cents,
    description: r.description,
    source: r.installment_group_uid !== null ? "installment" : r.recurrence_uid !== null ? "recurrence" : "transaction",
    account_uid: r.account_uid,
    transaction_uid: r.uid,
    recurrence_uid: null,
    installment_no: r.installment_no,
    installment_total: r.installment_total,
  }));

  for (const rule of (await repo.list("recurrence")).filter((r) => r.active)) {
    const next = rule.generated_through === null ? null : addDays(rule.generated_through, 1);
    const from = next !== null && next > today ? next : today;
    const dates = occurrences(ruleOf(rule), from, end, UPCOMING_CAP);
    if (dates.length === 0) {
      continue;
    }
    const held = new Set(
      (
        await db.all<{ occurrence_on: string }>("SELECT occurrence_on FROM transactions WHERE recurrence_uid = ?", [
          rule.uid,
        ])
      ).map((h) => h.occurrence_on),
    );
    for (const d of dates.filter((x) => !held.has(x))) {
      out.push({
        date: d,
        kind: rule.kind,
        amount_cents: rule.amount_cents,
        description: rule.description,
        source: "recurrence",
        account_uid: rule.account_uid,
        transaction_uid: null,
        recurrence_uid: rule.uid,
        installment_no: null,
        installment_total: null,
      });
    }
  }

  return out.sort((a, b) => (a.date < b.date ? -1 : a.date > b.date ? 1 : 0)).slice(0, UPCOMING_CAP);
}

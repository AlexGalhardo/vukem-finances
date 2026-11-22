import type { Database } from "../db/database";
import { daysInMonth, isIsoDate } from "./dates";

/** Statement status, derived from today, the closing and due dates and what has been paid; never stored. */
export type StatementStatus = "open" | "closed" | "paid" | "overdue";

function pad(n: number, width: number): string {
	return String(n).padStart(width, "0");
}

/**
 * The day a card statement closes in a month: the closing day, or the last
 * day of a shorter month (day 31 closes on Feb 28 or 29). `month` may
 * overflow (13 is January of the next year). Port of Go's closingDate.
 */
export function closingDate(year: number, month: number, closingDay: number): string {
	const index = year * 12 + (month - 1);
	const y = Math.floor(index / 12);
	const m = (((index % 12) + 12) % 12) + 1;
	return `${pad(y, 4)}-${pad(m, 2)}-${pad(Math.min(closingDay, daysInMonth(y, m)), 2)}`;
}

function parts(date: string): { year: number; month: number } {
	return { year: Number(date.slice(0, 4)), month: Number(date.slice(5, 7)) };
}

/**
 * The closing date of the statement a purchase made on `date` belongs to: the
 * closing of its own month when the purchase is on or before it, the next
 * month's otherwise. The statement is identified by that closing date.
 */
export function statementFor(date: string, closingDay: number): string {
	const { year, month } = parts(date);
	const c = closingDate(year, month, closingDay);
	return date > c ? closingDate(year, month + 1, closingDay) : c;
}

/** statementFor on an ISO date; "" for text that is not a date. */
export function statementRefFor(date: string, closingDay: number): string {
	return isIsoDate(date) ? statementFor(date, closingDay) : "";
}

/** The first occurrence of the due day strictly after the closing date, clamped to the month. */
export function dueDate(closing: string, dueDay: number): string {
	const { year, month } = parts(closing);
	const d = closingDate(year, month, dueDay);
	return d > closing ? d : closingDate(year, month + 1, dueDay);
}

/** Dates compare as text. `total` is what was bought (net of refunds), `paid` what the payments tagged with the statement add up to. */
export function statementStatus(today: string, closing: string, due: string, total: number, paid: number): StatementStatus {
	if (today <= closing) {
		return "open";
	}
	if (paid >= total) {
		return "paid";
	}
	return today > due ? "overdue" : "closed";
}

export interface StatementSummary {
	/** The closing date, the identity of the statement. */
	ref: string;
	closing_on: string;
	due_on: string;
	purchases_cents: number;
	credits_cents: number;
	/** Purchases minus credits (refunds). */
	total_cents: number;
	paid_cents: number;
	remaining_cents: number;
	status: StatementStatus;
}

export interface CardRule {
	account_uid: string;
	closing_day: number;
	due_day: number;
}

/**
 * The statements of a card, newest closing first: every one that has rows
 * plus the one open today. Purchases and refunds are the card account's rows
 * that are not transfers, tagged with their statement; payments are the
 * income legs of a transfer tagged with the statement they pay.
 */
export async function cardStatements(db: Database, card: CardRule, today: string): Promise<StatementSummary[]> {
	const rows = await db.all<{ ref: string; kind: string; is_transfer: number; cents: number }>(
		`SELECT statement_ref AS ref, kind, transfer_uid IS NOT NULL AS is_transfer, SUM(amount_cents) AS cents
		 FROM transactions
		 WHERE deleted_at IS NULL AND account_uid = ? AND statement_ref IS NOT NULL
		 GROUP BY statement_ref, kind, transfer_uid IS NOT NULL`,
		[card.account_uid],
	);

	const byRef = new Map<string, { purchases: number; credits: number; paid: number }>();
	const get = (ref: string): { purchases: number; credits: number; paid: number } => {
		let s = byRef.get(ref);
		if (s === undefined) {
			s = { purchases: 0, credits: 0, paid: 0 };
			byRef.set(ref, s);
		}
		return s;
	};
	for (const r of rows) {
		const s = get(r.ref);
		if (r.is_transfer === 1) {
			if (r.kind === "income") {
				s.paid += r.cents;
			}
		} else if (r.kind === "expense") {
			s.purchases += r.cents;
		} else {
			s.credits += r.cents;
		}
	}

	const current = statementRefFor(today, card.closing_day);
	if (current !== "") {
		get(current);
	}

	const out: StatementSummary[] = [];
	for (const [ref, s] of byRef) {
		if (!isIsoDate(ref)) {
			continue;
		}
		const due = dueDate(ref, card.due_day);
		const total = s.purchases - s.credits;
		out.push({
			ref,
			closing_on: ref,
			due_on: due,
			purchases_cents: s.purchases,
			credits_cents: s.credits,
			total_cents: total,
			paid_cents: s.paid,
			remaining_cents: Math.max(0, total - s.paid),
			status: statementStatus(today, ref, due, total, s.paid),
		});
	}
	return out.sort((a, b) => (a.ref < b.ref ? 1 : a.ref > b.ref ? -1 : 0));
}

/**
 * The statement a purchase on `accountUid` dated `date` belongs to: null
 * unless the account is a live credit card. The server derives it the same way.
 */
export async function statementRefIn(db: Database, accountUid: string, date: string): Promise<string | null> {
	const cards = await db.all<{ closing_day: number }>(
		"SELECT closing_day FROM credit_cards WHERE deleted_at IS NULL AND account_uid = ?",
		[accountUid],
	);
	const card = cards[0];
	if (card === undefined) {
		return null;
	}
	const ref = statementRefFor(date, card.closing_day);
	return ref === "" ? null : ref;
}

import type { Database } from "../db/database";
import type { CategoryFields, Row, TransactionFields } from "../db/entities";
import type { Repository } from "../db/repository";
import { newUid } from "./ids";
import { MAX_CENTS } from "./money";

/** Rule violations the screens turn into messages. */
export class LedgerError extends Error {
	constructor(
		readonly code: string,
		message: string,
	) {
		super(message);
		this.name = "LedgerError";
	}
}

export interface AccountBalance {
	account_uid: string;
	opening_cents: number;
	movement_cents: number;
	balance_cents: number;
	archived: boolean;
}

function addCents(a: number, b: number): number {
	const sum = a + b;
	if (!Number.isSafeInteger(sum) || Math.abs(sum) > MAX_CENTS) {
		throw new LedgerError("balance_range", "a balance is beyond the supported range");
	}
	return sum;
}

/**
 * One balance per live account (archived ones flagged) up to `asOf`, an ISO
 * date taken inclusively: opening plus income minus expense. Both legs of a
 * transfer count, each on its own account, so a transfer moves money between
 * balances and changes no total.
 */
export async function accountBalances(db: Database, asOf: string): Promise<AccountBalance[]> {
	const accounts = await db.all<{ uid: string; opening_cents: number; archived_at: string | null }>(
		"SELECT uid, opening_cents, archived_at FROM accounts WHERE deleted_at IS NULL ORDER BY position, name, uid",
	);
	const moves = await db.all<{ account_uid: string; kind: string; cents: number }>(
		`SELECT account_uid, kind, SUM(amount_cents) AS cents FROM transactions
		 WHERE deleted_at IS NULL AND occurred_on <= ? GROUP BY account_uid, kind`,
		[asOf],
	);

	const signed = new Map<string, number>();
	for (const m of moves) {
		const delta = m.kind === "income" ? m.cents : -m.cents;
		signed.set(m.account_uid, addCents(signed.get(m.account_uid) ?? 0, delta));
	}

	return accounts.map((a) => {
		const movement = signed.get(a.uid) ?? 0;
		return {
			account_uid: a.uid,
			opening_cents: a.opening_cents,
			movement_cents: movement,
			balance_cents: addCents(a.opening_cents, movement),
			archived: a.archived_at !== null,
		};
	});
}

/** The sum of the balances of accounts that are not archived. */
export function netWorthOf(balances: readonly AccountBalance[]): number {
	return balances.reduce((total, b) => (b.archived ? total : addCents(total, b.balance_cents)), 0);
}

export interface TransferInput {
	from_account_uid: string;
	to_account_uid: string;
	amount_cents: number;
	date: string;
	description: string;
}

/**
 * Account, transfer and category rules over the repositories, mirroring the
 * server (finance services): cascades write one outbox entry per row.
 */
export class Ledger {
	constructor(
		private readonly db: Database,
		private readonly repo: Repository,
	) {}

	/** Two legs sharing a transfer_uid: an expense on the source, an income on the destination, no category. */
	async createTransfer(input: TransferInput): Promise<{ transfer_uid: string; source: string; destination: string }> {
		if (!Number.isSafeInteger(input.amount_cents) || input.amount_cents <= 0 || input.amount_cents > MAX_CENTS) {
			throw new LedgerError("amount", "a transfer needs a positive amount");
		}
		if (input.from_account_uid === input.to_account_uid) {
			throw new LedgerError("same_account", "choose two different accounts");
		}

		return this.db.transaction(async () => {
			for (const uid of [input.from_account_uid, input.to_account_uid]) {
				const account = await this.repo.get("account", uid);
				if (account === null) {
					throw new LedgerError("not_found", "account not found");
				}
				if (account.archived_at !== null) {
					throw new LedgerError("archived", "an archived account cannot take a transfer");
				}
			}

			const transfer_uid = newUid();
			const leg = (account_uid: string, kind: string): TransactionFields => ({
				account_uid,
				category_uid: null,
				kind,
				amount_cents: input.amount_cents,
				description: input.description.trim(),
				occurred_on: input.date,
				transfer_uid,
				tag_uids: [],
				statement_ref: null,
				recurrence_uid: null,
				occurrence_on: null,
				installment_group_uid: null,
				installment_no: null,
				installment_total: null,
			});
			const source = await this.repo.create("transaction", leg(input.from_account_uid, "expense"));
			const destination = await this.repo.create("transaction", leg(input.to_account_uid, "income"));

			return { transfer_uid, source: source.uid, destination: destination.uid };
		});
	}

	/** The other leg of a transfer, if the row is one. */
	private async partner(tx: Row<"transaction">, includeDeleted: boolean): Promise<Row<"transaction"> | null> {
		if (tx.transfer_uid === null) {
			return null;
		}
		const rows = await this.repo.list("transaction", includeDeleted);
		return rows.find((r) => r.transfer_uid === tx.transfer_uid && r.uid !== tx.uid) ?? null;
	}

	/** Deleting a leg of a transfer deletes the other one with it. */
	async deleteTransaction(uid: string): Promise<void> {
		await this.db.transaction(async () => {
			const tx = await this.repo.get("transaction", uid);
			if (tx === null) {
				throw new LedgerError("not_found", "transaction not found");
			}
			const at = this.repo.nowStamp();
			const other = await this.partner(tx, false);
			await this.repo.remove("transaction", uid, at);
			if (other !== null) {
				await this.repo.remove("transaction", other.uid, at);
			}
		});
	}

	/** A transaction comes back only if its account is alive; a transfer leg brings the other leg back. */
	async restoreTransaction(uid: string): Promise<void> {
		await this.db.transaction(async () => {
			const tx = await this.repo.get("transaction", uid, true);
			if (tx === null) {
				throw new LedgerError("not_found", "transaction not found");
			}
			const other = await this.partner(tx, true);
			for (const t of other === null ? [tx] : [tx, other]) {
				if ((await this.repo.get("account", t.account_uid)) === null) {
					throw new LedgerError("account_deleted", "restore the account first");
				}
			}
			const at = this.repo.nowStamp();
			await this.repo.restore("transaction", uid, at);
			if (other !== null) {
				await this.repo.restore("transaction", other.uid, at);
			}
		});
	}

	/**
	 * Deletes an account, its transactions, the other leg of every transfer it
	 * took part in (so no transfer keeps one side) and its card, all stamped
	 * with one instant.
	 */
	async deleteAccount(uid: string): Promise<void> {
		await this.db.transaction(async () => {
			if ((await this.repo.get("account", uid)) === null) {
				throw new LedgerError("not_found", "account not found");
			}
			const at = this.repo.nowStamp();
			const all = await this.repo.list("transaction");
			const own = all.filter((t) => t.account_uid === uid);
			const transfers = new Set(own.map((t) => t.transfer_uid).filter((u): u is string => u !== null));
			const doomed = all.filter((t) => t.account_uid === uid || (t.transfer_uid !== null && transfers.has(t.transfer_uid)));
			for (const t of doomed) {
				await this.repo.remove("transaction", t.uid, at);
			}
			for (const card of await this.repo.list("credit_card")) {
				if (card.account_uid === uid) {
					await this.repo.remove("credit_card", card.uid, at);
				}
			}
			await this.repo.remove("account", uid, at);
		});
	}

	/** Brings back an account and what was deleted together with it (same instant). */
	async restoreAccount(uid: string): Promise<void> {
		await this.db.transaction(async () => {
			const account = await this.repo.get("account", uid, true);
			if (account === null || account.deleted_at === null) {
				throw new LedgerError("not_found", "deleted account not found");
			}
			const stamp = account.deleted_at;
			const at = this.repo.nowStamp();
			const together = (await this.repo.list("transaction", true)).filter((t) => t.deleted_at === stamp);

			for (const t of together) {
				if (t.account_uid === uid) {
					continue;
				}
				if ((await this.repo.get("account", t.account_uid)) === null) {
					throw new LedgerError("account_deleted", "restore the paired account first");
				}
			}
			await this.repo.restore("account", uid, at);
			for (const t of together) {
				await this.repo.restore("transaction", t.uid, at);
			}
			for (const card of await this.repo.list("credit_card", true)) {
				if (card.account_uid === uid && card.deleted_at === stamp) {
					await this.repo.restore("credit_card", card.uid, at);
				}
			}
		});
	}

	/** A subcategory goes under a live, top-level category of the same kind and takes its kind. */
	async createCategory(input: Omit<CategoryFields, "archived_at" | "kind"> & { kind: string }): Promise<Row<"category">> {
		return this.db.transaction(async () => {
			let kind = input.kind;
			if (input.parent_uid !== null) {
				const parent = await this.repo.get("category", input.parent_uid);
				if (parent === null) {
					throw new LedgerError("not_found", "parent category not found");
				}
				if (parent.parent_uid !== null) {
					throw new LedgerError("parent", "a subcategory goes under a top-level category");
				}
				if (parent.archived_at !== null) {
					throw new LedgerError("parent_archived", "unarchive the parent category first");
				}
				kind = parent.kind;
			}
			return this.repo.create("category", { ...input, kind, archived_at: null });
		});
	}

	/** A category never changes kind, and one with subcategories cannot become a subcategory. */
	async moveCategory(uid: string, parent_uid: string | null): Promise<Row<"category">> {
		return this.db.transaction(async () => {
			const c = await this.repo.get("category", uid);
			if (c === null) {
				throw new LedgerError("not_found", "category not found");
			}
			if (parent_uid !== null) {
				const parent = await this.repo.get("category", parent_uid);
				if (parent === null || parent.parent_uid !== null || parent.kind !== c.kind || parent.uid === uid) {
					throw new LedgerError("parent", "a subcategory goes under a top-level category of the same kind");
				}
				const kids = (await this.repo.list("category")).some((k) => k.parent_uid === uid);
				if (kids) {
					throw new LedgerError("nested", "a category with subcategories cannot become a subcategory");
				}
			}
			return this.repo.update("category", uid, { parent_uid });
		});
	}

	/**
	 * Deletes a category (never one with subcategories). Its budget and its
	 * transactions and recurrences either move to `reassignTo` (a live
	 * category of the same kind) or the budget is deleted and the others
	 * become uncategorized, each change one outbox entry.
	 */
	async deleteCategory(uid: string, reassignTo: string | null = null): Promise<void> {
		await this.db.transaction(async () => {
			const c = await this.repo.get("category", uid);
			if (c === null) {
				throw new LedgerError("not_found", "category not found");
			}
			if ((await this.repo.list("category")).some((k) => k.parent_uid === uid)) {
				throw new LedgerError("children", "move or delete its subcategories first");
			}
			if (reassignTo !== null) {
				const target = await this.repo.get("category", reassignTo);
				if (target === null || target.kind !== c.kind || target.uid === uid) {
					throw new LedgerError("reassign", "choose another category of the same kind");
				}
			}

			const at = this.repo.nowStamp();
			for (const t of await this.repo.list("transaction")) {
				if (t.category_uid === uid) {
					await this.repo.update("transaction", t.uid, { category_uid: reassignTo });
				}
			}
			for (const r of await this.repo.list("recurrence")) {
				if (r.category_uid === uid) {
					await this.repo.update("recurrence", r.uid, { category_uid: reassignTo });
				}
			}
			for (const b of await this.repo.list("budget")) {
				if (b.category_uid !== uid) {
					continue;
				}
				const targetHasBudget =
					reassignTo !== null && (await this.repo.list("budget")).some((o) => o.category_uid === reassignTo);
				if (reassignTo === null || targetHasBudget) {
					await this.repo.remove("budget", b.uid, at);
				} else {
					await this.repo.update("budget", b.uid, { category_uid: reassignTo });
				}
			}
			await this.repo.remove("category", uid, at);
		});
	}

	/** Deleting a tag removes its uid from every transaction that held it (each change logged). */
	async deleteTag(uid: string): Promise<void> {
		await this.db.transaction(async () => {
			if ((await this.repo.get("tag", uid)) === null) {
				throw new LedgerError("not_found", "tag not found");
			}
			for (const t of await this.repo.list("transaction")) {
				if (t.tag_uids.includes(uid)) {
					await this.repo.update("transaction", t.uid, { tag_uids: t.tag_uids.filter((x) => x !== uid) });
				}
			}
			await this.repo.remove("tag", uid);
		});
	}
}

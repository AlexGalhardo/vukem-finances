import type { AccountFields, TransactionFields } from "../../src/db/entities";
import { Repository } from "../../src/db/repository";
import { Ledger } from "../../src/domain/ledger";
import { type MemoryDatabase, openMemoryDatabase } from "./memory-db";

export const ACCOUNT: AccountFields = { name: "Checking", kind: "checking", opening_cents: 0, archived_at: null, position: 0 };

export function txFields(account: string, over: Partial<TransactionFields> = {}): TransactionFields {
	return {
		account_uid: account,
		category_uid: null,
		kind: "expense",
		amount_cents: 1000,
		description: "",
		occurred_on: "2026-11-10",
		transfer_uid: null,
		tag_uids: [],
		statement_ref: null,
		recurrence_uid: null,
		occurrence_on: null,
		installment_group_uid: null,
		installment_no: null,
		installment_total: null,
		...over,
	};
}

export interface Env {
	db: MemoryDatabase;
	repo: Repository;
	ledger: Ledger;
}

/** A database with a deterministic clock that ticks one second per write stamp. */
export async function openEnv(): Promise<Env> {
	const db = await openMemoryDatabase();
	let tick = 0;
	const repo = new Repository(db, {
		now: () => new Date(Date.UTC(2026, 10, 21, 10, 0, tick++)).toISOString(),
	});
	return { db, repo, ledger: new Ledger(db, repo) };
}

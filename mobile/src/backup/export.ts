import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";

export interface BackupWorkspace {
	name: string;
	currency: string;
	timezone: string;
}

/** The columns of each table that go into the backup: the synced fields of the entity, by their wire names. */
async function rowsOf(db: Database, entity: EntityName): Promise<Record<string, unknown>[]> {
	const meta = ENTITIES[entity];
	const fields = Object.keys(meta.fields);
	const raw = await db.all<Record<string, string | number | null>>(
		`SELECT uid, ${fields.map((f) => `"${f}"`).join(", ")} FROM ${meta.table} WHERE deleted_at IS NULL ORDER BY ${meta.order}`,
	);
	return raw.map((r) => {
		const out: Record<string, unknown> = { uid: r.uid };
		for (const f of fields) {
			const kind = meta.fields[f];
			const v = r[f] ?? null;
			out[f] = kind === "json" ? (typeof v === "string" ? (JSON.parse(v) as unknown) : []) : kind === "bool" ? v === 1 : v;
		}
		return out;
	});
}

/**
 * The whole local database as the JSON of docs/export-format.md (format
 * "vukem-finances-export", version 1): live rows only, money as integer cents,
 * references by uid. The device has no members and no goals yet. Nothing secret
 * is in it: the lock setting lives in secure storage and is never read here.
 */
export async function buildBackup(db: Database, workspace: BackupWorkspace, now: Date): Promise<Record<string, unknown>> {
	const [accounts, categories, tags, transactions, budgets, recurrences, cards] = await Promise.all([
		rowsOf(db, "account"),
		rowsOf(db, "category"),
		rowsOf(db, "tag"),
		rowsOf(db, "transaction"),
		rowsOf(db, "budget"),
		rowsOf(db, "recurrence"),
		rowsOf(db, "credit_card"),
	]);

	return {
		format: "vukem-finances-export",
		version: 1,
		exported_at: now.toISOString().replace(/\.\d{3}Z$/u, "Z"),
		workspace: { name: workspace.name, currency: workspace.currency, timezone: workspace.timezone, members: [] },
		accounts,
		categories,
		tags,
		transactions,
		budgets,
		goals: [],
		recurrences,
		cards,
	};
}

/** The file name the server uses for its exports, with the date of the workspace. */
export function backupFileName(today: string): string {
	return `vukem-finances-backup-${today}.json`;
}

import { MAX_CENTS } from "../domain/money";
import { newUid } from "../domain/ids";
import { type Database, getOne, type SqlValue } from "./database";
import { ENTITIES, type EntityFields, type EntityName, type FieldKind, type Row } from "./entities";

export type OutboxOp = "upsert" | "delete" | "restore";

export interface OutboxEntry {
	id: number;
	op_id: string;
	entity: EntityName;
	uid: string;
	op: OutboxOp;
	base_seq: number;
	/** Only the fields that changed; empty for a delete or a restore. */
	fields: Record<string, unknown>;
	attempts: number;
	created_at: string;
}

/** The outbox keeps growing with no server; past this the app asks before dropping anything. */
export const OUTBOX_SOFT_CAP = 5000;

export interface RepositoryOptions {
	now?: () => string;
	uid?: () => string;
}

interface RawRow {
	[column: string]: SqlValue;
}

function encode(kind: FieldKind, value: unknown): SqlValue {
	if (value === null || value === undefined) {
		return null;
	}
	switch (kind) {
		case "json":
			return JSON.stringify(value);
		case "bool":
			return value === true ? 1 : 0;
		default:
			return value as string | number;
	}
}

function decode(kind: FieldKind, value: SqlValue): unknown {
	switch (kind) {
		case "json":
			return typeof value === "string" ? (JSON.parse(value) as unknown) : [];
		case "bool":
			return value === 1;
		default:
			return value;
	}
}

function checkMoney(name: string, value: unknown): void {
	if (!name.endsWith("_cents") || value === null || value === undefined) {
		return;
	}
	if (typeof value !== "number" || !Number.isSafeInteger(value) || Math.abs(value) > MAX_CENTS) {
		throw new RangeError(`${name} must be an integer number of cents`);
	}
}

function sameValue(a: unknown, b: unknown): boolean {
	return JSON.stringify(a) === JSON.stringify(b);
}

function toRow<E extends EntityName>(entity: E, raw: RawRow): Row<E> {
	const row: Record<string, unknown> = {
		uid: raw.uid,
		version: raw.version,
		server_seq: raw.server_seq,
		deleted_at: raw.deleted_at,
		dirty: raw.dirty === 1,
		updated_local_at: raw.updated_local_at,
	};
	for (const [name, kind] of Object.entries(ENTITIES[entity].fields)) {
		row[name] = decode(kind, raw[name] ?? null);
	}
	return row as unknown as Row<E>;
}

export class OutboxCapError extends Error {
	constructor() {
		super("the sync queue is over its cap and needs a confirmation");
		this.name = "OutboxCapError";
	}
}

/**
 * Every write goes through here. In one SQLite transaction it changes the
 * row, bumps `version`, marks the row `dirty` and appends the mutation to the
 * outbox with only the changed fields (protocol: `upsert`, `delete`,
 * `restore`; `base_seq` is the newest server seq seen, 0 for a new row).
 */
export class Repository {
	private readonly now: () => string;
	private readonly uid: () => string;

	constructor(
		private readonly db: Database,
		options: RepositoryOptions = {},
	) {
		this.now = options.now ?? ((): string => new Date().toISOString());
		this.uid = options.uid ?? ((): string => newUid());
	}

	async get<E extends EntityName>(entity: E, uid: string, includeDeleted = false): Promise<Row<E> | null> {
		const raw = await getOne<RawRow>(
			this.db,
			`SELECT * FROM ${ENTITIES[entity].table} WHERE uid = ?${includeDeleted ? "" : " AND deleted_at IS NULL"}`,
			[uid],
		);
		return raw === null ? null : toRow(entity, raw);
	}

	async list<E extends EntityName>(entity: E, includeDeleted = false): Promise<Row<E>[]> {
		const rows = await this.db.all<RawRow>(
			`SELECT * FROM ${ENTITIES[entity].table}${includeDeleted ? "" : " WHERE deleted_at IS NULL"} ORDER BY uid`,
		);
		return rows.map((r) => toRow(entity, r));
	}

	async create<E extends EntityName>(entity: E, fields: EntityFields[E], uid: string = this.uid()): Promise<Row<E>> {
		const meta = ENTITIES[entity];
		const names = Object.keys(meta.fields);
		const values = fields as unknown as Record<string, unknown>;
		for (const name of names) {
			checkMoney(name, values[name]);
		}

		await this.db.transaction(async () => {
			const stamp = this.now();
			const columns = ["uid", "version", "dirty", "updated_local_at", ...names.map(quote)];
			const params: SqlValue[] = [uid, 1, 1, stamp, ...names.map((n) => encode(meta.fields[n] ?? "text", values[n]))];
			await this.db.run(
				`INSERT INTO ${meta.table} (${columns.join(", ")}) VALUES (${columns.map(() => "?").join(", ")})`,
				params,
			);
			await this.syncTags(entity, uid, values);
			const sent: Record<string, unknown> = {};
			for (const name of names) {
				sent[name] = values[name] ?? null;
			}
			await this.enqueue(entity, uid, "upsert", 0, sent, stamp);
		});

		return (await this.mustGet(entity, uid)) as Row<E>;
	}

	/** Writes only the fields whose value changed; a call that changes nothing writes nothing. */
	async update<E extends EntityName>(entity: E, uid: string, changes: Partial<EntityFields[E]>): Promise<Row<E>> {
		const meta = ENTITIES[entity];
		const current = await this.mustGet(entity, uid);
		const wanted = changes as Record<string, unknown>;
		const before = current as unknown as Record<string, unknown>;
		const changed: Record<string, unknown> = {};

		for (const name of Object.keys(wanted)) {
			if (!(name in meta.fields)) {
				throw new Error(`${entity} has no field ${name}`);
			}
			checkMoney(name, wanted[name]);
			if (!sameValue(before[name], wanted[name] ?? null)) {
				changed[name] = wanted[name] ?? null;
			}
		}
		const names = Object.keys(changed);
		if (names.length === 0) {
			return current;
		}

		await this.db.transaction(async () => {
			const stamp = this.now();
			const sets = names.map((n) => `${quote(n)} = ?`).join(", ");
			const params: SqlValue[] = names.map((n) => encode(meta.fields[n] ?? "text", changed[n]));
			await this.db.run(
				`UPDATE ${meta.table} SET ${sets}, version = version + 1, dirty = 1, updated_local_at = ? WHERE uid = ?`,
				[...params, stamp, uid],
			);
			await this.syncTags(entity, uid, changed);
			await this.enqueue(entity, uid, "upsert", current.server_seq ?? 0, changed, stamp);
		});

		return (await this.mustGet(entity, uid)) as Row<E>;
	}

	/** `at` lets a cascade stamp every row it deletes with one instant, so a restore can find them together. */
	async remove(entity: EntityName, uid: string, at?: string): Promise<void> {
		const current = await this.mustGet(entity, uid);
		await this.mark(entity, uid, current.server_seq ?? 0, "delete", at);
	}

	async restore(entity: EntityName, uid: string, at?: string): Promise<void> {
		const current = await this.mustGet(entity, uid, true);
		if (current.deleted_at === null) {
			return;
		}
		await this.mark(entity, uid, current.server_seq ?? 0, "restore", at);
	}

	nowStamp(): string {
		return this.now();
	}

	async outbox(): Promise<OutboxEntry[]> {
		const rows = await this.db.all<RawRow>("SELECT * FROM outbox ORDER BY id");
		return rows.map((r) => ({
			id: r.id as number,
			op_id: r.op_id as string,
			entity: r.entity as EntityName,
			uid: r.uid as string,
			op: r.op as OutboxOp,
			base_seq: r.base_seq as number,
			fields: JSON.parse(r.fields as string) as Record<string, unknown>,
			attempts: r.attempts as number,
			created_at: r.created_at as string,
		}));
	}

	async outboxCount(): Promise<number> {
		const row = await getOne<{ n: number }>(this.db, "SELECT COUNT(*) AS n FROM outbox");
		return row?.n ?? 0;
	}

	async outboxOverCap(): Promise<boolean> {
		return (await this.outboxCount()) > OUTBOX_SOFT_CAP;
	}

	/**
	 * Drops the oldest queued mutations until the queue is back at the cap.
	 * Their rows keep the local data but will not reach a server as written,
	 * so this runs only after the person confirmed: the argument must be the
	 * literal `true`, and nothing calls it on its own.
	 */
	async dropOldestOutbox(confirmed: true): Promise<number> {
		if (confirmed !== true) {
			throw new OutboxCapError();
		}
		const over = (await this.outboxCount()) - OUTBOX_SOFT_CAP;
		if (over <= 0) {
			return 0;
		}
		await this.db.run("DELETE FROM outbox WHERE id IN (SELECT id FROM outbox ORDER BY id LIMIT ?)", [over]);
		return over;
	}

	private async mark(
		entity: EntityName,
		uid: string,
		baseSeq: number,
		op: "delete" | "restore",
		at?: string,
	): Promise<void> {
		const table = ENTITIES[entity].table;
		await this.db.transaction(async () => {
			const stamp = at ?? this.now();
			await this.db.run(
				`UPDATE ${table} SET deleted_at = ?, version = version + 1, dirty = 1, updated_local_at = ? WHERE uid = ?`,
				[op === "delete" ? stamp : null, stamp, uid],
			);
			await this.enqueue(entity, uid, op, baseSeq, {}, stamp);
		});
	}

	private async mustGet<E extends EntityName>(entity: E, uid: string, includeDeleted = false): Promise<Row<E>> {
		const row = await this.get(entity, uid, includeDeleted);
		if (row === null) {
			throw new Error(`${entity} ${uid} not found`);
		}
		return row;
	}

	private async enqueue(
		entity: EntityName,
		uid: string,
		op: OutboxOp,
		baseSeq: number,
		fields: Record<string, unknown>,
		stamp: string,
	): Promise<void> {
		await this.db.run(
			"INSERT INTO outbox (op_id, entity, uid, op, base_seq, fields, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			[this.uid(), entity, uid, op, baseSeq, JSON.stringify(fields), stamp],
		);
	}

	/** The `transaction_tags` join rows follow `tag_uids` in the same database transaction. */
	private async syncTags(entity: EntityName, uid: string, values: Record<string, unknown>): Promise<void> {
		if (entity !== "transaction" || !("tag_uids" in values)) {
			return;
		}
		await this.db.run("DELETE FROM transaction_tags WHERE transaction_uid = ?", [uid]);
		const tags = values.tag_uids;
		if (!Array.isArray(tags)) {
			return;
		}
		for (const tag of new Set(tags as string[])) {
			await this.db.run("INSERT INTO transaction_tags (transaction_uid, tag_uid) VALUES (?, ?)", [uid, tag]);
		}
	}
}

function quote(name: string): string {
	return `"${name}"`;
}

export interface IntegrityIssue {
	entity: EntityName;
	uid: string;
	field: string;
	missing: string;
}

/**
 * The local schema has no foreign keys, so this checks that every live row's
 * references name a live row. Run it after writes and after syncs.
 */
export async function checkIntegrity(db: Database): Promise<IntegrityIssue[]> {
	const issues: IntegrityIssue[] = [];

	for (const entity of Object.keys(ENTITIES) as EntityName[]) {
		const meta = ENTITIES[entity];
		for (const [field, parent] of meta.refs) {
			const rows = await db.all<{ uid: string; ref: string }>(
				`SELECT c.uid AS uid, c."${field}" AS ref FROM ${meta.table} c
				 WHERE c.deleted_at IS NULL AND c."${field}" IS NOT NULL
				 AND NOT EXISTS (SELECT 1 FROM ${ENTITIES[parent].table} p WHERE p.uid = c."${field}" AND p.deleted_at IS NULL)`,
			);
			for (const r of rows) {
				issues.push({ entity, uid: r.uid, field, missing: r.ref });
			}
		}
	}

	const txs = await db.all<{ uid: string; tag_uids: string }>(
		"SELECT uid, tag_uids FROM transactions WHERE deleted_at IS NULL AND tag_uids != '[]'",
	);
	for (const t of txs) {
		for (const tag of JSON.parse(t.tag_uids) as string[]) {
			const live = await getOne<{ uid: string }>(db, "SELECT uid FROM tags WHERE uid = ? AND deleted_at IS NULL", [tag]);
			if (live === null) {
				issues.push({ entity: "transaction", uid: t.uid, field: "tag_uids", missing: tag });
			}
		}
	}

	return issues;
}

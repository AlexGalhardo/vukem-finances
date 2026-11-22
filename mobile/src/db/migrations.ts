import { type Database, getOne } from "./database";

export interface Migration {
  version: number;
  statements: readonly string[];
}

/**
 * Columns every synced table carries (sync-data-model.md): the `uid`, the
 * local `version`, the newest server `seq` seen (null until synced), the
 * soft delete mark and the sync bookkeeping. There are no foreign keys: the
 * protocol delivers references out of order, so integrity is checked by
 * `checkIntegrity` after writes and syncs.
 */
const SYNCED = `
	uid TEXT PRIMARY KEY NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	server_seq INTEGER,
	deleted_at TEXT,
	dirty INTEGER NOT NULL DEFAULT 0,
	updated_local_at TEXT NOT NULL`;

export const MIGRATIONS: readonly Migration[] = [
  {
    version: 1,
    statements: [
      "CREATE TABLE meta (key TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL)",
      `CREATE TABLE accounts (${SYNCED},
				name TEXT NOT NULL, kind TEXT NOT NULL, opening_cents INTEGER NOT NULL DEFAULT 0,
				archived_at TEXT, position INTEGER NOT NULL DEFAULT 0)`,
      `CREATE TABLE categories (${SYNCED},
				name TEXT NOT NULL, kind TEXT NOT NULL, parent_uid TEXT, color TEXT NOT NULL DEFAULT '',
				icon TEXT NOT NULL DEFAULT '', archived_at TEXT)`,
      "CREATE UNIQUE INDEX categories_name_live ON categories (name) WHERE deleted_at IS NULL",
      `CREATE TABLE tags (${SYNCED}, name TEXT NOT NULL, color TEXT NOT NULL DEFAULT '')`,
      `CREATE TABLE transactions (${SYNCED},
				account_uid TEXT NOT NULL, category_uid TEXT, kind TEXT NOT NULL, amount_cents INTEGER NOT NULL,
				description TEXT NOT NULL DEFAULT '', occurred_on TEXT NOT NULL, transfer_uid TEXT,
				tag_uids TEXT NOT NULL DEFAULT '[]', statement_ref TEXT, recurrence_uid TEXT, occurrence_on TEXT,
				installment_group_uid TEXT, installment_no INTEGER, installment_total INTEGER)`,
      "CREATE INDEX transactions_account ON transactions (account_uid, occurred_on)",
      "CREATE INDEX transactions_date ON transactions (occurred_on)",
      "CREATE INDEX transactions_transfer ON transactions (transfer_uid) WHERE transfer_uid IS NOT NULL",
      // The natural key of a generated occurrence, soft-deleted rows included, so a deleted one is never recreated.
      `CREATE UNIQUE INDEX transactions_occurrence ON transactions (recurrence_uid, occurrence_on)
				WHERE recurrence_uid IS NOT NULL`,
      `CREATE TABLE transaction_tags (
				transaction_uid TEXT NOT NULL, tag_uid TEXT NOT NULL, PRIMARY KEY (transaction_uid, tag_uid))`,
      "CREATE INDEX transaction_tags_tag ON transaction_tags (tag_uid)",
      `CREATE TABLE budgets (${SYNCED}, category_uid TEXT NOT NULL, limit_cents INTEGER NOT NULL,
				rollover INTEGER NOT NULL DEFAULT 0)`,
      "CREATE UNIQUE INDEX budgets_category_live ON budgets (category_uid) WHERE deleted_at IS NULL",
      `CREATE TABLE credit_cards (${SYNCED},
				account_uid TEXT NOT NULL, limit_cents INTEGER NOT NULL DEFAULT 0, closing_day INTEGER NOT NULL,
				due_day INTEGER NOT NULL, payment_account_uid TEXT, brand TEXT NOT NULL DEFAULT '',
				last4 TEXT NOT NULL DEFAULT '')`,
      `CREATE TABLE recurrences (${SYNCED},
				account_uid TEXT NOT NULL, category_uid TEXT, kind TEXT NOT NULL, amount_cents INTEGER NOT NULL,
				description TEXT NOT NULL DEFAULT '', frequency TEXT NOT NULL, "interval" INTEGER NOT NULL DEFAULT 1,
				start_date TEXT NOT NULL, end_date TEXT, max_count INTEGER, generated_through TEXT,
				active INTEGER NOT NULL DEFAULT 1)`,
      `CREATE TABLE outbox (
				id INTEGER PRIMARY KEY AUTOINCREMENT, op_id TEXT NOT NULL UNIQUE, entity TEXT NOT NULL,
				uid TEXT NOT NULL, op TEXT NOT NULL, base_seq INTEGER NOT NULL DEFAULT 0,
				fields TEXT NOT NULL DEFAULT '{}', attempts INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL)`,
      "CREATE INDEX outbox_row ON outbox (entity, uid)",
    ],
  },
  {
    version: 2,
    // The overall budget has no category (the server allows a null category_id), so the column must be nullable.
    statements: [
      "DROP INDEX budgets_category_live",
      "ALTER TABLE budgets RENAME TO budgets_v1",
      `CREATE TABLE budgets (${SYNCED}, category_uid TEXT, limit_cents INTEGER NOT NULL,
				rollover INTEGER NOT NULL DEFAULT 0)`,
      `INSERT INTO budgets (uid, version, server_seq, deleted_at, dirty, updated_local_at, category_uid, limit_cents, rollover)
				SELECT uid, version, server_seq, deleted_at, dirty, updated_local_at, category_uid, limit_cents, rollover
				FROM budgets_v1`,
      "DROP TABLE budgets_v1",
      "CREATE UNIQUE INDEX budgets_category_live ON budgets (COALESCE(category_uid, '')) WHERE deleted_at IS NULL",
    ],
  },
  {
    version: 3,
    // Epic 20, the sync client: the outbox retry count and the tables of conflicts, problems, resync staging and the data check.
    statements: [
      // How many runs a retryable rejection (unknown_reference, parent_deleted) already survived.
      "ALTER TABLE outbox ADD COLUMN retries INTEGER NOT NULL DEFAULT 0",
      // The values that lost a merge, for the person to look at and re-apply.
      `CREATE TABLE sync_conflicts (
				id INTEGER PRIMARY KEY AUTOINCREMENT, entity TEXT NOT NULL, uid TEXT NOT NULL, field TEXT NOT NULL,
				kind TEXT NOT NULL, lost_value TEXT, applied_value TEXT, created_at TEXT NOT NULL, dismissed_at TEXT)`,
      // Mutations the server refused for good: the person discards them or edits the row again.
      `CREATE TABLE sync_problems (
				id INTEGER PRIMARY KEY AUTOINCREMENT, entity TEXT NOT NULL, uid TEXT NOT NULL, op_id TEXT NOT NULL,
				op TEXT NOT NULL, code TEXT NOT NULL, message TEXT NOT NULL DEFAULT '', field TEXT NOT NULL DEFAULT '',
				fields TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, dismissed_at TEXT)`,
      // The snapshot of a resync, held here until every entity is downloaded.
      "CREATE TABLE sync_staging (entity TEXT NOT NULL, uid TEXT NOT NULL, row TEXT NOT NULL, PRIMARY KEY (entity, uid))",
      // References that still name nothing after the integrity pass: the data check list. Never deleted by the app.
      `CREATE TABLE sync_ref_missing (
				entity TEXT NOT NULL, uid TEXT NOT NULL, field TEXT NOT NULL, missing TEXT NOT NULL,
				PRIMARY KEY (entity, uid, field))`,
    ],
  },
];

export const LATEST_VERSION = MIGRATIONS[MIGRATIONS.length - 1]?.version ?? 0;

/** The schema version held in `meta`, 0 for a database that was never migrated. */
export async function schemaVersion(db: Database): Promise<number> {
  const meta = await getOne<{ name: string }>(
    db,
    "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'meta'",
  );
  if (meta === null) {
    return 0;
  }
  const row = await getOne<{ value: string }>(db, "SELECT value FROM meta WHERE key = 'schema_version'");
  return row === null ? 0 : Number(row.value);
}

/**
 * Applies, in order, every migration newer than the stored version. Each one
 * runs in its own transaction with its version bump, so a failure leaves the
 * database at the last good version.
 */
export async function migrate(db: Database, migrations: readonly Migration[] = MIGRATIONS): Promise<number> {
  let current = await schemaVersion(db);
  const ordered = [...migrations].sort((a, b) => a.version - b.version);

  for (const m of ordered) {
    if (m.version <= current) {
      continue;
    }
    await db.transaction(async (db) => {
      for (const sql of m.statements) {
        await db.exec(sql);
      }
      await db.run(
        "INSERT INTO meta (key, value) VALUES ('schema_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
        [String(m.version)],
      );
    });
    current = m.version;
  }

  return current;
}

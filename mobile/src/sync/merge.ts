import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import { type RawRow, toRow } from "../db/repository";
import { newUid } from "../domain/ids";
import { rekeyRow } from "./apply";
import { applyStaged, downloadSnapshot } from "./resync";
import type { Transport } from "./types";

export interface MergeDeps {
  db: Database;
  transport: Transport;
  now: () => string;
  /** Makes the op_id of a queued mutation. */
  uid?: () => string;
}

export const FIRST_SYNC_KEY = "sync.first_sync_done";

/** Parents first. */
const ORDER: readonly EntityName[] = [
  "account",
  "category",
  "tag",
  "credit_card",
  "recurrence",
  "budget",
  "transaction",
];

/** The entities whose name (and kind, where they have one) says that two rows are the same thing. */
const BY_NAME: readonly { entity: EntityName; fields: readonly string[] }[] = [
  { entity: "account", fields: ["name", "kind"] },
  { entity: "category", fields: ["name", "kind"] },
  { entity: "tag", fields: ["name"] },
];

async function markFirstSyncDone(db: Database): Promise<void> {
  await db.run(
    `INSERT INTO meta (key, value) VALUES ('${FIRST_SYNC_KEY}', '1') ON CONFLICT(key) DO UPDATE SET value = '1'`,
  );
}

/**
 * Queues every live row that never reached a server as one full create. What was
 * already queued for such a row is replaced: a partial edit cannot create a row.
 */
export async function enqueueUnsyncedAsCreates(db: Database, now: string, uid: () => string): Promise<number> {
  let queued = 0;
  for (const entity of ORDER) {
    const meta = ENTITIES[entity];
    const rows = await db.all<RawRow>(`SELECT * FROM ${meta.table} WHERE server_seq IS NULL`);
    for (const raw of rows) {
      await db.run("DELETE FROM outbox WHERE entity = ? AND uid = ?", [entity, raw.uid as string]);
      if (raw.deleted_at !== null) {
        continue;
      }
      const row = toRow(entity, raw) as unknown as Record<string, unknown>;
      const fields: Record<string, unknown> = {};
      for (const name of Object.keys(meta.fields)) {
        fields[name] = row[name] ?? null;
      }
      await db.run(
        "INSERT INTO outbox (op_id, entity, uid, op, base_seq, fields, created_at) VALUES (?, ?, ?, 'upsert', 0, ?, ?)",
        [uid(), entity, raw.uid as string, JSON.stringify(fields), now],
      );
      await db.run(`UPDATE ${meta.table} SET dirty = 1 WHERE uid = ?`, [raw.uid as string]);
      queued += 1;
    }
  }
  return queued;
}

export interface MergeResult {
  /** Local rows that took the uid of the server row with the same name. */
  merged: number;
  queued: number;
}

/**
 * "Merge (recommended)": the server's snapshot is downloaded, a local account,
 * category or tag whose name and kind equal a server row's adopts that row's uid
 * (the references to it follow), every other local row is queued as a create,
 * and the snapshot is written beside them. Nothing is deleted on either side.
 */
export async function mergeWithServer(deps: MergeDeps): Promise<MergeResult> {
  const { db, transport } = deps;
  const cursor = await downloadSnapshot(db, transport);
  let merged = 0;
  let queued = 0;

  await db.transaction(async () => {
    for (const { entity, fields } of BY_NAME) {
      const staged = (
        await db.all<{ uid: string; row: string }>("SELECT uid, row FROM sync_staging WHERE entity = ?", [entity])
      ).map((s) => ({ uid: s.uid, row: JSON.parse(s.row) as Record<string, unknown> }));
      const key = (row: Record<string, unknown>): string => JSON.stringify(fields.map((f) => row[f] ?? null));
      const byKey = new Map(staged.map((s) => [key(s.row), s.uid]));

      const locals = await db.all<RawRow>(
        `SELECT * FROM ${ENTITIES[entity].table} WHERE server_seq IS NULL AND deleted_at IS NULL ORDER BY uid`,
      );
      for (const local of locals) {
        const serverUid = byKey.get(key(local as Record<string, unknown>));
        if (serverUid !== undefined && serverUid !== local.uid) {
          await rekeyRow(db, entity, local.uid as string, serverUid, true);
          // It is the server's row now: it needs no create, and the snapshot fills in the rest.
          await db.run(`UPDATE ${ENTITIES[entity].table} SET server_seq = ?, dirty = 0 WHERE uid = ?`, [
            cursor,
            serverUid,
          ]);
          merged += 1;
        }
      }
    }
    queued = await enqueueUnsyncedAsCreates(db, deps.now(), deps.uid ?? newUid);
    await applyStaged(db, cursor, deps.now);
    await markFirstSyncDone(db);
  });
  return { merged, queued };
}

/** What "Replace this phone's data" would delete, for the confirmation that says so. */
export async function describeReplace(db: Database): Promise<{ rows: number; pending: number }> {
  let rows = 0;
  for (const meta of Object.values(ENTITIES)) {
    rows +=
      (await db.all<{ n: number }>(`SELECT COUNT(*) AS n FROM ${meta.table} WHERE deleted_at IS NULL`))[0]?.n ?? 0;
  }
  const pending = (await db.all<{ n: number }>("SELECT COUNT(*) AS n FROM outbox"))[0]?.n ?? 0;
  return { rows, pending };
}

/**
 * "Replace this phone's data with the server's". The argument must be the
 * literal `true`: the screen passes it only after the person confirmed the
 * warning and was offered an export. The snapshot is downloaded first, so a
 * failed download deletes nothing.
 */
export async function replaceWithServer(deps: MergeDeps, confirmed: true): Promise<void> {
  if (confirmed !== true) {
    throw new Error("replacing the local data needs a confirmation");
  }
  const { db, transport } = deps;
  const cursor = await downloadSnapshot(db, transport);
  await db.transaction(async () => {
    await db.run("DELETE FROM outbox");
    await db.run("DELETE FROM sync_problems");
    await db.run("DELETE FROM sync_conflicts");
    await db.run("DELETE FROM sync_ref_missing");
    await applyStaged(db, cursor, deps.now, true);
    await markFirstSyncDone(db);
  });
}

/** An empty phone connects without a prompt: the first pull is the whole merge. */
export async function hasLocalData(db: Database): Promise<boolean> {
  const { rows } = await describeReplace(db);
  return rows > 0;
}

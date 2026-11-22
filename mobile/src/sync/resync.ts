import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import { applyChange } from "./apply";
import { setCursor } from "./state";
import type { PullChange, Transport } from "./types";

export const SNAPSHOT_LIMIT = 500;

/** Parents first, so most references resolve while the rows are written. */
const ORDER: readonly EntityName[] = [
  "account",
  "category",
  "tag",
  "credit_card",
  "recurrence",
  "budget",
  "transaction",
];

/** Downloads every entity into the staging table. Nothing else changes, so a failure half way costs nothing. Returns the snapshot cursor. */
export async function downloadSnapshot(db: Database, transport: Transport): Promise<number> {
  await db.run("DELETE FROM sync_staging");

  let snapshotId = "";
  let cursor = 0;
  for (const entity of ORDER) {
    let after = "";
    for (;;) {
      const page = await transport.snapshot({ entity, after, snapshot_id: snapshotId, limit: SNAPSHOT_LIMIT });
      snapshotId = page.snapshot_id;
      cursor = page.snapshot_cursor;
      await db.transaction(async (db) => {
        for (const row of page.rows) {
          await db.run("INSERT OR REPLACE INTO sync_staging (entity, uid, row) VALUES (?, ?, ?)", [
            entity,
            String(row.uid),
            JSON.stringify(row),
          ]);
        }
      });
      if (!page.has_more || page.next_after === "") break;
      after = page.next_after;
    }
  }
  return cursor;
}

/**
 * Writes the staged snapshot: every row that is not dirty is replaced (or, with
 * `wipe`, every row), dirty rows and the outbox are kept, the cursor becomes the
 * snapshot's. Call it inside a transaction of its own or let it open one.
 */
export async function applyStaged(db: Database, cursor: number, now: () => string, wipe = false): Promise<void> {
  await db.transaction(async (db) => {
    for (const entity of ORDER) {
      await db.run(`DELETE FROM ${ENTITIES[entity].table}${wipe ? "" : " WHERE dirty = 0"}`);
    }
    await db.run("DELETE FROM transaction_tags WHERE transaction_uid NOT IN (SELECT uid FROM transactions)");

    const stamp = now();
    const stats = { upserts: 0, deletes: 0, conflicts: 0 };
    const staged = await db.all<{ entity: string; uid: string; row: string }>(
      "SELECT entity, uid, row FROM sync_staging",
    );
    const rank = (e: string): number => ORDER.indexOf(e as EntityName);
    staged.sort((a, b) => rank(a.entity) - rank(b.entity) || (a.uid < b.uid ? -1 : 1));
    for (const s of staged) {
      const row = JSON.parse(s.row) as Record<string, unknown>;
      const change: PullChange = {
        seq: cursor,
        entity: s.entity,
        uid: s.uid,
        op: "upsert",
        version: Number(row.version ?? 1),
        row,
      };
      await applyChange(db, change, stamp, stats);
    }
    await db.run("DELETE FROM sync_staging");
    await setCursor(db, cursor);
  });
}

/**
 * The answer to `410 resync_required`. Every entity is downloaded into the
 * staging table first, so a failure while downloading changes nothing. Then, in
 * ONE transaction, every row that is not dirty is replaced by the snapshot,
 * dirty rows and the outbox are kept, and the cursor becomes the snapshot's.
 * The caller then runs a normal pull from that cursor.
 */
export async function resync(db: Database, transport: Transport, now: () => string): Promise<number> {
  const cursor = await downloadSnapshot(db, transport);
  await applyStaged(db, cursor, now);
  return cursor;
}

import type { Database } from "../db/database";
import { ENTITIES } from "../db/entities";

/** Deletes every synced row, the queue and the sync bookkeeping of this phone, in one transaction. */
export async function wipeLocalData(db: Database): Promise<void> {
  await db.transaction(async (db) => {
    for (const meta of Object.values(ENTITIES)) {
      await db.run(`DELETE FROM ${meta.table}`);
    }
    for (const table of [
      "transaction_tags",
      "outbox",
      "sync_problems",
      "sync_conflicts",
      "sync_ref_missing",
      "sync_staging",
    ]) {
      await db.run(`DELETE FROM ${table}`);
    }
    await db.run("DELETE FROM meta WHERE key = 'sync.cursor'");
  });
}

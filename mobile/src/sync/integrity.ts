import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import { PULL_LIMIT, pullUntilDone } from "./apply";
import { getCursor } from "./state";
import type { Transport } from "./types";

export interface MissingRef {
  entity: EntityName;
  uid: string;
  field: string;
  missing: string;
}

/**
 * Live rows that name a uid no row holds at all (a deleted row still counts as
 * held: its tombstone is here). The local schema has no foreign keys because a
 * pull delivers children before parents, so this runs when the pull is over.
 */
export async function findMissingRefs(db: Database): Promise<MissingRef[]> {
  const out: MissingRef[] = [];
  const held = async (entity: EntityName, column: string, table: string, ref: string): Promise<void> => {
    const rows = await db.all<{ uid: string; ref: string }>(
      `SELECT c.uid AS uid, c."${column}" AS ref FROM ${table} c
			 WHERE c.deleted_at IS NULL AND c."${column}" IS NOT NULL AND c."${column}" != ''
			 AND NOT EXISTS (SELECT 1 FROM ${ref} p WHERE p.uid = c."${column}")`,
    );
    for (const r of rows) {
      out.push({ entity, uid: r.uid, field: column, missing: r.ref });
    }
  };

  for (const entity of Object.keys(ENTITIES) as EntityName[]) {
    for (const [field, parent] of ENTITIES[entity].refs) {
      await held(entity, field, ENTITIES[entity].table, ENTITIES[parent].table);
    }
  }
  await held("transaction", "recurrence_uid", "transactions", "recurrences");

  const txs = await db.all<{ uid: string; tag_uid: string }>(
    `SELECT tt.transaction_uid AS uid, tt.tag_uid AS tag_uid FROM transaction_tags tt
		 JOIN transactions t ON t.uid = tt.transaction_uid AND t.deleted_at IS NULL
		 WHERE NOT EXISTS (SELECT 1 FROM tags g WHERE g.uid = tt.tag_uid)`,
  );
  for (const t of txs) {
    out.push({ entity: "transaction", uid: t.uid, field: "tag_uids", missing: t.tag_uid });
  }
  return out;
}

async function store(db: Database, issues: readonly MissingRef[]): Promise<void> {
  await db.transaction(async () => {
    await db.run("DELETE FROM sync_ref_missing");
    for (const i of issues) {
      await db.run("INSERT OR REPLACE INTO sync_ref_missing (entity, uid, field, missing) VALUES (?, ?, ?, ?)", [
        i.entity,
        i.uid,
        i.field,
        i.missing,
      ]);
    }
  });
}

/** The data check list: references that still name nothing after the integrity pass. */
export async function listMissingRefs(db: Database): Promise<MissingRef[]> {
  return db.all<MissingRef>("SELECT entity, uid, field, missing FROM sync_ref_missing ORDER BY entity, uid, field");
}

/**
 * After a pull, looks for references to rows this device does not hold. If there
 * are any, it pulls once more from the cursor minus one page (the saved cursor
 * does not move back). What is still missing afterwards is listed for the
 * person and never deleted by the app.
 */
export async function integrityPass(
  db: Database,
  transport: Transport,
  now: () => string,
  limit = PULL_LIMIT,
): Promise<{ missing: number; repulled: boolean }> {
  let issues = await findMissingRefs(db);
  let repulled = false;
  if (issues.length > 0) {
    repulled = true;
    await pullUntilDone(db, transport, now, Math.max(0, (await getCursor(db)) - limit), limit);
    issues = await findMissingRefs(db);
  }
  await store(db, issues);
  return { missing: issues.length, repulled };
}

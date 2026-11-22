import { type Database, getOne, type SqlValue } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import { encode, type RawRow } from "../db/repository";
import { pendingFor } from "./outbox";
import { setCursor } from "./state";
import type { PullChange, PullPage } from "./types";

export interface ApplyStats {
  upserts: number;
  deletes: number;
  conflicts: number;
}

export function isEntity(name: string): name is EntityName {
  return Object.hasOwn(ENTITIES, name);
}

function quote(name: string): string {
  return `"${name}"`;
}

export async function recordConflict(
  db: Database,
  c: { entity: string; uid: string; field: string; kind: string; lost: unknown; applied: unknown },
  now: string,
): Promise<void> {
  await db.run(
    `INSERT INTO sync_conflicts (entity, uid, field, kind, lost_value, applied_value, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
    [c.entity, c.uid, c.field, c.kind, JSON.stringify(c.lost ?? null), JSON.stringify(c.applied ?? null), now],
  );
}

/** The `transaction_tags` join rows follow `tag_uids`. */
export async function syncTagRows(db: Database, uid: string, tagUidsJson: SqlValue): Promise<void> {
  await db.run("DELETE FROM transaction_tags WHERE transaction_uid = ?", [uid]);
  const tags = typeof tagUidsJson === "string" ? (JSON.parse(tagUidsJson) as unknown) : [];
  if (Array.isArray(tags)) {
    for (const tag of new Set(tags as string[])) {
      await db.run("INSERT INTO transaction_tags (transaction_uid, tag_uid) VALUES (?, ?)", [uid, tag]);
    }
  }
}

/**
 * Points every local reference to `from` at `to`, in the rows and in the
 * queued mutations, so a row that took the server's uid keeps its history.
 * The caller runs it inside a transaction.
 */
export async function repointReferences(db: Database, entity: EntityName, from: string, to: string): Promise<void> {
  const refFields: { table: string; entity: EntityName; field: string }[] = [];
  for (const [name, meta] of Object.entries(ENTITIES) as [EntityName, (typeof ENTITIES)[EntityName]][]) {
    for (const [field, parent] of meta.refs) {
      if (parent === entity) {
        refFields.push({ table: meta.table, entity: name, field });
      }
    }
  }
  if (entity === "recurrence") {
    refFields.push({ table: ENTITIES.transaction.table, entity: "transaction", field: "recurrence_uid" });
  }

  for (const r of refFields) {
    await db.run(`UPDATE ${r.table} SET ${quote(r.field)} = ? WHERE ${quote(r.field)} = ?`, [to, from]);
  }

  if (entity === "tag") {
    const txs = await db.all<{ uid: string; tag_uids: string }>(
      "SELECT uid, tag_uids FROM transactions WHERE tag_uids LIKE ?",
      [`%${from}%`],
    );
    for (const t of txs) {
      const next = (JSON.parse(t.tag_uids) as string[]).map((u) => (u === from ? to : u));
      await db.run("UPDATE transactions SET tag_uids = ? WHERE uid = ?", [JSON.stringify(next), t.uid]);
      await syncTagRows(db, t.uid, JSON.stringify(next));
    }
  }

  const queued = await db.all<RawRow>("SELECT id, entity, fields FROM outbox WHERE fields LIKE ?", [`%${from}%`]);
  for (const q of queued) {
    const fields = JSON.parse(q.fields as string) as Record<string, unknown>;
    let changed = false;
    for (const [name, value] of Object.entries(fields)) {
      if (value === from) {
        const meta = isEntity(q.entity as string) ? ENTITIES[q.entity as EntityName] : null;
        const isRef = meta?.refs.some(([f, parent]) => f === name && parent === entity) ?? false;
        if (isRef || (entity === "recurrence" && name === "recurrence_uid")) {
          fields[name] = to;
          changed = true;
        }
      } else if (entity === "tag" && name === "tag_uids" && Array.isArray(value) && value.includes(from)) {
        fields[name] = value.map((u: unknown) => (u === from ? to : u));
        changed = true;
      }
    }
    if (changed) {
      await db.run("UPDATE outbox SET fields = ? WHERE id = ?", [JSON.stringify(fields), q.id as number]);
    }
  }
}

/**
 * Gives the local row `from` the uid `to`: the references follow, the queued
 * mutations of the row are dropped when `dropQueued` (the server already has
 * the row) or retargeted. When a row named `to` already exists the local one
 * is removed instead, since the server's copy wins.
 */
export async function rekeyRow(
  db: Database,
  entity: EntityName,
  from: string,
  to: string,
  dropQueued: boolean,
): Promise<void> {
  const table = ENTITIES[entity].table;
  await repointReferences(db, entity, from, to);
  const taken = await getOne<{ uid: string }>(db, `SELECT uid FROM ${table} WHERE uid = ?`, [to]);
  if (dropQueued) {
    await db.run("DELETE FROM outbox WHERE entity = ? AND uid = ?", [entity, from]);
  } else {
    await db.run("UPDATE outbox SET uid = ? WHERE entity = ? AND uid = ?", [to, entity, from]);
  }
  if (entity === "transaction") {
    await db.run("DELETE FROM transaction_tags WHERE transaction_uid = ?", [from]);
  }
  if (taken === null) {
    await db.run(`UPDATE ${table} SET uid = ? WHERE uid = ?`, [to, from]);
    if (entity === "transaction") {
      const row = await getOne<{ tag_uids: string }>(db, "SELECT tag_uids FROM transactions WHERE uid = ?", [to]);
      await syncTagRows(db, to, row?.tag_uids ?? "[]");
    }
  } else {
    await db.run(`DELETE FROM ${table} WHERE uid = ?`, [from]);
  }
}

type Encoded = Record<string, SqlValue>;

/**
 * The server is the arbiter of the unique keys (a category name, the budget of
 * a category, an occurrence of a rule), but a pull may deliver rows in an order
 * that makes two of them meet for a moment. A local row that never reached the
 * server is the same thing as the server's: it adopts the server's uid. A synced
 * one is parked under a suffix until its own change arrives.
 */
async function clearUniqueKey(db: Database, entity: EntityName, uid: string, values: Encoded): Promise<void> {
  let where: string;
  let params: SqlValue[];
  let parkColumn: string;
  switch (entity) {
    case "category":
      if (values.deleted_at !== null && values.deleted_at !== undefined) return;
      where = "name = ? AND deleted_at IS NULL";
      params = [values.name ?? null];
      parkColumn = "name";
      break;
    case "budget":
      where = "COALESCE(category_uid, '') = ? AND deleted_at IS NULL";
      params = [(values.category_uid as string | null) ?? ""];
      parkColumn = "category_uid";
      break;
    case "transaction":
      if (values.recurrence_uid === null || values.recurrence_uid === undefined) return;
      where = "recurrence_uid = ? AND occurrence_on = ?";
      params = [values.recurrence_uid, values.occurrence_on ?? null];
      parkColumn = "occurrence_on";
      break;
    default:
      return;
  }
  const clashes = await db.all<{ uid: string; server_seq: number | null }>(
    `SELECT uid, server_seq FROM ${ENTITIES[entity].table} WHERE ${where} AND uid != ?`,
    [...params, uid],
  );
  for (const c of clashes) {
    if (c.server_seq === null) {
      await rekeyRow(db, entity, c.uid, uid, true);
    } else {
      await db.run(`UPDATE ${ENTITIES[entity].table} SET ${quote(parkColumn)} = ? WHERE uid = ?`, [
        `~parked~${c.uid}`,
        c.uid,
      ]);
    }
  }
}

function serverValues(entity: EntityName, row: Record<string, unknown>): Encoded {
  const out: Encoded = {};
  for (const [name, kind] of Object.entries(ENTITIES[entity].fields)) {
    out[name] = encode(kind, row[name]);
  }
  return out;
}

async function applyUpsert(db: Database, change: PullChange, entity: EntityName, stats: ApplyStats): Promise<void> {
  const meta = ENTITIES[entity];
  const row = change.row ?? {};
  const server = serverValues(entity, row);
  const existing = await getOne<RawRow>(db, `SELECT * FROM ${meta.table} WHERE uid = ?`, [change.uid]);
  const pending = existing === null ? null : await pendingFor(db, entity, change.uid);
  const names = Object.keys(meta.fields);

  const kept = existing !== null && pending !== null && pending.any;
  const values: Encoded = {};
  for (const name of names) {
    values[name] = kept && pending.fields.has(name) ? ((existing[name] as SqlValue) ?? null) : (server[name] ?? null);
  }
  let deletedAt: string | null = null;
  if (kept && pending.lifecycle === "delete") {
    deletedAt = (existing.deleted_at as string | null) ?? null;
  }

  await clearUniqueKey(db, entity, change.uid, { ...values, deleted_at: deletedAt });

  const stamp = (existing?.updated_local_at as string | undefined) ?? new Date(0).toISOString();
  const columns = ["uid", "version", "server_seq", "deleted_at", "dirty", "updated_local_at", ...names.map(quote)];
  const params: SqlValue[] = [
    change.uid,
    change.version,
    change.seq,
    deletedAt,
    kept ? 1 : 0,
    stamp,
    ...names.map((n) => values[n] ?? null),
  ];
  const updates = columns
    .slice(1)
    .map((c) => `${c} = excluded.${c}`)
    .join(", ");
  await db.run(
    `INSERT INTO ${meta.table} (${columns.join(", ")}) VALUES (${columns.map(() => "?").join(", ")})
		 ON CONFLICT(uid) DO UPDATE SET ${updates}`,
    params,
  );
  if (entity === "transaction") {
    await syncTagRows(db, change.uid, values.tag_uids ?? "[]");
  }
  stats.upserts += 1;
}

async function applyDelete(
  db: Database,
  change: PullChange,
  entity: EntityName,
  now: string,
  stats: ApplyStats,
): Promise<void> {
  const meta = ENTITIES[entity];
  const existing = await getOne<RawRow>(db, `SELECT * FROM ${meta.table} WHERE uid = ?`, [change.uid]);
  if (existing === null) {
    return;
  }
  const pending = await pendingFor(db, entity, change.uid);
  if (pending.lifecycle === "restore") {
    // The person brought the row back after the server deleted it: the push decides.
    return;
  }
  if (pending.any) {
    // Rule 4 of the conflict policy: a delete wins over a concurrent edit; the edit is dropped and recorded.
    const queued = await db.all<RawRow>("SELECT fields FROM outbox WHERE entity = ? AND uid = ? ORDER BY id", [
      entity,
      change.uid,
    ]);
    const lost = new Map<string, unknown>();
    for (const q of queued) {
      for (const [field, value] of Object.entries(JSON.parse(q.fields as string) as Record<string, unknown>)) {
        lost.set(field, value);
      }
    }
    for (const [field, value] of lost) {
      await recordConflict(
        db,
        { entity, uid: change.uid, field, kind: "edited_deleted_row", lost: value, applied: null },
        now,
      );
      stats.conflicts += 1;
    }
    await db.run("DELETE FROM outbox WHERE entity = ? AND uid = ?", [entity, change.uid]);
  }
  await db.run(
    `UPDATE ${meta.table} SET deleted_at = COALESCE(deleted_at, ?), version = ?, server_seq = ?, dirty = 0 WHERE uid = ?`,
    [now, change.version, change.seq, change.uid],
  );
  stats.deletes += 1;
}

/** Applies one change. A change of an entity this client does not know (a newer server) is ignored. */
export async function applyChange(db: Database, change: PullChange, now: string, stats: ApplyStats): Promise<void> {
  if (!isEntity(change.entity)) {
    return;
  }
  if (change.op === "delete" || change.row === null) {
    await applyDelete(db, change, change.entity, now, stats);
  } else {
    await applyUpsert(db, change, change.entity, stats);
  }
}

/**
 * Applies a page and saves its cursor in ONE SQLite transaction. A crash
 * between the two cannot happen, and a page that runs twice (a crash before the
 * commit, or the server sending a row again) ends in the same state.
 */
export async function applyPage(db: Database, page: PullPage, now: string): Promise<ApplyStats> {
  const stats: ApplyStats = { upserts: 0, deletes: 0, conflicts: 0 };
  await db.transaction(async () => {
    for (const change of page.changes) {
      await applyChange(db, change, now, stats);
    }
    await setCursor(db, page.next_cursor);
  });
  return stats;
}

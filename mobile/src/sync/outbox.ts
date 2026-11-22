import type { Database } from "../db/database";
import type { EntityName } from "../db/entities";
import type { OutboxEntry, OutboxOp, RawRow } from "../db/repository";
import { pausedEntities } from "./state";
import type { Mutation } from "./types";

/** The protocol takes at most 200 mutations a push. */
export const BATCH_SIZE = 200;

/** A retryable rejection is tried again after the next pull this many runs, then it becomes a problem. */
export const MAX_RETRIES = 3;

export function toEntry(r: RawRow): OutboxEntry {
  return {
    id: r.id as number,
    op_id: r.op_id as string,
    entity: r.entity as EntityName,
    uid: r.uid as string,
    op: r.op as OutboxOp,
    base_seq: r.base_seq as number,
    fields: JSON.parse(r.fields as string) as Record<string, unknown>,
    attempts: r.attempts as number,
    created_at: r.created_at as string,
  };
}

export function toMutation(e: OutboxEntry): Mutation {
  return { op_id: e.op_id, entity: e.entity, uid: e.uid, op: e.op, base_seq: e.base_seq, fields: e.fields };
}

export interface Batch {
  entries: OutboxEntry[];
  mutations: Mutation[];
}

/**
 * The next mutations to send, in creation order, up to 200, skipping the
 * entities a quota refusal paused. The `op_id` of an entry is fixed when the
 * write is queued, so a batch rebuilt after a timeout carries the same ids and
 * the server returns the stored results instead of writing twice.
 */
export async function nextBatch(
  db: Database,
  limit = BATCH_SIZE,
  exclude: ReadonlySet<number> = new Set(),
): Promise<Batch> {
  const paused = [...(await pausedEntities(db)).keys()];
  const skipped = [...exclude];
  const where: string[] = [];
  if (paused.length > 0) where.push(`entity NOT IN (${paused.map(() => "?").join(", ")})`);
  if (skipped.length > 0) where.push(`id NOT IN (${skipped.map(() => "?").join(", ")})`);
  const rows = await db.all<RawRow>(
    `SELECT * FROM outbox${where.length === 0 ? "" : ` WHERE ${where.join(" AND ")}`} ORDER BY id LIMIT ?`,
    [...paused, ...skipped, limit],
  );
  const entries = rows.map(toEntry);
  return { entries, mutations: entries.map(toMutation) };
}

export async function markAttempt(db: Database, ids: readonly number[]): Promise<void> {
  for (const id of ids) {
    await db.run("UPDATE outbox SET attempts = attempts + 1 WHERE id = ?", [id]);
  }
}

/** The fields an edit still waits to send for a row, and whether a delete or a restore waits too. */
export interface Pending {
  fields: Set<string>;
  /** The newest queued delete or restore: the state the person wants the row to end in. */
  lifecycle: "delete" | "restore" | null;
  any: boolean;
}

export async function pendingFor(db: Database, entity: string, uid: string): Promise<Pending> {
  const rows = await db.all<RawRow>("SELECT op, fields FROM outbox WHERE entity = ? AND uid = ? ORDER BY id", [
    entity,
    uid,
  ]);
  const pending: Pending = { fields: new Set(), lifecycle: null, any: rows.length > 0 };
  for (const r of rows) {
    if (r.op === "delete" || r.op === "restore") {
      pending.lifecycle = r.op;
    }
    for (const name of Object.keys(JSON.parse(r.fields as string) as object)) {
      pending.fields.add(name);
    }
  }
  return pending;
}

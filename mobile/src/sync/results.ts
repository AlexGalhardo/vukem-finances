import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import type { OutboxEntry } from "../db/repository";
import { type ApplyStats, applyChange, recordConflict, rekeyRow } from "./apply";
import { type Batch, MAX_RETRIES } from "./outbox";
import { pauseEntity } from "./state";
import type { PushResult } from "./types";

/** Rejections that may heal after the next pull: the parent row may simply not be here yet. */
const RETRYABLE = new Set(["unknown_reference", "parent_deleted"]);

export interface ResultSummary {
  applied: number;
  merged: number;
  duplicates: number;
  rekeyed: number;
  conflicts: number;
  /** Left in the outbox to try again after the next pull. */
  retryLater: number;
  /** Removed from the outbox and listed as a problem. */
  problems: number;
  /** Entities whose pushes the plan quota paused, with the server's message. */
  paused: Map<string, string>;
}

function emptySummary(): ResultSummary {
  return {
    applied: 0,
    merged: 0,
    duplicates: 0,
    rekeyed: 0,
    conflicts: 0,
    retryLater: 0,
    problems: 0,
    paused: new Map(),
  };
}

async function dropEntry(db: Database, entry: OutboxEntry): Promise<void> {
  await db.run("DELETE FROM outbox WHERE id = ?", [entry.id]);
}

async function recordProblem(db: Database, entry: OutboxEntry, result: PushResult, now: string): Promise<void> {
  await db.run(
    `INSERT INTO sync_problems (entity, uid, op_id, op, code, message, field, fields, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    [
      entry.entity,
      entry.uid,
      entry.op_id,
      entry.op,
      result.error?.code ?? "rejected",
      result.error?.message ?? "",
      result.error?.field ?? "",
      JSON.stringify(entry.fields),
      now,
    ],
  );
  await dropEntry(db, entry);
  // The row no longer waits to be sent. The problem keeps what was lost, so "try again" can send it once more.
  const pending = await db.all("SELECT id FROM outbox WHERE entity = ? AND uid = ? LIMIT 1", [entry.entity, entry.uid]);
  if (pending.length === 0) {
    await db.run(`UPDATE ${ENTITIES[entry.entity].table} SET dirty = 0 WHERE uid = ?`, [entry.uid]);
  }
}

async function localTransferUid(db: Database, uid: string): Promise<string | null> {
  const rows = await db.all<{ transfer_uid: string | null }>("SELECT transfer_uid FROM transactions WHERE uid = ?", [
    uid,
  ]);
  return rows[0]?.transfer_uid ?? null;
}

/**
 * Reads what the server said about each mutation of a batch and updates the
 * rows, the outbox, the conflicts and the problems, all in ONE SQLite
 * transaction: a crash leaves the whole batch unhandled and the retry (same
 * `op_id`s) gets the stored results again.
 */
export async function handleResults(
  db: Database,
  batch: Batch,
  results: readonly PushResult[],
  now: string,
): Promise<ResultSummary> {
  if (results.length !== batch.entries.length) {
    throw new Error("the server answered a different number of results than mutations");
  }
  const summary = emptySummary();
  const stats: ApplyStats = { upserts: 0, deletes: 0, conflicts: 0 };

  await db.transaction(async () => {
    const incomplete = new Map<string, number>();
    for (const [i, entry] of batch.entries.entries()) {
      const result = results[i];
      if (
        result?.status === "rejected" &&
        result.error?.code === "transfer_incomplete" &&
        entry.entity === "transaction"
      ) {
        const transfer = await localTransferUid(db, entry.uid);
        if (transfer !== null) incomplete.set(transfer, (incomplete.get(transfer) ?? 0) + 1);
      }
    }

    for (const [i, entry] of batch.entries.entries()) {
      const result = results[i];
      if (result === undefined || result.op_id !== entry.op_id) {
        throw new Error("the server answered the mutations in another order");
      }
      switch (result.status) {
        case "applied":
        case "merged":
          await applyAccepted(db, entry, result, now, stats, summary);
          break;
        case "duplicate":
          await applyDuplicate(db, entry, result, now, stats, summary);
          break;
        default:
          await applyRejected(db, entry, result, now, summary, incomplete);
      }
    }
  });

  summary.conflicts += stats.conflicts;
  return summary;
}

async function applyAccepted(
  db: Database,
  entry: OutboxEntry,
  result: PushResult,
  now: string,
  stats: ApplyStats,
  summary: ResultSummary,
): Promise<void> {
  await dropEntry(db, entry);
  // Later queued edits of the row were written against an older base: this push is part of their history now.
  await db.run("UPDATE outbox SET base_seq = MAX(base_seq, ?) WHERE entity = ? AND uid = ?", [
    result.seq,
    entry.entity,
    entry.uid,
  ]);
  if (entry.op === "delete") {
    await applyChange(
      db,
      { seq: result.seq, entity: entry.entity, uid: entry.uid, op: "delete", version: result.version, row: null },
      now,
      stats,
    );
  } else if (result.row !== null) {
    await applyChange(
      db,
      { seq: result.seq, entity: entry.entity, uid: entry.uid, op: "upsert", version: result.version, row: result.row },
      now,
      stats,
    );
  }
  for (const c of result.conflicts) {
    await recordConflict(
      db,
      {
        entity: entry.entity,
        uid: entry.uid,
        field: c.field,
        kind: c.kind,
        lost: c.lost_value,
        applied: c.applied_value,
      },
      now,
    );
    summary.conflicts += 1;
  }
  summary[result.status === "merged" ? "merged" : "applied"] += 1;
}

async function applyDuplicate(
  db: Database,
  entry: OutboxEntry,
  result: PushResult,
  now: string,
  stats: ApplyStats,
  summary: ResultSummary,
): Promise<void> {
  summary.duplicates += 1;
  if (result.replaced_by_uid !== "" && result.replaced_by_uid !== entry.uid) {
    await rekeyRow(db, entry.entity, entry.uid, result.replaced_by_uid, true);
    summary.rekeyed += 1;
    return;
  }
  await dropEntry(db, entry);
  if (result.row !== null) {
    await applyChange(
      db,
      { seq: result.seq, entity: entry.entity, uid: entry.uid, op: "upsert", version: result.version, row: result.row },
      now,
      stats,
    );
  } else {
    await db.run(
      `UPDATE ${ENTITIES[entry.entity].table} SET dirty = 0 WHERE uid = ? AND NOT EXISTS (SELECT 1 FROM outbox WHERE entity = ? AND uid = ?)`,
      [entry.uid, entry.entity, entry.uid],
    );
  }
}

async function applyRejected(
  db: Database,
  entry: OutboxEntry,
  result: PushResult,
  now: string,
  summary: ResultSummary,
  incomplete: Map<string, number>,
): Promise<void> {
  const code = result.error?.code ?? "rejected";

  if (code === "quota_exceeded") {
    const message = result.error?.message ?? "";
    await pauseEntity(db, entry.entity, message);
    summary.paused.set(entry.entity, message);
    return;
  }

  if (code === "edited_deleted_row") {
    // The row was deleted on the server: the edit is gone and the person is told. The tombstone arrives by pull.
    await dropEntry(db, entry);
    for (const c of result.conflicts) {
      await recordConflict(
        db,
        {
          entity: entry.entity,
          uid: entry.uid,
          field: c.field,
          kind: c.kind,
          lost: c.lost_value,
          applied: c.applied_value,
        },
        now,
      );
      summary.conflicts += 1;
    }
    return;
  }

  let retryable = RETRYABLE.has(code);
  if (code === "transfer_incomplete" && entry.entity === "transaction") {
    // Both legs refused together means the pair is broken for good; a lone leg may only be waiting for its partner.
    const transfer = await localTransferUid(db, entry.uid);
    retryable = transfer === null || (incomplete.get(transfer) ?? 0) < 2;
  }

  if (retryable) {
    const attempts = await db.all<{ retries: number }>("SELECT retries FROM outbox WHERE id = ?", [entry.id]);
    const retries = (attempts[0]?.retries ?? 0) + 1;
    if (retries < MAX_RETRIES) {
      await db.run("UPDATE outbox SET retries = ? WHERE id = ?", [retries, entry.id]);
      summary.retryLater += 1;
      return;
    }
  }

  await recordProblem(db, entry, result, now);
  summary.problems += 1;
}

export interface Problem {
  id: number;
  entity: EntityName;
  uid: string;
  op: string;
  code: string;
  message: string;
  field: string;
  fields: Record<string, unknown>;
  created_at: string;
}

export async function listProblems(db: Database): Promise<Problem[]> {
  const rows = await db.all<Omit<Problem, "fields"> & { fields: string }>(
    "SELECT id, entity, uid, op, code, message, field, fields, created_at FROM sync_problems WHERE dismissed_at IS NULL ORDER BY id",
  );
  return rows.map((r) => ({ ...r, fields: JSON.parse(r.fields) as Record<string, unknown> }));
}

import type { PullChange, PullPage } from "../../src/sync/types";

/** Builders for the rows a server sends, with every field the client stores. */

export function accountRow(uid: string, over: Record<string, unknown> = {}): Record<string, unknown> {
  return { uid, version: 1, name: "Wallet", kind: "cash", opening_cents: 0, archived_at: null, position: 0, ...over };
}

export function categoryRow(uid: string, over: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    uid,
    version: 1,
    name: "Food",
    kind: "expense",
    parent_uid: null,
    color: "",
    icon: "",
    archived_at: null,
    ...over,
  };
}

export function tagRow(uid: string, over: Record<string, unknown> = {}): Record<string, unknown> {
  return { uid, version: 1, name: "trip", color: "", ...over };
}

export function transactionRow(
  uid: string,
  account: string,
  over: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    uid,
    version: 1,
    account_uid: account,
    category_uid: null,
    kind: "expense",
    amount_cents: 1000,
    description: "",
    occurred_on: "2026-11-10",
    transfer_uid: null,
    tag_uids: [],
    statement_ref: null,
    recurrence_uid: null,
    occurrence_on: null,
    installment_group_uid: null,
    installment_no: null,
    installment_total: null,
    ...over,
  };
}

export function upsert(seq: number, entity: string, row: Record<string, unknown>): PullChange {
  return { seq, entity, uid: row.uid as string, op: "upsert", version: row.version as number, row };
}

export function tombstone(seq: number, entity: string, uid: string, version = 2): PullChange {
  return { seq, entity, uid, op: "delete", version, row: null };
}

export function page(changes: PullChange[], next?: number, hasMore = false): PullPage {
  return {
    changes,
    next_cursor: next ?? Math.max(0, ...changes.map((c) => c.seq)),
    has_more: hasMore,
    server_time: "2026-11-24T12:00:00Z",
  };
}

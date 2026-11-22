import type { EntityName } from "../db/entities";
import type { OutboxOp } from "../db/repository";

/** The wire shapes of docs/specs/sync-protocol.md that the client reads or sends. */

export interface Mutation {
  op_id: string;
  entity: EntityName;
  uid: string;
  op: OutboxOp;
  base_seq: number;
  fields: Record<string, unknown>;
}

export interface ConflictOut {
  field: string;
  kind: string;
  lost_value: unknown;
  applied_value: unknown;
}

export type ResultStatus = "applied" | "merged" | "duplicate" | "rejected";

export interface PushResult {
  op_id: string;
  status: ResultStatus;
  entity: EntityName;
  uid: string;
  row: Record<string, unknown> | null;
  version: number;
  seq: number;
  conflicts: ConflictOut[];
  effects: { entity: EntityName; uid: string; op: string; seq: number }[];
  replaced_by_uid: string;
  error: { code: string; message: string; field: string } | null;
}

export interface PullChange {
  seq: number;
  entity: string;
  uid: string;
  op: "upsert" | "delete";
  version: number;
  row: Record<string, unknown> | null;
}

export interface PullPage {
  changes: PullChange[];
  next_cursor: number;
  has_more: boolean;
  server_time: string;
}

export interface SnapshotQuery {
  entity: string;
  after: string;
  snapshot_id: string;
  limit: number;
}

export interface SnapshotPage {
  snapshot_id: string;
  snapshot_cursor: number;
  entity: string;
  rows: Record<string, unknown>[];
  next_after: string;
  has_more: boolean;
}

export interface ServerInfo {
  protocol: { min: number; max: number };
  server_time: string;
  workspace: { uid: string; currency: string; timezone: string };
  entities: { name: string }[];
  limits: { max_mutations: number; pull_max_limit: number; snapshot_max_rows: number };
}

export interface PairResult {
  token: string;
  device_id: string;
  workspace: { uid: string; name: string; currency: string; timezone: string };
  server_time: string;
}

export type DevicePlatform = "ios" | "android" | "web" | "other";

/** The only place the app touches the network, and it exists only after the person connects. */
export interface Transport {
  info(): Promise<ServerInfo>;
  registerDevice(name: string, platform: DevicePlatform): Promise<{ device_id: string }>;
  pull(cursor: number, limit: number): Promise<PullPage>;
  snapshot(query: SnapshotQuery): Promise<SnapshotPage>;
  push(mutations: readonly Mutation[]): Promise<PushResult[]>;
  pair(code: string, deviceName: string, platform: DevicePlatform): Promise<PairResult>;
}

export type TransportErrorKind = "network" | "timeout" | "http";

/** A failed call. The message never holds a token or a pairing code. */
export class TransportError extends Error {
  constructor(
    readonly kind: TransportErrorKind,
    message: string,
    readonly status = 0,
    readonly code = "",
    readonly retryAfterSeconds = 0,
  ) {
    super(message);
    this.name = "TransportError";
  }
}

export interface Clock {
  now(): number;
  setTimeout(fn: () => void, ms: number): unknown;
  clearTimeout(handle: unknown): void;
}

export interface Random {
  /** A number in [0, 1). */
  next(): number;
}

export const systemClock: Clock = {
  now: () => Date.now(),
  setTimeout: (fn, ms) => setTimeout(fn, ms),
  clearTimeout: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
};

export const systemRandom: Random = { next: () => Math.random() };

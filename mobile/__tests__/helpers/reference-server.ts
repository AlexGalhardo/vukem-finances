import { ENTITIES, type EntityName } from "../../src/db/entities";
import {
  type DevicePlatform,
  type Mutation,
  type PairResult,
  type PullChange,
  type PullPage,
  type PushResult,
  type ServerInfo,
  type SnapshotPage,
  type SnapshotQuery,
  type Transport,
  TransportError,
} from "../../src/sync/types";
import type { Harness, Role } from "./scenario-runner";

/**
 * A small model of the server of docs/specs/sync-protocol.md, enough to run the
 * shared scenarios and a convergence property test without a network: row
 * versions, field level merging with the later arrival winning, delete wins
 * over an edit, transfers, recurrence occurrences, quota, roles, a log with
 * compaction and the op_id store. The Go server is the real thing; this model
 * and the real server are held to the same scenario files.
 */

interface StoredRow {
  uid: string;
  entity: EntityName;
  version: number;
  fields: Record<string, unknown>;
  /** The seq at which each field last changed. */
  fieldSeq: Record<string, number>;
  /** The newest seq of the row. */
  seq: number;
  deleted: boolean;
  /** Days ago, in the model's own time, when it was deleted. */
  deletedDaysAgo: number;
}

const ORDER: readonly EntityName[] = [
  "account",
  "category",
  "tag",
  "credit_card",
  "recurrence",
  "budget",
  "transaction",
];
const RECORD_ROLES: readonly Role[] = ["owner", "admin", "member"];
const MANAGE_ROLES: readonly Role[] = ["owner", "admin"];

const DEFAULTS: Record<string, unknown> = {
  opening_cents: 0,
  position: 0,
  color: "",
  icon: "",
  archived_at: null,
  parent_uid: null,
  description: "",
  tag_uids: [],
  rollover: false,
  active: true,
  interval: 1,
  limit_cents: 0,
  brand: "",
  last4: "",
};

const REQUIRED: Record<EntityName, readonly string[]> = {
  account: ["name", "kind"],
  category: ["name", "kind"],
  tag: ["name"],
  transaction: ["account_uid", "kind", "amount_cents", "occurred_on"],
  budget: ["limit_cents"],
  credit_card: ["account_uid", "closing_day", "due_day"],
  recurrence: ["account_uid", "kind", "amount_cents", "frequency", "start_date"],
};

const RESTORE_WINDOW_DAYS = 30;

function fail(op: Mutation, code: string, message = code, field = ""): PushResult {
  return {
    op_id: op.op_id,
    status: "rejected",
    entity: op.entity,
    uid: op.uid,
    row: null,
    version: 0,
    seq: 0,
    conflicts: [],
    effects: [],
    replaced_by_uid: "",
    error: { code, message, field },
  };
}

export class ReferenceServer {
  private rows = new Map<string, StoredRow>();
  private log: { seq: number; entity: EntityName; uid: string }[] = [];
  private seq = 0;
  private compactedThrough = 0;
  private ops = new Map<string, PushResult>();
  private revoked = new Set<string>();
  readonly roles = new Map<string, Role>();
  private readonly plan: "pro" | "free";

  constructor(options: { plan?: "pro" | "free"; devices?: Record<string, Role> } = {}) {
    this.plan = options.plan ?? "pro";
    for (const [id, role] of Object.entries(options.devices ?? {})) this.roles.set(id, role);
    if (this.plan === "free") {
      // The Free plan holds one live account: the workspace starts with it.
      this.store("account", "01JD5Z8K3M00000000000SEED1", { name: "Seed", kind: "checking" }, 0);
    }
  }

  // ---- admin ----
  revoke(device: string): void {
    this.revoked.add(device);
  }

  compact(): void {
    this.compactedThrough = this.seq;
  }

  ageTombstones(): void {
    for (const r of this.rows.values()) if (r.deleted) r.deletedDaysAgo = 31;
  }

  transportFor(device: string): ReferenceTransport {
    return new ReferenceTransport(this, device);
  }

  // ---- inspection for tests ----
  live(entity: EntityName): Map<string, Record<string, unknown>> {
    const out = new Map<string, Record<string, unknown>>();
    for (const r of this.rows.values()) {
      if (r.entity === entity && !r.deleted) out.set(r.uid, this.wire(r));
    }
    return out;
  }

  lastSeq(): number {
    return this.seq;
  }

  // ---- protocol ----
  guard(device: string): void {
    if (this.revoked.has(device) || !this.roles.has(device)) {
      throw new TransportError("http", "This device is unknown or was revoked.", 403, "device_revoked");
    }
  }

  info(): ServerInfo {
    return {
      protocol: { min: 1, max: 1 },
      server_time: "2026-11-24T12:00:00Z",
      workspace: { uid: "1", currency: "USD", timezone: "UTC" },
      entities: ORDER.map((name) => ({ name })),
      limits: { max_mutations: 200, pull_max_limit: 1000, snapshot_max_rows: 1000 },
    };
  }

  pull(device: string, cursor: number, limit: number): PullPage {
    this.guard(device);
    if (cursor < this.compactedThrough) {
      throw new TransportError("http", "resync", 410, "resync_required");
    }
    const entries = this.log.filter((l) => l.seq > cursor).slice(0, limit);
    const last = entries[entries.length - 1];
    const next = last?.seq ?? cursor;
    const newest = new Map<string, number>();
    for (const e of entries) newest.set(`${e.entity}/${e.uid}`, e.seq);
    const changes: PullChange[] = [];
    for (const [key, seq] of newest) {
      const row = this.rows.get(key);
      if (row === undefined) continue;
      changes.push(
        row.deleted
          ? { seq, entity: row.entity, uid: row.uid, op: "delete", version: row.version, row: null }
          : { seq, entity: row.entity, uid: row.uid, op: "upsert", version: row.version, row: this.wire(row) },
      );
    }
    changes.sort(
      (a, b) => ORDER.indexOf(a.entity as EntityName) - ORDER.indexOf(b.entity as EntityName) || a.seq - b.seq,
    );
    return {
      changes,
      next_cursor: next,
      has_more: this.log.some((l) => l.seq > next),
      server_time: "2026-11-24T12:00:00Z",
    };
  }

  snapshot(device: string, q: SnapshotQuery): SnapshotPage {
    this.guard(device);
    const all = [...this.rows.values()]
      .filter((r) => r.entity === q.entity && !r.deleted && r.uid > q.after)
      .sort((a, b) => (a.uid < b.uid ? -1 : 1));
    const rows = all.slice(0, q.limit);
    const more = all.length > rows.length;
    return {
      snapshot_id: `s.${this.seq}`,
      snapshot_cursor: this.seq,
      entity: q.entity,
      rows: rows.map((r) => this.wire(r)),
      next_after: more ? (rows[rows.length - 1]?.uid ?? "") : "",
      has_more: more,
    };
  }

  push(device: string, mutations: readonly Mutation[]): PushResult[] {
    this.guard(device);
    const role = this.roles.get(device) ?? "viewer";
    const badTransfers = this.checkTransfers(mutations);
    const results: PushResult[] = [];
    const seenInBatch = new Map<string, PushResult>();
    for (const m of mutations) {
      const stored = this.ops.get(m.op_id) ?? seenInBatch.get(m.op_id);
      if (stored !== undefined) {
        results.push(stored);
        continue;
      }
      const result = badTransfers.has(m.op_id)
        ? fail(m, badTransfers.get(m.op_id) ?? "transfer_invalid")
        : this.apply(m, role);
      this.ops.set(m.op_id, result);
      seenInBatch.set(m.op_id, result);
      results.push(result);
    }
    return results;
  }

  // ---- internals ----
  private wire(r: StoredRow): Record<string, unknown> {
    return { uid: r.uid, version: r.version, ...r.fields };
  }

  private key(entity: string, uid: string): string {
    return `${entity}/${uid}`;
  }

  private touch(row: StoredRow): number {
    this.seq += 1;
    row.seq = this.seq;
    this.log.push({ seq: this.seq, entity: row.entity, uid: row.uid });
    return this.seq;
  }

  private store(entity: EntityName, uid: string, input: Record<string, unknown>, _seq: number): StoredRow {
    const fields: Record<string, unknown> = {};
    for (const name of Object.keys(ENTITIES[entity].fields)) fields[name] = input[name] ?? DEFAULTS[name] ?? null;
    const row: StoredRow = { uid, entity, version: 1, fields, fieldSeq: {}, seq: 0, deleted: false, deletedDaysAgo: 0 };
    this.rows.set(this.key(entity, uid), row);
    const seq = this.touch(row);
    for (const name of Object.keys(fields)) row.fieldSeq[name] = seq;
    return row;
  }

  private allowed(role: Role, entity: EntityName): boolean {
    return (entity === "transaction" || entity === "recurrence" ? RECORD_ROLES : MANAGE_ROLES).includes(role);
  }

  /** The legs of a transfer must come as exactly two live rows, one expense and one income of the same amount and date. */
  private checkTransfers(mutations: readonly Mutation[]): Map<string, string> {
    const bad = new Map<string, string>();
    const groups = new Map<string, Mutation[]>();
    for (const m of mutations) {
      if (m.entity !== "transaction" || m.op !== "upsert") continue;
      const group = m.fields.transfer_uid;
      if (typeof group === "string") groups.set(group, [...(groups.get(group) ?? []), m]);
    }
    for (const [group, legs] of groups) {
      const stored = [...this.rows.values()].filter(
        (r) =>
          r.entity === "transaction" &&
          !r.deleted &&
          r.fields.transfer_uid === group &&
          !legs.some((l) => l.uid === r.uid),
      );
      if (stored.length + legs.length !== 2) {
        for (const l of legs) bad.set(l.op_id, "transfer_incomplete");
        continue;
      }
      const [a, b] = legs.map((l) => l.fields);
      const valid =
        legs.length === 2 &&
        a !== undefined &&
        b !== undefined &&
        a.kind !== b.kind &&
        a.amount_cents === b.amount_cents &&
        a.occurred_on === b.occurred_on &&
        a.account_uid !== b.account_uid &&
        (a.category_uid ?? null) === null &&
        (b.category_uid ?? null) === null;
      if (!valid) for (const l of legs) bad.set(l.op_id, "transfer_invalid");
    }
    return bad;
  }

  private apply(m: Mutation, role: Role): PushResult {
    if (!Object.hasOwn(ENTITIES, m.entity)) return fail(m, "unknown_entity");
    if (!this.allowed(role, m.entity)) return fail(m, "forbidden", "Your role cannot change that.");
    const existing = this.rows.get(this.key(m.entity, m.uid));
    if (m.op === "delete") return this.remove(m, existing);
    if (m.op === "restore") return this.restore(m, existing);
    return this.upsert(m, existing);
  }

  private ok(m: Mutation, row: StoredRow, status: PushResult["status"], extra: Partial<PushResult> = {}): PushResult {
    return {
      op_id: m.op_id,
      status,
      entity: m.entity,
      uid: m.uid,
      row: row.deleted ? null : this.wire(row),
      version: row.version,
      seq: row.seq,
      conflicts: [],
      effects: [],
      replaced_by_uid: "",
      error: null,
      ...extra,
    };
  }

  private remove(m: Mutation, row: StoredRow | undefined): PushResult {
    if (row === undefined) return fail(m, "unknown_row");
    if (row.deleted) return this.ok(m, row, "duplicate");
    const effects: PushResult["effects"] = [];
    row.deleted = true;
    row.deletedDaysAgo = 0;
    row.version += 1;
    this.touch(row);
    const cascade: StoredRow[] = [];
    if (row.entity === "transaction" && typeof row.fields.transfer_uid === "string") {
      cascade.push(
        ...[...this.rows.values()].filter(
          (r) => r.entity === "transaction" && !r.deleted && r.fields.transfer_uid === row.fields.transfer_uid,
        ),
      );
    }
    if (row.entity === "account") {
      cascade.push(
        ...[...this.rows.values()].filter(
          (r) =>
            !r.deleted &&
            ["transaction", "credit_card", "recurrence"].includes(r.entity) &&
            r.fields.account_uid === row.uid,
        ),
      );
    }
    for (const c of cascade) {
      c.deleted = true;
      c.deletedDaysAgo = 0;
      c.version += 1;
      effects.push({ entity: c.entity, uid: c.uid, op: "delete", seq: this.touch(c) });
    }
    return this.ok(m, row, "applied", { effects });
  }

  private restore(m: Mutation, row: StoredRow | undefined): PushResult {
    if (row === undefined) return fail(m, "unknown_row");
    if (!row.deleted) return this.ok(m, row, "duplicate");
    if (row.deletedDaysAgo > RESTORE_WINDOW_DAYS) return fail(m, "restore_window_expired");
    const missing = this.checkRefs(row.entity, row.fields);
    if (missing !== null) return fail(m, missing, missing);
    row.deleted = false;
    row.version += 1;
    this.touch(row);
    return this.ok(m, row, "applied");
  }

  private checkRefs(entity: EntityName, fields: Record<string, unknown>): string | null {
    const refs: [string, EntityName][] = [...ENTITIES[entity].refs.map(([f, e]): [string, EntityName] => [f, e])];
    if (entity === "transaction" && typeof fields.recurrence_uid === "string")
      refs.push(["recurrence_uid", "recurrence"]);
    if (entity === "credit_card" && typeof fields.payment_account_uid === "string")
      refs.push(["payment_account_uid", "account"]);
    for (const [field, parent] of refs) {
      const value = fields[field];
      if (typeof value !== "string" || value === "") continue;
      const p = this.rows.get(this.key(parent, value));
      if (p === undefined) return "unknown_reference";
      if (p.deleted) return "parent_deleted";
    }
    const tags = Array.isArray(fields.tag_uids) ? (fields.tag_uids as string[]) : [];
    for (const t of tags) {
      const p = this.rows.get(this.key("tag", t));
      if (p === undefined) return "unknown_reference";
      if (p.deleted) return "parent_deleted";
    }
    return null;
  }

  private upsert(m: Mutation, row: StoredRow | undefined): PushResult {
    const known = new Set(Object.keys(ENTITIES[m.entity].fields));
    for (const name of Object.keys(m.fields)) {
      if (!known.has(name)) return fail(m, "unknown_field", `unknown field ${name}`, name);
    }
    return row === undefined ? this.create(m) : this.edit(m, row);
  }

  private create(m: Mutation): PushResult {
    for (const name of REQUIRED[m.entity]) {
      if (m.fields[name] === undefined || m.fields[name] === null)
        return fail(m, "missing_field", `${name} is required`, name);
    }
    const merged = {
      ...Object.fromEntries(Object.keys(ENTITIES[m.entity].fields).map((n) => [n, DEFAULTS[n] ?? null])),
      ...m.fields,
    };
    const refError = this.checkRefs(m.entity, merged);
    if (refError !== null) return fail(m, refError);

    if (
      m.entity === "transaction" &&
      typeof merged.recurrence_uid === "string" &&
      typeof merged.occurrence_on === "string"
    ) {
      const twin = [...this.rows.values()].find(
        (r) =>
          r.entity === "transaction" &&
          r.fields.recurrence_uid === merged.recurrence_uid &&
          r.fields.occurrence_on === merged.occurrence_on,
      );
      if (twin !== undefined) {
        return { ...this.ok(m, twin, "duplicate", { replaced_by_uid: twin.uid }), uid: m.uid };
      }
    }
    if (
      (m.entity === "category" || m.entity === "tag") &&
      [...this.rows.values()].some((r) => r.entity === m.entity && !r.deleted && r.fields.name === merged.name)
    ) {
      return fail(m, "duplicate_name", "A live row has this name.", "name");
    }
    if (this.plan === "free" && m.entity === "account" && merged.archived_at === null) {
      const live = [...this.rows.values()].filter(
        (r) => r.entity === "account" && !r.deleted && r.fields.archived_at === null,
      ).length;
      if (live >= 1) return fail(m, "quota_exceeded", "The Free plan holds one account.");
    }
    return this.ok(m, this.store(m.entity, m.uid, merged, 0), "applied");
  }

  private edit(m: Mutation, row: StoredRow): PushResult {
    if (row.deleted) {
      return {
        ...fail(m, "edited_deleted_row", "The row was deleted."),
        conflicts: Object.entries(m.fields).map(([field, value]) => ({
          field,
          kind: "edited_deleted_row",
          lost_value: value,
          applied_value: null,
        })),
      };
    }
    const changedAfterBase = row.seq > m.base_seq;
    const changed = Object.entries(m.fields).filter(([f, v]) => JSON.stringify(row.fields[f]) !== JSON.stringify(v));
    if (changed.length === 0) return this.ok(m, row, "duplicate");
    const candidate = { ...row.fields, ...Object.fromEntries(changed) };
    const refError = this.checkRefs(m.entity, candidate);
    if (refError !== null) return fail(m, refError);
    if (
      (m.entity === "category" || m.entity === "tag") &&
      changed.some(([f]) => f === "name") &&
      [...this.rows.values()].some(
        (r) => r.entity === m.entity && !r.deleted && r.uid !== row.uid && r.fields.name === candidate.name,
      )
    ) {
      return fail(m, "duplicate_name", "A live row has this name.", "name");
    }
    const conflicts: PushResult["conflicts"] = [];
    for (const [field, value] of changed) {
      if ((row.fieldSeq[field] ?? 0) > m.base_seq) {
        conflicts.push({ field, kind: "field", lost_value: row.fields[field], applied_value: value });
      }
    }
    for (const [field, value] of changed) row.fields[field] = value;
    row.version += 1;
    const seq = this.touch(row);
    for (const [field] of changed) row.fieldSeq[field] = seq;
    return this.ok(m, row, changedAfterBase ? "merged" : "applied", { conflicts });
  }

  pair(_code: string, name: string): PairResult {
    return {
      token: "reference-token",
      device_id: `device-${name}`,
      workspace: { uid: "1", name: "Reference", currency: "USD", timezone: "UTC" },
      server_time: "",
    };
  }
}

/** A device's view of the reference server, behind the same Transport interface the engine uses. */
export class ReferenceTransport implements Transport {
  /** What the last calls returned, for the scenario runner. */
  readonly pushes: PushResult[][] = [];
  readonly pulls: PullPage[] = [];
  /** Fail the next calls with these errors (a dropped connection, a timeout). */
  readonly failures: TransportError[] = [];
  /** Lose the answer of the next push after the server processed it. */
  dropNextPushResponse = false;

  constructor(
    private readonly server: ReferenceServer,
    private readonly device: string,
  ) {}

  private maybeFail(): void {
    const f = this.failures.shift();
    if (f !== undefined) throw f;
  }

  async info(): Promise<ServerInfo> {
    this.server.guard(this.device);
    this.maybeFail();
    return this.server.info();
  }

  async registerDevice(): Promise<{ device_id: string }> {
    return { device_id: this.device };
  }

  async pull(cursor: number, limit: number): Promise<PullPage> {
    this.maybeFail();
    const page = this.server.pull(this.device, cursor, limit);
    this.pulls.push(page);
    return page;
  }

  async snapshot(q: SnapshotQuery): Promise<SnapshotPage> {
    this.maybeFail();
    return this.server.snapshot(this.device, q);
  }

  async push(mutations: readonly Mutation[]): Promise<PushResult[]> {
    this.maybeFail();
    const results = this.server.push(this.device, mutations);
    this.pushes.push(results);
    if (this.dropNextPushResponse) {
      this.dropNextPushResponse = false;
      throw new TransportError("timeout", "the answer was lost");
    }
    return results;
  }

  async pair(_code: string, name: string, _platform: DevicePlatform): Promise<PairResult> {
    return this.server.pair("", name);
  }
}

/** The reference server behind the scenario runner's Harness. */
export function referenceHarness(
  plan: "pro" | "free",
  devices: Record<string, Role>,
): Harness & { server: ReferenceServer } {
  const server = new ReferenceServer({ plan, devices });
  return {
    server,
    transportFor: async (device) => server.transportFor(device),
    revoke: async (device) => server.revoke(device),
    compact: async () => server.compact(),
    ageTombstones: async () => server.ageTombstones(),
    live: async (entity) => server.live(entity),
  };
}

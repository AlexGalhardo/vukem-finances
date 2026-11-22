import type { Database } from "../db/database";
import { ENTITIES, type EntityName } from "../db/entities";
import type { Repository } from "../db/repository";
import { newUid } from "../domain/ids";
import { type ConnectInput, type ConnectResult, connectToServer } from "./connect";
import { SyncEngine } from "./engine";
import { describeReplace, type MergeResult, mergeWithServer, replaceWithServer } from "./merge";
import { type Connection, loadConnection, requestResync } from "./state";
import type { TokenStore } from "./transport";
import {
  type Clock,
  type DevicePlatform,
  type Mutation,
  type PairResult,
  type PullPage,
  type PushResult,
  type Random,
  type ServerInfo,
  type SnapshotPage,
  type SnapshotQuery,
  type Transport,
  TransportError,
} from "./types";

export interface SyncServices {
  db: Database;
  repo: Repository;
  tokens: TokenStore;
  /** Builds the HTTP transport for an address. Pairing needs no token; every other call reads it from `tokens`. */
  makeTransport: (baseUrl: string, allowInsecure: boolean, deviceId: () => string | null) => Transport;
  clock: Clock;
  random: Random;
  /** What the phone is called until the person types a name. */
  defaultDeviceName: string;
  platform: DevicePlatform;
}

export type ConnectOutcome = ConnectResult | { ok: false; reason: "pairing" | "network" | "rate" };

/** A transport that follows the saved connection: it does not exist until the person connects. */
class FollowingTransport implements Transport {
  private current: { key: string; transport: Transport } | null = null;

  constructor(private readonly s: SyncServices) {}

  private async inner(): Promise<Transport> {
    const c = await loadConnection(this.s.db);
    if (c === null) {
      throw new Error("not connected");
    }
    const key = `${c.serverUrl}|${c.insecure}`;
    if (this.current?.key !== key) {
      this.current = { key, transport: this.s.makeTransport(c.serverUrl, c.insecure, () => this.deviceId) };
    }
    this.deviceId = c.deviceId;
    return this.current.transport;
  }

  private deviceId: string | null = null;

  async info(): Promise<ServerInfo> {
    return (await this.inner()).info();
  }

  async registerDevice(name: string, platform: DevicePlatform): Promise<{ device_id: string }> {
    return (await this.inner()).registerDevice(name, platform);
  }

  async pull(cursor: number, limit: number): Promise<PullPage> {
    return (await this.inner()).pull(cursor, limit);
  }

  async snapshot(q: SnapshotQuery): Promise<SnapshotPage> {
    return (await this.inner()).snapshot(q);
  }

  async push(mutations: readonly Mutation[]): Promise<PushResult[]> {
    return (await this.inner()).push(mutations);
  }

  async pair(code: string, name: string, platform: DevicePlatform): Promise<PairResult> {
    return (await this.inner()).pair(code, name, platform);
  }
}

/** What the sync screens call: the engine and the choices a person makes around it. */
export class SyncController {
  readonly engine: SyncEngine;
  private readonly transport: Transport;

  constructor(private readonly s: SyncServices) {
    this.transport = new FollowingTransport(s);
    this.engine = new SyncEngine({
      db: s.db,
      repo: s.repo,
      transport: this.transport,
      tokens: s.tokens,
      clock: s.clock,
      random: s.random,
      deviceName: s.defaultDeviceName,
      platform: s.platform,
    });
  }

  get defaultDeviceName(): string {
    return this.s.defaultDeviceName;
  }

  connection(): Promise<Connection | null> {
    return loadConnection(this.s.db);
  }

  private iso(): string {
    return new Date(this.s.clock.now()).toISOString();
  }

  /** Pairs with a server. On success the engine starts, and a phone with data is asked how to join. */
  async connect(input: ConnectInput): Promise<ConnectOutcome> {
    try {
      const result = await connectToServer(
        {
          db: this.s.db,
          tokens: this.s.tokens,
          makeTransport: (url, insecure) => this.s.makeTransport(url, insecure, () => null),
        },
        input,
      );
      if (result.ok) {
        this.engine.stop();
        await this.engine.start();
        void this.engine.trigger("manual");
      }
      return result;
    } catch (error) {
      if (error instanceof TransportError) {
        if (error.status === 429) return { ok: false, reason: "rate" };
        if (error.kind !== "http") return { ok: false, reason: "network" };
        return { ok: false, reason: "pairing" };
      }
      throw error;
    }
  }

  /** "Merge (recommended)". */
  async merge(): Promise<MergeResult> {
    const result = await mergeWithServer({ db: this.s.db, transport: this.transport, now: () => this.iso() });
    await this.engine.afterMergeChoice();
    return result;
  }

  describeReplace(): Promise<{ rows: number; pending: number }> {
    return describeReplace(this.s.db);
  }

  /** "Replace this phone's data": call it only after the confirmation. */
  async replace(): Promise<void> {
    await replaceWithServer({ db: this.s.db, transport: this.transport, now: () => this.iso() }, true);
    await this.engine.afterMergeChoice();
  }

  async disconnect(clearQueue: boolean): Promise<void> {
    await this.engine.disconnect(clearQueue);
  }

  /** The separate, confirmed action that deletes the data on this phone. */
  async resetLocalData(): Promise<void> {
    const { db } = this.s;
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
    await this.engine.refresh();
  }

  async dismissConflict(id: number): Promise<void> {
    await this.s.db.run("UPDATE sync_conflicts SET dismissed_at = ? WHERE id = ?", [this.iso(), id]);
    await this.engine.refresh();
  }

  /** Writes the value that lost again, as a normal edit that syncs like any other. */
  async applyConflictAgain(id: number): Promise<void> {
    const rows = await this.s.db.all<{
      entity: EntityName;
      uid: string;
      field: string;
      kind: string;
      lost_value: string;
    }>("SELECT entity, uid, field, kind, lost_value FROM sync_conflicts WHERE id = ?", [id]);
    const c = rows[0];
    if (c === undefined) return;
    const { repo } = this.s;
    if (c.kind === "edited_deleted_row") {
      await repo.restore(c.entity, c.uid);
    }
    await repo.update(c.entity, c.uid, { [c.field]: JSON.parse(c.lost_value) as unknown } as never);
    await this.dismissConflict(id);
    this.engine.onLocalWrite();
  }

  /** Sends a rejected mutation once more, as a new queued change. */
  async retryProblem(id: number): Promise<void> {
    const { db } = this.s;
    const rows = await db.all<{ entity: EntityName; uid: string; op: string; fields: string }>(
      "SELECT entity, uid, op, fields FROM sync_problems WHERE id = ?",
      [id],
    );
    const p = rows[0];
    if (p === undefined) return;
    await db.transaction(async (db) => {
      const row = await db.all<{ server_seq: number | null }>(
        `SELECT server_seq FROM ${ENTITIES[p.entity].table} WHERE uid = ?`,
        [p.uid],
      );
      await db.run(
        "INSERT INTO outbox (op_id, entity, uid, op, base_seq, fields, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
        [newUid(), p.entity, p.uid, p.op, row[0]?.server_seq ?? 0, p.fields, this.iso()],
      );
      await db.run(`UPDATE ${ENTITIES[p.entity].table} SET dirty = 1 WHERE uid = ?`, [p.uid]);
      await db.run("UPDATE sync_problems SET dismissed_at = ? WHERE id = ?", [this.iso(), id]);
    });
    await this.engine.refresh();
    this.engine.onLocalWrite();
  }

  /**
   * Gives up on a rejected mutation. A row the server never had goes away with
   * it; for anything else the next run downloads a snapshot, so this phone does
   * not keep a value the server refused.
   */
  async discardProblem(id: number): Promise<void> {
    const { db } = this.s;
    const rows = await db.all<{ entity: EntityName; uid: string; op: string }>(
      "SELECT entity, uid, op FROM sync_problems WHERE id = ?",
      [id],
    );
    const p = rows[0];
    if (p === undefined) return;
    await db.transaction(async (db) => {
      const table = ENTITIES[p.entity].table;
      const row = await db.all<{ server_seq: number | null }>(`SELECT server_seq FROM ${table} WHERE uid = ?`, [p.uid]);
      if (p.op === "upsert" && row[0] !== undefined && row[0].server_seq === null) {
        await db.run(`DELETE FROM ${table} WHERE uid = ?`, [p.uid]);
        await db.run("DELETE FROM outbox WHERE entity = ? AND uid = ?", [p.entity, p.uid]);
        if (p.entity === "transaction") await db.run("DELETE FROM transaction_tags WHERE transaction_uid = ?", [p.uid]);
      } else {
        await requestResync(db);
      }
      await db.run("UPDATE sync_problems SET dismissed_at = ? WHERE id = ?", [this.iso(), id]);
    });
    await this.engine.refresh();
  }
}

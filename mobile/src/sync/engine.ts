import type { Database } from "../db/database";
import { ENTITIES } from "../db/entities";
import type { Repository } from "../db/repository";
import { PULL_LIMIT, pullUntilDone } from "./apply";
import { backoffDelay } from "./backoff";
import { integrityPass } from "./integrity";
import { BATCH_SIZE, markAttempt, nextBatch } from "./outbox";
import { handleResults, type ResultSummary } from "./results";
import { resync } from "./resync";
import {
  clearConnection,
  getCursor,
  getMeta,
  lastSyncAt,
  loadConnection,
  pausedEntities,
  resumeEntities,
  setDeviceId,
  setLastSyncAt,
} from "./state";
import type { TokenStore } from "./transport";
import { type Clock, type DevicePlatform, type Random, type Transport, TransportError } from "./types";

export type SyncStatus =
  | "disconnected"
  | "idle"
  | "syncing"
  | "offline"
  | "error"
  | "auth_failed"
  | "needs_merge_choice";

export type Trigger = "manual" | "foreground" | "write" | "timer" | "backoff";

export interface EngineSnapshot {
  status: SyncStatus;
  /** Queued changes not sent yet. */
  pending: number;
  lastSyncAt: string | null;
  problems: number;
  conflicts: number;
  /** The plan quota message per paused entity. */
  paused: Record<string, string>;
  /** A short code of the last failure: `unsupported`, `server`, `rate_limited`. */
  failure: string | null;
  /** Milliseconds until the next automatic try after a failure, or 0. */
  retryInMs: number;
}

export interface EngineOptions {
  db: Database;
  repo: Repository;
  /** Null until the person connects: with no transport the engine never touches the network. */
  transport: Transport | null;
  tokens: TokenStore;
  clock: Clock;
  random: Random;
  deviceName?: string;
  platform?: DevicePlatform;
}

export const DEBOUNCE_MS = 3_000;
export const FOREGROUND_INTERVAL_MS = 5 * 60_000;
const MAX_PUSH_ROUNDS = 25;
const MAX_REPEATS = 3;

/**
 * The sync client. It depends on a Database, a Transport, a Clock and a Random
 * and nothing else, so every behavior is testable without a device or a network.
 */
export class SyncEngine {
  private status: SyncStatus = "disconnected";
  private failures = 0;
  private failure: string | null = null;
  private retryInMs = 0;
  private current: Promise<void> | null = null;
  private wroteDuringRun = false;
  private debounce: unknown = null;
  private retryTimer: unknown = null;
  private interval: unknown = null;
  private readonly listeners = new Set<() => void>();
  private cache: EngineSnapshot = {
    status: "disconnected",
    pending: 0,
    lastSyncAt: null,
    problems: 0,
    conflicts: 0,
    paused: {},
    failure: null,
    retryInMs: 0,
  };

  constructor(private readonly o: EngineOptions) {}

  getStatus(): SyncStatus {
    return this.status;
  }

  /** The last computed view, for a screen to render without waiting. */
  view(): EngineSnapshot {
    return this.cache;
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private iso(): string {
    return new Date(this.o.clock.now()).toISOString();
  }

  async refresh(): Promise<EngineSnapshot> {
    const { db } = this.o;
    const count = async (sql: string): Promise<number> => (await db.all<{ n: number }>(sql))[0]?.n ?? 0;
    this.cache = {
      status: this.status,
      pending: await count("SELECT COUNT(*) AS n FROM outbox"),
      lastSyncAt: await lastSyncAt(db),
      problems: await count("SELECT COUNT(*) AS n FROM sync_problems WHERE dismissed_at IS NULL"),
      conflicts: await count("SELECT COUNT(*) AS n FROM sync_conflicts WHERE dismissed_at IS NULL"),
      paused: Object.fromEntries(await pausedEntities(db)),
      failure: this.failure,
      retryInMs: this.retryInMs,
    };
    for (const l of this.listeners) l();
    return this.cache;
  }

  private async setStatus(status: SyncStatus): Promise<void> {
    this.status = status;
    await this.refresh();
  }

  /** Reads the connection and sets up the periodic trigger. Call it when the app opens and after a connect. */
  async start(): Promise<void> {
    const connection = await loadConnection(this.o.db);
    if (connection === null || this.o.transport === null) {
      await this.setStatus("disconnected");
      return;
    }
    if (this.status === "disconnected") {
      await this.setStatus("idle");
    }
    this.interval ??= this.o.clock.setTimeout(() => this.tick(), FOREGROUND_INTERVAL_MS);
  }

  stop(): void {
    for (const handle of [this.debounce, this.retryTimer, this.interval]) {
      if (handle !== null) this.o.clock.clearTimeout(handle);
    }
    this.debounce = this.retryTimer = this.interval = null;
  }

  private tick(): void {
    this.interval = this.o.clock.setTimeout(() => this.tick(), FOREGROUND_INTERVAL_MS);
    void this.trigger("timer");
  }

  /** The app calls this after every local write: the sync starts 3 seconds after the last one. */
  onLocalWrite(): void {
    if (this.status === "disconnected" || this.status === "auth_failed" || this.status === "needs_merge_choice") {
      void this.refresh();
      return;
    }
    if (this.current !== null) {
      this.wroteDuringRun = true;
      return;
    }
    if (this.debounce !== null) this.o.clock.clearTimeout(this.debounce);
    this.debounce = this.o.clock.setTimeout(() => {
      this.debounce = null;
      void this.trigger("write");
    }, DEBOUNCE_MS);
  }

  /** Asks for a sync. Triggers that arrive while one runs share that run. */
  trigger(reason: Trigger): Promise<void> {
    if (this.status === "disconnected" || this.status === "auth_failed" || this.status === "needs_merge_choice") {
      return this.refresh().then(() => undefined);
    }
    if (reason === "manual") {
      // A person pressing "Sync now" starts over: no waiting out a backoff, and a quota pause is tried again.
      this.cancelRetry();
    }
    if (this.current !== null) {
      this.wroteDuringRun = true;
      return this.current;
    }
    this.cancelRetry();
    const run = this.execute(reason === "manual").finally(() => {
      this.current = null;
    });
    this.current = run;
    return run;
  }

  private cancelRetry(): void {
    if (this.retryTimer !== null) {
      this.o.clock.clearTimeout(this.retryTimer);
      this.retryTimer = null;
      this.retryInMs = 0;
    }
  }

  private async execute(manual: boolean): Promise<void> {
    const { db, transport } = this.o;
    if (transport === null) {
      await this.setStatus("disconnected");
      return;
    }
    try {
      await this.setStatus("syncing");
      this.failure = null;
      this.retryInMs = 0;
      if (manual) await resumeEntities(db);

      const info = await transport.info();
      if (info.protocol.min > 1 || info.protocol.max < 1) {
        this.failure = "unsupported";
        await this.setStatus("error");
        return;
      }

      const connection = await loadConnection(db);
      if (connection === null) {
        await this.setStatus("disconnected");
        return;
      }
      if (connection.deviceId === null) {
        const device = await transport.registerDevice(connection.deviceName || "Phone", this.o.platform ?? "other");
        await setDeviceId(db, device.device_id);
      }
      if ((await getMeta(db, "sync.first_sync_done")) !== "1" && (await this.hasLocalData())) {
        await this.setStatus("needs_merge_choice");
        return;
      }

      let repeats = 0;
      do {
        this.wroteDuringRun = false;
        await this.pushAll();
        await this.pullAll();
        await integrityPass(db, transport, () => this.iso());
        repeats += 1;
      } while (this.wroteDuringRun && repeats < MAX_REPEATS);
      if (this.wroteDuringRun) await this.pushAll();

      await db.run(
        "INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1') ON CONFLICT(key) DO UPDATE SET value = '1'",
      );
      await setLastSyncAt(db, this.iso());
      this.failures = 0;
      await this.setStatus("idle");
    } catch (error) {
      await this.fail(error);
    }
  }

  private async hasLocalData(): Promise<boolean> {
    for (const meta of Object.values(ENTITIES)) {
      const rows = await this.o.db.all<{ n: number }>(
        `SELECT COUNT(*) AS n FROM ${meta.table} WHERE deleted_at IS NULL`,
      );
      if ((rows[0]?.n ?? 0) > 0) return true;
    }
    return false;
  }

  async pushAll(): Promise<ResultSummary[]> {
    const { db, transport } = this.o;
    const out: ResultSummary[] = [];
    if (transport === null) return out;
    const waiting = new Set<number>();
    for (let round = 0; round < MAX_PUSH_ROUNDS; round++) {
      const batch = await nextBatch(db, BATCH_SIZE, waiting);
      if (batch.entries.length === 0) break;
      await markAttempt(
        db,
        batch.entries.map((e) => e.id),
      );
      const results = await transport.push(batch.mutations);
      const summary = await handleResults(db, batch, results, this.iso());
      out.push(summary);
      // What is still queued after its result was handled waits for the next run (after the next pull).
      const left = await db.all<{ id: number }>(
        `SELECT id FROM outbox WHERE id IN (${batch.entries.map(() => "?").join(", ")})`,
        batch.entries.map((e) => e.id),
      );
      for (const l of left) waiting.add(l.id);
      await this.refresh();
    }
    return out;
  }

  async pullAll(): Promise<void> {
    const { db, transport } = this.o;
    if (transport === null) return;
    try {
      await pullUntilDone(db, transport, () => this.iso(), await getCursor(db), PULL_LIMIT);
    } catch (error) {
      if (error instanceof TransportError && error.status === 410 && error.code === "resync_required") {
        await resync(db, transport, () => this.iso());
        await pullUntilDone(db, transport, () => this.iso(), await getCursor(db), PULL_LIMIT);
        return;
      }
      throw error;
    }
  }

  private async fail(error: unknown): Promise<void> {
    if (error instanceof TransportError) {
      if (error.status === 401 || error.code === "device_revoked") {
        // Sync stops, the data stays; the person pairs again or disconnects.
        this.failure = "auth";
        await this.setStatus("auth_failed");
        return;
      }
      if (error.status === 426) {
        this.failure = "unsupported";
        await this.setStatus("error");
        return;
      }
      this.failures += 1;
      const wait = Math.max(backoffDelay(this.failures, this.o.random), error.retryAfterSeconds * 1000);
      this.failure = error.status === 429 ? "rate_limited" : error.kind === "http" ? "server" : null;
      this.schedule(wait);
      // "Offline" means a request failed: there is no network-state package.
      await this.setStatus(error.kind === "http" && error.status !== 429 ? "error" : "offline");
      return;
    }
    // Something in the client broke: show it, and try again later, since it may be a bad page of a flaky server.
    this.failures += 1;
    this.failure = "client";
    this.schedule(backoffDelay(this.failures, this.o.random));
    await this.setStatus("error");
  }

  private schedule(ms: number): void {
    this.retryInMs = ms;
    this.retryTimer = this.o.clock.setTimeout(() => {
      this.retryTimer = null;
      this.retryInMs = 0;
      void this.trigger("backoff");
    }, ms);
  }

  /** Forgets the server. Data and, unless the person clears it, the queue stay. */
  async disconnect(clearQueue: boolean): Promise<void> {
    this.stop();
    await this.o.tokens.clear();
    await clearConnection(this.o.db);
    if (clearQueue) await this.o.repo.clearOutbox();
    this.failures = 0;
    this.failure = null;
    await this.setStatus("disconnected");
  }

  /** The person chose Merge or Replace and it finished: sync as usual from here. */
  async afterMergeChoice(): Promise<void> {
    await this.setStatus("idle");
    await this.trigger("manual");
  }

  /** The person pressed "Pair again": the engine waits for a new token. */
  async resumeAfterRepair(): Promise<void> {
    this.failures = 0;
    this.failure = null;
    await this.setStatus("idle");
    await this.trigger("manual");
  }
}

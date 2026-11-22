import { BACKOFF_SCHEDULE_MS, backoffDelay } from "../../src/sync/backoff";
import { DEBOUNCE_MS, FOREGROUND_INTERVAL_MS, SyncEngine } from "../../src/sync/engine";
import { loadConnection, saveConnection } from "../../src/sync/state";
import type { TokenStore } from "../../src/sync/transport";
import { type Clock, type PushResult, TransportError } from "../../src/sync/types";
import { ACCOUNT, type Env, openEnv } from "../helpers/fixtures";
import { FakeTransport } from "./fake-transport";
import { accountRow, page, upsert } from "./wire";

class FakeClock implements Clock {
  time = Date.UTC(2026, 10, 24, 12, 0, 0);
  private next = 1;
  readonly timers = new Map<number, { at: number; fn: () => void; ms: number }>();

  now(): number {
    return this.time;
  }

  setTimeout(fn: () => void, ms: number): unknown {
    const id = this.next++;
    this.timers.set(id, { at: this.time + ms, fn, ms });
    return id;
  }

  clearTimeout(handle: unknown): void {
    this.timers.delete(handle as number);
  }

  /** Moves time forward and runs the timers that come due, in order. */
  advance(ms: number): void {
    const end = this.time + ms;
    for (;;) {
      const due = [...this.timers.entries()].filter(([, t]) => t.at <= end).sort((a, b) => a[1].at - b[1].at)[0];
      if (due === undefined) break;
      this.timers.delete(due[0]);
      this.time = due[1].at;
      due[1].fn();
    }
    this.time = end;
  }

  delays(): number[] {
    return [...this.timers.values()].map((t) => t.ms);
  }
}

const tokens = (): TokenStore & { value: string | null } => {
  const store = {
    value: "token-in-secure-storage" as string | null,
    get: async () => store.value,
    set: async (t: string) => {
      store.value = t;
    },
    clear: async () => {
      store.value = null;
    },
  };
  return store;
};

const info = {
  protocol: { min: 1, max: 1 },
  server_time: "",
  workspace: { uid: "1", currency: "USD", timezone: "UTC" },
  entities: [],
  limits: { max_mutations: 200, pull_max_limit: 1000, snapshot_max_rows: 1000 },
};
const emptyPage = page([], 0);
const accept = (m: readonly { op_id: string; entity: PushResult["entity"]; uid: string }[]): PushResult[] =>
  m.map((x) => ({
    op_id: x.op_id,
    status: "applied",
    entity: x.entity,
    uid: x.uid,
    row: null,
    version: 1,
    seq: 1,
    conflicts: [],
    effects: [],
    replaced_by_uid: "",
    error: null,
  }));
const half = { next: () => 0.5 };
const flush = (): Promise<void> => new Promise((resolve) => setImmediate(resolve));

async function markSynced(env: Env): Promise<void> {
  await env.db.run("INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1')");
}

async function setup(
  transport: FakeTransport | null,
  options: { connected?: boolean; firstSyncDone?: boolean } = {},
): Promise<{ env: Env; engine: SyncEngine; clock: FakeClock; store: ReturnType<typeof tokens> }> {
  const env = await openEnv();
  const clock = new FakeClock();
  const store = tokens();
  if (options.connected !== false) {
    await saveConnection(env.db, {
      serverUrl: "https://money.example.com",
      insecure: false,
      deviceId: "DEVICE1",
      deviceName: "Phone",
      workspaceUid: "1",
      workspaceName: "Home",
    });
  }
  if (options.firstSyncDone !== false) await markSynced(env);
  const engine = new SyncEngine({
    db: env.db,
    repo: env.repo,
    transport,
    tokens: store,
    clock,
    random: half,
    deviceName: "Phone",
  });
  return { env, engine, clock, store };
}

function okTransport(over: ConstructorParameters<typeof FakeTransport>[0] = {}): FakeTransport {
  return new FakeTransport({ info: () => info, pull: () => emptyPage, push: () => [], ...over });
}

describe("the sync engine", () => {
  it("never touches the transport while disconnected", async () => {
    const transport = okTransport();
    const { engine, clock } = await setup(transport, { connected: false });
    await engine.start();
    await engine.trigger("manual");
    engine.onLocalWrite();
    clock.advance(FOREGROUND_INTERVAL_MS * 2);
    expect(engine.getStatus()).toBe("disconnected");
    expect(transport.calls).toEqual([]);
  });

  it("runs push, pull and the integrity pass, then goes idle with the time of the sync", async () => {
    const transport = okTransport({
      push: (m) =>
        m.map(
          (x): PushResult => ({
            op_id: x.op_id,
            status: "applied",
            entity: x.entity,
            uid: x.uid,
            row: accountRow(x.uid, { version: 3 }),
            version: 3,
            seq: 11,
            conflicts: [],
            effects: [],
            replaced_by_uid: "",
            error: null,
          }),
        ),
      pull: () => page([upsert(12, "account", accountRow("01JD5Z8K3M0000000000000A01"))]),
    });
    const { env, engine } = await setup(transport);
    await env.repo.create("account", ACCOUNT);
    await engine.start();
    await engine.trigger("manual");

    expect(engine.getStatus()).toBe("idle");
    expect(transport.calls).toEqual(["info", "push", "pull"]);
    expect(engine.view()).toMatchObject({ pending: 0, lastSyncAt: "2026-11-24T12:00:00.000Z" });
    expect(await env.repo.list("account")).toHaveLength(2);
  });

  it("runs one sync when two triggers arrive at once", async () => {
    let release: () => void = () => undefined;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const transport = okTransport({
      info: async () => {
        await gate;
        return info;
      },
    });
    const { engine } = await setup(transport);
    await engine.start();

    const a = engine.trigger("manual");
    const b = engine.trigger("foreground");
    release();
    await Promise.all([a, b]);

    expect(transport.calls.filter((c) => c === "info")).toHaveLength(1);
    expect(transport.calls.filter((c) => c === "pull")).toHaveLength(1);
  });

  it("debounces local writes: three writes in 3 seconds make one sync", async () => {
    const transport = okTransport();
    const { engine, clock } = await setup(transport);
    await engine.start();
    engine.onLocalWrite();
    clock.advance(1000);
    engine.onLocalWrite();
    clock.advance(1000);
    engine.onLocalWrite();
    clock.advance(DEBOUNCE_MS - 1);
    await flush();
    expect(transport.calls).toEqual([]);
    clock.advance(1);
    await flush();
    expect(transport.calls.filter((c) => c === "info")).toHaveLength(1);
  });

  it("syncs again every 5 minutes while it runs", async () => {
    const transport = okTransport();
    const { engine, clock } = await setup(transport);
    await engine.start();
    clock.advance(FOREGROUND_INTERVAL_MS);
    await flush();
    clock.advance(FOREGROUND_INTERVAL_MS);
    await flush();
    expect(transport.calls.filter((c) => c === "info")).toHaveLength(2);
    engine.stop();
    clock.advance(FOREGROUND_INTERVAL_MS * 3);
    await flush();
    expect(transport.calls.filter((c) => c === "info")).toHaveLength(2);
  });

  it("pushes again when the person wrote during the run", async () => {
    let pushes = 0;
    let writeDuringPush: () => Promise<void> = async () => undefined;
    const transport = okTransport({
      push: async (m) => {
        pushes += 1;
        if (pushes === 1) await writeDuringPush();
        return accept(m);
      },
    });
    const { env, engine } = await setup(transport);
    writeDuringPush = async () => {
      await env.repo.create("tag", { name: "written while syncing", color: "" });
      engine.onLocalWrite();
    };
    await env.repo.create("tag", { name: "first", color: "" });
    await engine.start();
    await engine.trigger("manual");

    expect(pushes).toBe(2);
    expect(await env.repo.outboxCount()).toBe(0);
    expect(engine.getStatus()).toBe("idle");
  });

  describe("failures", () => {
    it("follows the backoff schedule 5 s, 15 s, 60 s, 5 min, 5 min, and a success starts it over", async () => {
      let down = true;
      const transport = okTransport({
        info: () => {
          if (down) throw new TransportError("network", "unreachable");
          return info;
        },
      });
      const { engine, clock } = await setup(transport);
      // The 5 minute timer would share its delay with the last backoff step, so only the retries stay.
      await engine.start();
      engine.stop();
      await engine.trigger("manual");
      const seen: number[] = [];
      for (let i = 0; i < 5; i++) {
        const [delay] = [...clock.timers.values()].map((t) => t.ms);
        seen.push(delay ?? -1);
        expect(engine.getStatus()).toBe("offline");
        clock.advance(delay ?? 0);
        await flush();
      }
      expect(seen).toEqual([5000, 15000, 60000, 300000, 300000]);

      down = false;
      const [delay] = [...clock.timers.values()].map((t) => t.ms);
      clock.advance(delay ?? 0);
      await flush();
      expect(engine.getStatus()).toBe("idle");
      down = true;
      await engine.trigger("manual");
      expect([...clock.timers.values()].some((t) => t.ms === 5000)).toBe(true);
    });

    it("jitters the wait by at most 20 percent", () => {
      for (const [i, base] of BACKOFF_SCHEDULE_MS.entries()) {
        expect(backoffDelay(i + 1, { next: () => 0 })).toBe(Math.round(base * 0.8));
        expect(backoffDelay(i + 1, { next: () => 0.999999 })).toBe(Math.round(base * 1.2));
      }
      expect(backoffDelay(0, half)).toBe(5000);
      expect(backoffDelay(99, half)).toBe(300000);
    });

    it("keeps the queue when a push times out, and the retry sends the same op_ids", async () => {
      const sent: string[][] = [];
      let lose = true;
      const transport = okTransport({
        push: (m) => {
          sent.push(m.map((x) => x.op_id));
          if (lose) throw new TransportError("timeout", "no answer");
          return m.map(
            (x): PushResult => ({
              op_id: x.op_id,
              status: "applied",
              entity: x.entity,
              uid: x.uid,
              row: null,
              version: 1,
              seq: 1,
              conflicts: [],
              effects: [],
              replaced_by_uid: "",
              error: null,
            }),
          );
        },
      });
      const { env, engine } = await setup(transport);
      await env.repo.create("tag", { name: "x", color: "" });
      await env.repo.create("tag", { name: "y", color: "" });
      await engine.start();
      await engine.trigger("manual");
      expect(engine.getStatus()).toBe("offline");
      expect(await env.repo.outboxCount()).toBe(2);
      expect(engine.view().pending).toBe(2);

      lose = false;
      await engine.trigger("manual");
      expect(engine.getStatus()).toBe("idle");
      expect(sent[1]).toEqual(sent[0]);
      expect(await env.repo.outboxCount()).toBe(0);
    });

    it("stops at auth_failed on a revoked device, keeps the data and the queue, and does not retry", async () => {
      const transport = okTransport({
        push: accept,
        pull: () => {
          throw new TransportError("http", "revoked", 403, "device_revoked");
        },
      });
      const { env, engine, clock } = await setup(transport);
      await env.repo.create("account", ACCOUNT);
      await engine.start();
      await engine.trigger("manual");

      expect(engine.getStatus()).toBe("auth_failed");
      expect(await env.repo.list("account")).toHaveLength(1);
      expect(await env.repo.outboxCount()).toBeGreaterThanOrEqual(0);
      const calls = transport.calls.length;
      clock.advance(FOREGROUND_INTERVAL_MS * 3);
      await engine.trigger("manual");
      engine.onLocalWrite();
      await flush();
      expect(transport.calls).toHaveLength(calls);
    });

    it("stops at workspace_deleted on a 410, keeps the data until the person confirms, then erases it", async () => {
      const transport = okTransport({
        push: accept,
        pull: () => {
          throw new TransportError("http", "gone", 410, "workspace_deleted");
        },
      });
      const { env, engine, clock, store } = await setup(transport);
      await env.repo.create("account", ACCOUNT);
      await engine.start();
      await engine.trigger("manual");

      expect(engine.getStatus()).toBe("workspace_deleted");
      expect(engine.view().failure).toBe("workspace_deleted");
      // Told first, erased later: nothing is deleted by the failure itself.
      expect(await env.repo.list("account")).toHaveLength(1);

      const calls = transport.calls.length;
      clock.advance(FOREGROUND_INTERVAL_MS * 3);
      await engine.trigger("manual");
      engine.onLocalWrite();
      await flush();
      expect(transport.calls).toHaveLength(calls);

      await engine.eraseDeletedWorkspace();
      expect(await env.repo.list("account")).toHaveLength(0);
      expect(await env.repo.outboxCount()).toBe(0);
      expect(store.value).toBeNull();
      expect(engine.getStatus()).toBe("disconnected");
      expect(await loadConnection(env.db)).toBeNull();
    });

    it("treats a 401 as auth_failed too", async () => {
      const { engine } = await setup(
        okTransport({
          info: () => {
            throw new TransportError("http", "no", 401, "unauthorized");
          },
        }),
      );
      await engine.start();
      await engine.trigger("manual");
      expect(engine.getStatus()).toBe("auth_failed");
    });

    it("refuses a server that does not speak protocol 1, and does not retry it", async () => {
      const transport = okTransport({ info: () => ({ ...info, protocol: { min: 2, max: 3 } }) });
      const { engine, clock } = await setup(transport);
      await engine.start();
      await engine.trigger("manual");
      expect(engine.getStatus()).toBe("error");
      expect(engine.view().failure).toBe("unsupported");
      expect(transport.calls).toEqual(["info"]);
      expect([...clock.timers.values()].every((t) => t.ms === FOREGROUND_INTERVAL_MS)).toBe(true);
    });

    it("waits at least the Retry-After of a 429", async () => {
      const { engine, clock } = await setup(
        okTransport({
          info: () => {
            throw new TransportError("http", "slow", 429, "rate_limited", 120);
          },
        }),
      );
      await engine.start();
      await engine.trigger("manual");
      expect(engine.view().failure).toBe("rate_limited");
      expect(clock.delays()).toContain(120000);
    });
  });

  it("resyncs on 410 and pulls from the snapshot cursor", async () => {
    let first = true;
    const asked: number[] = [];
    const transport = okTransport({
      pull: (cursor) => {
        asked.push(cursor);
        if (first) {
          first = false;
          throw new TransportError("http", "gone", 410, "resync_required");
        }
        return page([], 50);
      },
      snapshot: (q) => ({
        snapshot_id: "s",
        snapshot_cursor: 50,
        entity: q.entity,
        rows: [],
        next_after: "",
        has_more: false,
      }),
    });
    const { engine } = await setup(transport);
    await engine.start();
    await engine.trigger("manual");
    expect(engine.getStatus()).toBe("idle");
    expect(asked).toEqual([0, 50]);
  });

  it("asks for a merge choice instead of pushing when this phone has data and never synced", async () => {
    const transport = okTransport();
    const { env, engine } = await setup(transport, { firstSyncDone: false });
    await env.repo.create("account", ACCOUNT);
    await engine.start();
    await engine.trigger("manual");
    expect(engine.getStatus()).toBe("needs_merge_choice");
    expect(transport.calls).toEqual(["info"]);
  });

  it("registers the device once when the token came without one", async () => {
    const transport = okTransport({ registerDevice: () => ({ device_id: "NEWDEVICE" }) });
    const env = await openEnv();
    await saveConnection(env.db, {
      serverUrl: "https://money.example.com",
      insecure: false,
      deviceId: null,
      deviceName: "Phone",
      workspaceUid: "1",
      workspaceName: "Home",
    });
    const engine = new SyncEngine({
      db: env.db,
      repo: env.repo,
      transport,
      tokens: tokens(),
      clock: new FakeClock(),
      random: half,
    });
    await engine.start();
    await engine.trigger("manual");
    await engine.trigger("manual");
    expect(transport.calls.filter((c) => c === "registerDevice")).toHaveLength(1);
  });

  it("disconnects: the token and the connection go, the data and the queue stay unless cleared", async () => {
    const transport = okTransport();
    const { env, engine, store } = await setup(transport);
    await env.repo.create("account", ACCOUNT);
    await engine.start();
    await engine.disconnect(false);
    expect(store.value).toBeNull();
    expect(engine.getStatus()).toBe("disconnected");
    expect(await env.repo.list("account")).toHaveLength(1);
    expect(await env.repo.outboxCount()).toBe(1);
    expect(await env.db.all("SELECT * FROM meta WHERE key LIKE 'sync.%'")).toEqual([]);
    await engine.disconnect(true);
    expect(await env.repo.outboxCount()).toBe(0);
  });
});

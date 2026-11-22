import { createHash, randomBytes } from "node:crypto";
import { DatabaseSync } from "node:sqlite";
import { ENTITIES, type EntityName } from "../../src/db/entities";
import { connectToServer } from "../../src/sync/connect";
import { HttpTransport, type TokenStore } from "../../src/sync/transport";
import type {
  DevicePlatform,
  Mutation,
  PairResult,
  PullPage,
  PushResult,
  ServerInfo,
  SnapshotPage,
  SnapshotQuery,
  Transport,
} from "../../src/sync/types";
import type { ConvergenceEnv } from "../helpers/convergence";
import { openEnv } from "../helpers/fixtures";
import type { Harness, Recorder, Role } from "../helpers/scenario-runner";
import { type NodeResponse, nodeFetch } from "./node-fetch";

/**
 * The real server, over real HTTP. scripts/mobile-sync-e2e.sh starts it on a free
 * port with a throwaway SQLite file and a seeded workspace, and gives these two
 * variables to the Jest run.
 */
export const SERVER_URL = process.env.SYNC_E2E_URL ?? "";
export const SERVER_DB = process.env.SYNC_E2E_DB ?? "";
const ADMIN_EMAIL = process.env.SYNC_E2E_ADMIN_EMAIL ?? "";
const ADMIN_PASSWORD = process.env.SYNC_E2E_ADMIN_PASSWORD ?? "";

/** The test must never fall back to a default address: it writes to whatever it is pointed at. */
export function requireServer(): void {
  if (SERVER_URL === "" || SERVER_DB === "" || ADMIN_EMAIL === "") {
    throw new Error("run this suite through scripts/mobile-sync-e2e.sh: it starts the server and sets SYNC_E2E_*");
  }
}

/** Signs in on the web like a person and presses "Add device". */
class WebSession {
  private readonly jar = new Map<string, string>();

  private cookies(): string {
    return [...this.jar].map(([k, v]) => `${k}=${v}`).join("; ");
  }

  private take(response: NodeResponse): void {
    for (const line of response.headers.getSetCookie()) {
      const [pair = ""] = line.split(";");
      const eq = pair.indexOf("=");
      if (eq > 0) this.jar.set(pair.slice(0, eq).trim(), pair.slice(eq + 1).trim());
    }
  }

  private async send(
    path: string,
    init: { method: string; headers?: Record<string, string>; body?: string },
  ): Promise<NodeResponse> {
    const response = await nodeFetch(`${SERVER_URL}${path}`, {
      ...init,
      headers: {
        Cookie: this.cookies(),
        Origin: SERVER_URL,
        "Sec-Fetch-Site": "same-origin",
        ...init.headers,
      },
    });
    this.take(response);
    return response;
  }

  async login(): Promise<void> {
    await this.send("/login", { method: "GET" });
    const response = await this.send("/login", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }).toString(),
    });
    if (response.status !== 303) throw new Error(`sign-in answered ${response.status}`);
  }

  /** The Devices page action: the code is read from the page that shows it once. */
  async addDevice(): Promise<string> {
    const response = await this.send("/devices/pairing", {
      method: "POST",
      headers: { "X-Inertia": "true", "X-Requested-With": "XMLHttpRequest" },
    });
    const body = (await response.json()) as { props?: { pairing?: { code?: string } } };
    const code = body.props?.pairing?.code;
    if (response.status !== 200 || code === undefined) throw new Error(`Add device answered ${response.status}`);
    return code;
  }

  async revokeDevice(deviceId: string): Promise<void> {
    const response = await this.send(`/devices/${deviceId}`, {
      method: "DELETE",
      headers: { "X-Inertia": "true", "X-Requested-With": "XMLHttpRequest" },
    });
    if (response.status >= 400) throw new Error(`revoking a device answered ${response.status}`);
  }
}

let addressCounter = 10;

/** One transport per device. The pairing limit is per address, so each device claims its own (the server trusts 127.0.0.1 as a proxy in this run). */
export interface FaultState {
  dropNextPush: boolean;
  /** The bodies of every push the server received, to compare what a retry sent. */
  pushBodies: string[];
}

function forwardedFetch(address: string, state: FaultState): typeof fetch {
  return (async (input: string, init: RequestInit) => {
    const headers = { ...(init.headers as Record<string, string>), "X-Forwarded-For": address };
    if (String(input).endsWith("/push")) state.pushBodies.push(String(init.body));
    const response = await nodeFetch(input, { ...init, headers } as never);
    if (state.dropNextPush && String(input).endsWith("/push")) {
      // The server processed the push and the answer never arrived.
      state.dropNextPush = false;
      await response.text();
      throw new TypeError("connection reset");
    }
    return response;
  }) as unknown as typeof fetch;
}

class RecordingTransport implements Recorder {
  readonly pushes: PushResult[][] = [];
  readonly pulls: PullPage[] = [];
  constructor(
    private readonly inner: Transport,
    readonly state: FaultState,
  ) {}

  info(): Promise<ServerInfo> {
    return this.inner.info();
  }

  registerDevice(name: string, platform: DevicePlatform): Promise<{ device_id: string }> {
    return this.inner.registerDevice(name, platform);
  }

  async pull(cursor: number, limit: number): Promise<PullPage> {
    const page = await this.inner.pull(cursor, limit);
    this.pulls.push(page);
    return page;
  }

  snapshot(q: SnapshotQuery): Promise<SnapshotPage> {
    return this.inner.snapshot(q);
  }

  async push(mutations: readonly Mutation[]): Promise<PushResult[]> {
    const results = await this.inner.push(mutations);
    this.pushes.push(results);
    return results;
  }

  pair(code: string, name: string, platform: DevicePlatform): Promise<PairResult> {
    return this.inner.pair(code, name, platform);
  }
}

const ENTITY_TABLES = [
  "transaction_tags",
  "transactions",
  "credit_cards",
  "recurrences",
  "budgets",
  "goal_contributions",
  "goals",
  "categories",
  "accounts",
  "tags",
];
const SYNC_TABLES = ["changes", "sync_ops", "sync_conflicts", "sync_state"];

export class RealServer {
  private readonly web = new WebSession();
  private readonly deviceIds = new Map<string, string>();
  private readonly sql = new DatabaseSync(SERVER_DB);
  private reader: RecordingTransport | null = null;
  private pairedOnce = false;

  constructor() {
    this.sql.exec("PRAGMA busy_timeout = 10000");
  }

  async start(): Promise<void> {
    requireServer();
    await this.web.login();
  }

  close(): void {
    this.sql.close();
  }

  /** An empty workspace: the rows, the change log and the stored results go; the account and its devices stay. */
  async wipe(): Promise<void> {
    this.sql.exec("PRAGMA foreign_keys = OFF");
    for (const t of ENTITY_TABLES) this.sql.exec(`DELETE FROM ${t}`);
    for (const t of SYNC_TABLES) this.sql.exec(`DELETE FROM ${t}`);
    this.deviceIds.clear();
    this.reader = null;
  }

  async newDevice(name: string): Promise<RecordingTransport> {
    addressCounter += 1;
    const address = `10.77.${Math.floor(addressCounter / 250)}.${(addressCounter % 250) + 1}`;
    const state: FaultState = { dropNextPush: false, pushBodies: [] };
    const fetchImpl = forwardedFetch(address, state);
    let token: string | null = null;
    const tokens: TokenStore = {
      get: async () => token,
      set: async (t) => {
        token = t;
      },
      clear: async () => {
        token = null;
      },
    };
    let deviceId: string | null = null;
    const make = (): HttpTransport =>
      new HttpTransport({
        baseUrl: SERVER_URL,
        allowInsecure: true,
        tokens,
        deviceId: () => deviceId,
        fetchImpl,
      });
    const env = await openEnv();
    // The first device of a run presses "Add device" on the web; the rest take a code made the same way
    // (the hash of 8 characters, 10 minutes), because that page allows 60 requests a minute and a run pairs hundreds.
    const code = this.pairedOnce ? this.mintCode() : await this.web.addDevice();
    this.pairedOnce = true;
    const result = await connectToServer(
      { db: env.db, tokens, makeTransport: () => make() },
      { serverUrl: SERVER_URL, confirmedInsecure: true, code, deviceName: name, platform: "android" },
    );
    if (!result.ok) throw new Error(`pairing failed: ${result.reason}`);
    const meta = await env.db.all<{ value: string }>("SELECT value FROM meta WHERE key = 'sync.device_id'");
    deviceId = meta[0]?.value ?? null;
    if (deviceId === null) throw new Error("the pairing gave no device");
    this.deviceIds.set(name, deviceId);
    return new RecordingTransport(make(), state);
  }

  private mintCode(): string {
    const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ";
    const bytes = randomBytes(8);
    let code = "";
    for (let i = 0; i < 8; i++) code += alphabet[(bytes[i] ?? 0) & 31];
    const owner = this.sql.prepare("SELECT workspace_id, user_id FROM memberships LIMIT 1").all()[0] as
      | { workspace_id: number; user_id: number }
      | undefined;
    if (owner === undefined) throw new Error("the seeded workspace has no member");
    // The format the Go driver writes, so the server's comparisons of times read it the same way.
    const stamp = (d: Date): string => `${d.toISOString().replace("T", " ").replace("Z", "")}0000 +0000 UTC`;
    const now = new Date();
    this.sql
      .prepare(
        "INSERT INTO pairing_codes (workspace_id, user_id, code_hash, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
      )
      .run(
        owner.workspace_id,
        owner.user_id,
        createHash("sha256").update(code).digest("hex"),
        stamp(new Date(now.getTime() + 600_000)),
        stamp(now),
        stamp(now),
      );
    return `${code.slice(0, 4)}-${code.slice(4)}`;
  }

  async revokeDevice(name: string): Promise<void> {
    const id = this.deviceIds.get(name);
    if (id === undefined) throw new Error(`no device ${name}`);
    await this.web.revokeDevice(id);
  }

  compact(): void {
    this.sql.exec(`INSERT INTO sync_state (workspace_id, compacted_through_seq, created_at, updated_at)
      SELECT w.id, COALESCE((SELECT MAX(seq) FROM changes WHERE workspace_id = w.id), 0), datetime('now'), datetime('now')
      FROM workspaces w WHERE true
      ON CONFLICT (workspace_id) DO UPDATE SET compacted_through_seq = excluded.compacted_through_seq`);
  }

  ageTombstones(): void {
    const old = new Date(Date.now() - 31 * 24 * 3600 * 1000).toISOString().replace("T", " ").replace("Z", " +0000 UTC");
    for (const t of ["accounts", "categories", "tags", "transactions", "budgets", "credit_cards", "recurrences"]) {
      this.sql.prepare(`UPDATE ${t} SET deleted_at = ? WHERE deleted_at IS NOT NULL`).run(old);
    }
  }

  /** The live rows of the server, read through a device of its own, as a phone would download them. */
  async liveRows(
    entities: readonly EntityName[] = Object.keys(ENTITIES) as EntityName[],
    fresh = false,
  ): Promise<Map<EntityName, Map<string, Record<string, unknown>>>> {
    // A device has 60 pulls and snapshots a minute: a check that asks often takes a reader of its own.
    if (fresh) this.reader = null;
    this.reader ??= await this.newDevice("reader");
    const transport = this.reader;
    const out = new Map<EntityName, Map<string, Record<string, unknown>>>();
    let snapshotId = "";
    for (const entity of entities) {
      const rows = new Map<string, Record<string, unknown>>();
      let after = "";
      for (;;) {
        const page = await transport.snapshot({ entity, after, snapshot_id: snapshotId, limit: 500 });
        snapshotId = page.snapshot_id;
        for (const row of page.rows) {
          rows.set(
            String(row.uid),
            Object.fromEntries(Object.keys(ENTITIES[entity].fields).map((n) => [n, row[n] ?? null])),
          );
        }
        if (!page.has_more) break;
        after = page.next_after;
      }
      out.set(entity, rows);
    }
    return out;
  }
}

export function scenarioHarness(server: RealServer): Harness {
  return {
    transportFor: async (device: string, _role: Role) => server.newDevice(device),
    revoke: (device) => server.revokeDevice(device),
    compact: async () => server.compact(),
    ageTombstones: async () => server.ageTombstones(),
    live: async (entity) => (await server.liveRows([entity], true)).get(entity) ?? new Map(),
  };
}

export function convergenceEnv(server: RealServer): ConvergenceEnv {
  let counter = 0;
  return {
    newTransport: (name) => {
      counter += 1;
      return server.newDevice(`${name}-${counter}`);
    },
    serverRows: () => server.liveRows(),
  };
}

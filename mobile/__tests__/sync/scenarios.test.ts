import { readdirSync, readFileSync } from "node:fs";
import type { Database } from "../../src/db/database";
import { ENTITIES, type EntityName } from "../../src/db/entities";
import { toRow } from "../../src/db/repository";
import { pullUntilDone } from "../../src/sync/apply";
import { SyncEngine } from "../../src/sync/engine";
import { resync } from "../../src/sync/resync";
import { getCursor, resumeEntities, saveConnection } from "../../src/sync/state";
import { type Clock, type PullChange, type PushResult, TransportError } from "../../src/sync/types";
import { type Env, openEnv } from "../helpers/fixtures";
import { ReferenceServer, type ReferenceTransport, type Role } from "../helpers/reference-server";

const DIR = `${__dirname}/../../../testdata/sync`;
const STAMP = "2026-11-24T12:00:00.000Z";
const now = (): string => STAMP;

interface Step {
  device: string;
  action: "push" | "pull" | "snapshot" | "admin";
  input?: { mutations?: Record<string, unknown>[]; from?: string; do?: string; target?: string };
  expect?: Record<string, unknown>;
}

interface Scenario {
  name: string;
  workspace: { plan: "pro" | "free"; devices: Record<string, Role> };
  steps: Step[];
  final?: {
    devices_equal?: string[];
    server?: Record<string, Record<string, Record<string, unknown>>>;
    absent?: string[];
  };
}

const idle: Clock = { now: () => Date.parse(STAMP), setTimeout: () => 0, clearTimeout: () => undefined };

interface Device {
  env: Env;
  engine: SyncEngine;
  transport: ReferenceTransport;
  seen: Map<string, number>;
  seenPushes: number;
  seenPulls: number;
}

async function startDevice(server: ReferenceServer, id: string): Promise<Device> {
  const env = await openEnv();
  const transport = server.transportFor(id);
  await saveConnection(env.db, {
    serverUrl: "https://reference.example",
    insecure: false,
    deviceId: id,
    deviceName: id,
    workspaceUid: "1",
    workspaceName: "Reference",
  });
  await env.db.run("INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1')");
  const tokens = { get: async () => null, set: async () => undefined, clear: async () => undefined };
  const engine = new SyncEngine({
    db: env.db,
    repo: env.repo,
    transport,
    tokens,
    clock: idle,
    random: { next: () => 0.5 },
  });
  await engine.start();
  engine.stop();
  return { env, engine, transport, seen: new Map(), seenPushes: 0, seenPulls: 0 };
}

/** What a device has seen of each row, from its own results and its pulls: the meaning of base_seq "$seen". */
function learn(d: Device): void {
  for (const page of d.transport.pulls.slice(d.seenPulls)) {
    for (const c of page.changes) d.seen.set(`${c.entity}/${c.uid}`, c.seq);
  }
  for (const results of d.transport.pushes.slice(d.seenPushes)) {
    for (const r of results) {
      if (r.seq > 0) d.seen.set(`${r.entity}/${r.uid}`, r.seq);
      for (const e of r.effects) d.seen.set(`${e.entity}/${e.uid}`, e.seq);
    }
  }
  d.seenPulls = d.transport.pulls.length;
  d.seenPushes = d.transport.pushes.length;
}

/** The shared matching rule: objects by the keys listed, arrays by length and position, null matches null or missing. */
function subset(actual: unknown, wanted: unknown, path: string): string[] {
  if (wanted === null)
    return actual === null || actual === undefined ? [] : [`${path}: want null, got ${JSON.stringify(actual)}`];
  if (Array.isArray(wanted)) {
    if (!Array.isArray(actual) || actual.length !== wanted.length)
      return [`${path}: want ${wanted.length} items, got ${JSON.stringify(actual)}`];
    return wanted.flatMap((w, i) => subset(actual[i], w, `${path}[${i}]`));
  }
  if (typeof wanted === "object") {
    if (typeof actual !== "object" || actual === null)
      return [`${path}: want an object, got ${JSON.stringify(actual)}`];
    return Object.entries(wanted as Record<string, unknown>).flatMap(([k, w]) =>
      subset((actual as Record<string, unknown>)[k], w, `${path}.${k}`),
    );
  }
  return actual === wanted ? [] : [`${path}: want ${JSON.stringify(wanted)}, got ${JSON.stringify(actual)}`];
}

async function insertOutbox(db: Database, m: Record<string, unknown>, d: Device): Promise<void> {
  const base = m.base_seq === "$seen" ? (d.seen.get(`${m.entity}/${m.uid}`) ?? 0) : (m.base_seq as number);
  await db.run(
    "INSERT INTO outbox (op_id, entity, uid, op, base_seq, fields, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
    [
      m.op_id as string,
      m.entity as string,
      m.uid as string,
      m.op as string,
      base,
      JSON.stringify(m.fields ?? {}),
      STAMP,
    ],
  );
}

async function localLive(db: Database, entity: EntityName): Promise<Map<string, Record<string, unknown>>> {
  const out = new Map<string, Record<string, unknown>>();
  const rows = await db.all<Record<string, string | number | null>>(
    `SELECT * FROM ${ENTITIES[entity].table} WHERE deleted_at IS NULL`,
  );
  for (const r of rows) {
    const row = toRow(entity, r) as unknown as Record<string, unknown>;
    const fields: Record<string, unknown> = {};
    for (const name of Object.keys(ENTITIES[entity].fields)) fields[name] = row[name];
    out.set(r.uid as string, fields);
  }
  return out;
}

async function run(s: Scenario): Promise<string[]> {
  const server = new ReferenceServer({ plan: s.workspace.plan, devices: s.workspace.devices });
  const devices = new Map<string, Device>();
  for (const id of Object.keys(s.workspace.devices)) devices.set(id, await startDevice(server, id));
  const problems: string[] = [];

  for (const [i, step] of s.steps.entries()) {
    const d = devices.get(step.device);
    if (d === undefined) throw new Error(`step ${i}: no device ${step.device}`);
    const where = `step ${i + 1} (${step.device} ${step.action})`;
    const want = step.expect ?? {};
    let error: TransportError | null = null;
    let results: PushResult[] = [];
    let changes: PullChange[] = [];
    let hasMore = false;
    const pushedBefore = d.transport.pushes.length;
    const pulledBefore = d.transport.pulls.length;

    try {
      if (step.action === "push") {
        const seen = new Set<string>();
        for (const m of step.input?.mutations ?? []) {
          if (seen.has(m.op_id as string)) continue; // the client never sends one op twice in a batch
          seen.add(m.op_id as string);
          await insertOutbox(d.env.db, m, d);
        }
        // Each push step is a "Sync now": it tries a quota-paused entity again.
        await resumeEntities(d.env.db);
        await d.engine.pushAll();
      } else if (step.action === "pull") {
        const from = step.input?.from === "start" ? 0 : await getCursor(d.env.db);
        await pullUntilDone(d.env.db, d.transport, now, from);
      } else if (step.action === "snapshot") {
        await resync(d.env.db, d.transport, now);
      } else if (step.input?.do === "revoke_device") {
        server.revoke(step.input.target ?? step.device);
      } else if (step.input?.do === "compact") {
        server.compact();
      } else if (step.input?.do === "age_tombstones") {
        server.ageTombstones();
      }
    } catch (e) {
      if (!(e instanceof TransportError)) throw e;
      error = e;
    }

    results = d.transport.pushes.slice(pushedBefore).flat();
    changes = d.transport.pulls.slice(pulledBefore).flatMap((p) => p.changes);
    hasMore = d.transport.pulls.slice(pulledBefore).at(-1)?.has_more ?? false;
    learn(d);

    const status = error?.status ?? 200;
    if (step.action !== "admin" || "status" in want) {
      problems.push(...subset({ status }, { status: (want.status as number | undefined) ?? 200 }, where));
    }
    if (want.code !== undefined) problems.push(...subset({ code: error?.code }, { code: want.code }, where));

    if (step.action === "push" && Array.isArray(want.results)) {
      // A repeated op_id answers with the stored result, so the expectation list is matched by op_id order of the file.
      const ordered = (step.input?.mutations ?? []).map((m) => results.find((r) => r.op_id === m.op_id) ?? null);
      problems.push(...subset(ordered, want.results, `${where} results`));
    }
    if (step.action === "pull" && error === null) {
      if (Array.isArray(want.changes)) problems.push(...subset(changes, want.changes, `${where} changes`));
      if (want.has_more !== undefined)
        problems.push(...subset({ has_more: hasMore }, { has_more: want.has_more }, where));
      for (const c of (want.contains as Record<string, unknown>[] | undefined) ?? []) {
        const found = changes.find((x) => x.uid === c.uid);
        problems.push(
          ...(found === undefined
            ? [`${where}: ${String(c.uid)} was not pulled`]
            : subset(found, c, `${where} ${String(c.uid)}`)),
        );
      }
      for (const uid of (want.absent_uids as string[] | undefined) ?? []) {
        if (changes.some((x) => x.uid === uid)) problems.push(`${where}: ${uid} must not be pulled`);
      }
      if (want.parents_first === true) {
        const order = ["account", "category", "tag", "credit_card", "recurrence", "budget", "transaction"];
        const ranks = changes.map((c) => order.indexOf(c.entity));
        if (ranks.some((r, k) => k > 0 && r < (ranks[k - 1] ?? 0)))
          problems.push(`${where}: parents do not come first`);
      }
    }
    if (step.action === "snapshot" && want.replica_contains !== undefined) {
      for (const [entity, uids] of Object.entries(want.replica_contains as Record<string, string[]>)) {
        const local = await localLive(d.env.db, entity as EntityName);
        for (const uid of uids) if (!local.has(uid)) problems.push(`${where}: replica lacks ${entity} ${uid}`);
      }
    }
  }

  const fin = s.final ?? {};
  for (const id of fin.devices_equal ?? []) {
    const d = devices.get(id);
    if (d === undefined) continue;
    await d.engine.trigger("manual");
    for (const entity of Object.keys(ENTITIES) as EntityName[]) {
      const local = await localLive(d.env.db, entity);
      const remote = server.live(entity);
      const remoteFields = new Map(
        [...remote].map(([uid, row]) => [
          uid,
          Object.fromEntries(Object.keys(ENTITIES[entity].fields).map((n) => [n, row[n]])),
        ]),
      );
      if (JSON.stringify([...local.keys()].sort()) !== JSON.stringify([...remoteFields.keys()].sort())) {
        problems.push(
          `final: device ${id} holds ${[...local.keys()].length} ${entity} rows, the server ${remoteFields.size}`,
        );
        continue;
      }
      for (const [uid, row] of remoteFields) {
        if (JSON.stringify(local.get(uid)) !== JSON.stringify(row))
          problems.push(`final: device ${id} differs on ${entity} ${uid}`);
      }
    }
  }
  for (const [entity, rows] of Object.entries(fin.server ?? {})) {
    const live = server.live(entity as EntityName);
    for (const [uid, wanted] of Object.entries(rows))
      problems.push(...subset(live.get(uid) ?? null, wanted, `final server ${entity} ${uid}`));
  }
  for (const uid of fin.absent ?? []) {
    for (const entity of Object.keys(ENTITIES) as EntityName[]) {
      if (server.live(entity).has(uid)) problems.push(`final: ${uid} must not be live on the server`);
    }
  }
  return problems;
}

const files = readdirSync(DIR)
  .filter((f) => f.endsWith(".json"))
  .sort();

describe("the shared sync scenarios, run through the client engine", () => {
  it("finds the scenario files", () => {
    expect(files.length).toBeGreaterThanOrEqual(13);
  });

  it.each(files)("%s", async (file) => {
    const scenario = JSON.parse(readFileSync(`${DIR}/${file}`, "utf8")) as Scenario;
    expect(await run(scenario)).toEqual([]);
  });
});

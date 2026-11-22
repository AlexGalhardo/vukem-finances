import { ENTITIES, type EntityName } from "../../src/db/entities";
import { toRow } from "../../src/db/repository";
import { SyncEngine } from "../../src/sync/engine";
import { saveConnection } from "../../src/sync/state";
import type { Clock, Transport } from "../../src/sync/types";
import { type Env, openEnv } from "./fixtures";

/**
 * The property of the epic: after any interleaving of edits, deletes, restores and
 * offline periods on three devices, full syncs leave every device with exactly the
 * live rows of the server. The three actors stand for two phones and the web.
 */

export interface ConvergenceEnv {
  /** A transport for one more device of the same workspace. */
  newTransport(name: string): Promise<Transport>;
  /** The live rows of the server, by entity and uid, with the fields the client stores. */
  serverRows(): Promise<Map<EntityName, Map<string, Record<string, unknown>>>>;
}

export interface ConvergenceReport {
  seed: number;
  steps: number;
  differences: string[];
  /** Conflicts the devices recorded: "kind:entity.field" to a count. */
  conflicts: Record<string, number>;
  problems: string[];
}

const clock: Clock = {
  now: () => Date.parse("2026-11-24T12:00:00.000Z"),
  setTimeout: () => 0,
  clearTimeout: () => undefined,
};

/** A small seeded generator, so a failing run is repeated by its seed. */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

interface Actor {
  name: string;
  env: Env;
  engine: SyncEngine;
}

async function actor(name: string, transport: Transport): Promise<Actor> {
  const env = await openEnv();
  await saveConnection(env.db, {
    serverUrl: "https://reference.example",
    insecure: false,
    deviceId: name,
    deviceName: name,
    workspaceUid: "1",
    workspaceName: "Reference",
  });
  await env.db.run("INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1')");
  const tokens = { get: async () => null, set: async () => undefined, clear: async () => undefined };
  const engine = new SyncEngine({ db: env.db, repo: env.repo, transport, tokens, clock, random: { next: () => 0.5 } });
  await engine.start();
  engine.stop();
  return { name, env, engine };
}

async function liveRows(a: Actor, entity: EntityName): Promise<Map<string, Record<string, unknown>>> {
  const raws = await a.env.db.all<Record<string, string | number | null>>(
    `SELECT * FROM ${ENTITIES[entity].table} WHERE deleted_at IS NULL`,
  );
  const out = new Map<string, Record<string, unknown>>();
  for (const raw of raws) {
    const row = toRow(entity, raw) as unknown as Record<string, unknown>;
    out.set(raw.uid as string, Object.fromEntries(Object.keys(ENTITIES[entity].fields).map((n) => [n, row[n]])));
  }
  return out;
}

const pick = <T>(rand: () => number, list: readonly T[]): T | undefined => list[Math.floor(rand() * list.length)];

export async function runConvergence(seed: number, env: ConvergenceEnv): Promise<ConvergenceReport> {
  const rand = mulberry32(seed);
  const actors = [
    await actor("A", await env.newTransport("A")),
    await actor("B", await env.newTransport("B")),
    await actor("W", await env.newTransport("W")),
  ] as const;
  const report: ConvergenceReport = { seed, steps: 0, differences: [], conflicts: {}, problems: [] };
  const [first] = actors;

  // A shared starting point: one account and a few categories, known to everyone.
  const account = await first.env.repo.create("account", {
    name: `account-${seed}`,
    kind: "checking",
    opening_cents: 0,
    archived_at: null,
    position: 0,
  });
  for (const n of [1, 2, 3]) {
    await first.env.repo.create("category", {
      name: `cat-${seed}-${n}`,
      kind: "expense",
      parent_uid: null,
      color: "",
      icon: "",
      archived_at: null,
    });
  }
  for (const a of actors) await a.engine.trigger("manual");

  let counter = 0;
  const steps = 14 + Math.floor(rand() * 12);
  for (let i = 0; i < steps; i++) {
    const who = pick(rand, actors) as Actor;
    const repo = who.env.repo;
    const roll = rand();
    const txs = await repo.list("transaction");
    const cats = await repo.list("category");
    report.steps += 1;
    try {
      if (roll < 0.3) {
        counter += 1;
        const cat = rand() < 0.7 ? pick(rand, cats) : undefined;
        await repo.create("transaction", {
          account_uid: account.uid,
          category_uid: cat?.uid ?? null,
          kind: "expense",
          amount_cents: 100 + Math.floor(rand() * 9000),
          description: `${who.name}-${seed}-${counter}`,
          occurred_on: `2026-11-${String(1 + Math.floor(rand() * 20)).padStart(2, "0")}`,
          transfer_uid: null,
          tag_uids: [],
          statement_ref: null,
          recurrence_uid: null,
          occurrence_on: null,
          installment_group_uid: null,
          installment_no: null,
          installment_total: null,
        });
      } else if (roll < 0.55) {
        const tx = pick(rand, txs);
        if (tx !== undefined) {
          const change: Record<string, unknown> = {};
          if (rand() < 0.6) {
            counter += 1;
            change.description = `${who.name}-edit-${counter}`;
          }
          if (rand() < 0.5) change.amount_cents = 100 + Math.floor(rand() * 9000);
          if (rand() < 0.3) change.category_uid = pick(rand, cats)?.uid ?? null;
          await repo.update("transaction", tx.uid, change);
        }
      } else if (roll < 0.65) {
        const tx = pick(rand, txs);
        if (tx !== undefined) await repo.remove("transaction", tx.uid);
      } else if (roll < 0.7) {
        const gone = (await repo.list("transaction", true)).filter((t) => t.deleted_at !== null);
        const tx = pick(rand, gone);
        if (tx !== undefined) await repo.restore("transaction", tx.uid);
      } else if (roll < 0.75) {
        counter += 1;
        await repo.create("category", {
          name: `cat-${who.name}-${seed}-${counter}`,
          kind: "expense",
          parent_uid: null,
          color: "",
          icon: "",
          archived_at: null,
        });
      } else if (roll < 0.8) {
        const cat = pick(rand, cats);
        counter += 1;
        if (cat !== undefined)
          await repo.update("category", cat.uid, { name: `renamed-${who.name}-${seed}-${counter}` });
      } else {
        await who.engine.trigger("manual");
      }
    } catch (error) {
      report.problems.push(
        `seed ${seed}: ${who.name} step ${i}: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  // Full syncs, in rounds, until nobody waits for anything.
  for (let round = 0; round < 8; round++) {
    for (const a of actors) await a.engine.trigger("manual");
    const views = await Promise.all(actors.map((a) => a.engine.refresh()));
    if (round >= 2 && views.every((v) => v.pending === 0 && v.status === "idle")) break;
  }

  const truth = await env.serverRows();
  for (const a of actors) {
    const view = await a.engine.refresh();
    if (view.status !== "idle")
      report.problems.push(
        `seed ${seed}: ${a.name} ended ${view.status} (${view.failure ?? "no failure"}: ${view.failureMessage ?? ""})`,
      );
    for (const p of await a.env.db.all<{ entity: string; code: string; message: string }>(
      "SELECT entity, code, message FROM sync_problems WHERE dismissed_at IS NULL",
    )) {
      report.problems.push(`seed ${seed}: ${a.name} has a problem ${p.entity} ${p.code} ${p.message}`);
    }
    for (const c of await a.env.db.all<{ kind: string; entity: string; field: string; n: number }>(
      "SELECT kind, entity, field, COUNT(*) AS n FROM sync_conflicts GROUP BY kind, entity, field",
    )) {
      const key = `${c.kind}:${c.entity}.${c.field}`;
      report.conflicts[key] = (report.conflicts[key] ?? 0) + c.n;
    }
    for (const entity of Object.keys(ENTITIES) as EntityName[]) {
      const local = await liveRows(a, entity);
      const remote = truth.get(entity) ?? new Map();
      const onlyLocal = [...local.keys()].filter((u) => !remote.has(u));
      const onlyRemote = [...remote.keys()].filter((u) => !local.has(u));
      if (onlyLocal.length > 0)
        report.differences.push(`seed ${seed}: ${a.name} holds ${entity} ${onlyLocal.join(",")} the server does not`);
      if (onlyRemote.length > 0)
        report.differences.push(`seed ${seed}: ${a.name} lacks ${entity} ${onlyRemote.join(",")}`);
      for (const [uid, row] of local) {
        const other = remote.get(uid);
        if (other !== undefined && JSON.stringify(row) !== JSON.stringify(other)) {
          report.differences.push(
            `seed ${seed}: ${a.name} differs on ${entity} ${uid}: ${JSON.stringify(row)} vs ${JSON.stringify(other)}`,
          );
        }
      }
    }
  }
  return report;
}

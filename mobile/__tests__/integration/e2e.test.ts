import { SyncEngine } from "../../src/sync/engine";
import { saveConnection } from "../../src/sync/state";
import type { Clock } from "../../src/sync/types";
import { runConvergence } from "../helpers/convergence";
import { openEnv } from "../helpers/fixtures";
import { loadScenario, runScenario, scenarioFiles } from "../helpers/scenario-runner";
import { convergenceEnv, RealServer, scenarioHarness } from "./http-harness";

/**
 * Run by scripts/mobile-sync-e2e.sh against the real Go server over HTTP. It is not
 * part of `bunx jest`: it needs the server and its two SYNC_E2E_* variables.
 */

const server = new RealServer();
const clock: Clock = {
  now: () => Date.parse("2026-11-24T12:00:00.000Z"),
  setTimeout: () => 0,
  clearTimeout: () => undefined,
};

beforeAll(async () => {
  await server.start();
});

afterAll(() => {
  server.close();
});

/** One owner of a Pro workspace is all this run has: the Free plan and another role need the Go runner. */
const NEEDS_OTHER_SETUP = ["10-quota-exceeded.json", "11-viewer-push-refused.json"];

describe("the real server, over HTTP", () => {
  const runnable = scenarioFiles().filter((f) => !NEEDS_OTHER_SETUP.includes(f));

  it("runs every scenario that owners of a Pro workspace can run, and names the two it cannot", () => {
    const owners = (f: string): boolean => {
      const s = loadScenario(f);
      return s.workspace.plan === "pro" && Object.values(s.workspace.devices).every((r) => r === "owner");
    };
    expect(scenarioFiles().filter((f) => !owners(f))).toEqual(NEEDS_OTHER_SETUP);
    expect(runnable.length).toBe(scenarioFiles().length - 2);
  });

  it.each(runnable)("scenario %s", async (file) => {
    await server.wipe();
    expect(await runScenario(loadScenario(file), scenarioHarness(server))).toEqual([]);
  });

  it("retries a push whose response was lost with the same op_ids and creates no duplicate", async () => {
    await server.wipe();
    const transport = await server.newDevice("phone");
    const env = await openEnv();
    await saveConnection(env.db, {
      serverUrl: "http://example.invalid",
      insecure: false,
      deviceId: "phone",
      deviceName: "phone",
      workspaceUid: "1",
      workspaceName: "Home",
    });
    await env.db.run("INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1')");
    const tokens = { get: async () => null, set: async () => undefined, clear: async () => undefined };
    const engine = new SyncEngine({
      db: env.db,
      repo: env.repo,
      transport,
      tokens,
      clock,
      random: { next: () => 0.5 },
    });
    await engine.start();
    engine.stop();

    await env.repo.create("category", {
      name: "Lost answer",
      kind: "expense",
      parent_uid: null,
      color: "",
      icon: "",
      archived_at: null,
    });
    await env.repo.create("tag", { name: "retry", color: "" });

    transport.state.dropNextPush = true;
    await engine.trigger("manual");
    expect(engine.getStatus()).toBe("offline");
    expect(await env.repo.outboxCount()).toBe(2);

    await engine.trigger("manual");
    expect(engine.getStatus()).toBe("idle");
    expect(await env.repo.outboxCount()).toBe(0);

    const ids = transport.state.pushBodies.map((b) =>
      (JSON.parse(b) as { mutations: { op_id: string }[] }).mutations.map((m) => m.op_id),
    );
    expect(ids).toHaveLength(2);
    expect(ids[1]).toEqual(ids[0]);

    const rows = await server.liveRows();
    expect([...(rows.get("category")?.values() ?? [])].map((c) => c.name)).toEqual(["Lost answer"]);
    expect([...(rows.get("tag")?.values() ?? [])].map((t) => t.name)).toEqual(["retry"]);
  });

  it("converges two phones and a third client to the server after 50 seeded runs", async () => {
    const conflicts: Record<string, number> = {};
    for (let seed = 1; seed <= 50; seed++) {
      await server.wipe();
      const report = await runConvergence(seed, convergenceEnv(server));
      expect({ seed, differences: report.differences, problems: report.problems }).toEqual({
        seed,
        differences: [],
        problems: [],
      });
      for (const [k, n] of Object.entries(report.conflicts)) conflicts[k] = (conflicts[k] ?? 0) + n;
    }
    console.log(`conflicts seen in 50 runs: ${JSON.stringify(conflicts)}`);
    expect(Object.keys(conflicts).length).toBeGreaterThan(0);
  }, 900_000);
});

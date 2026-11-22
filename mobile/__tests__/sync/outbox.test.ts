import { BATCH_SIZE, markAttempt, nextBatch, pendingFor } from "../../src/sync/outbox";
import { pauseEntity } from "../../src/sync/state";
import { ACCOUNT, openEnv } from "../helpers/fixtures";

describe("the sync outbox", () => {
  it("sends the entries in creation order with the fields that changed", async () => {
    const { repo, db } = await openEnv();
    const a = await repo.create("account", ACCOUNT);
    await repo.create("tag", { name: "trip", color: "" });
    await repo.update("account", a.uid, { name: "Wallet" });
    await repo.remove("account", a.uid);

    const { mutations } = await nextBatch(db);
    expect(mutations.map((m) => [m.entity, m.op, Object.keys(m.fields).join()])).toEqual([
      ["account", "upsert", "name,kind,opening_cents,archived_at,position"],
      ["tag", "upsert", "name,color"],
      ["account", "upsert", "name"],
      ["account", "delete", ""],
    ]);
    expect(mutations[0]?.base_seq).toBe(0);
  });

  it("rebuilds the same batch after a timeout: the op_ids do not change", async () => {
    const env = await openEnv();
    for (let i = 0; i < 5; i++) {
      await env.repo.create("tag", { name: `t${i}`, color: "" });
    }
    const first = await nextBatch(env.db);
    await markAttempt(
      env.db,
      first.entries.map((e) => e.id),
    );
    await env.repo.create("tag", { name: "written while the request was lost", color: "" });
    const second = await nextBatch(env.db);

    expect(second.mutations.slice(0, 5).map((m) => m.op_id)).toEqual(first.mutations.map((m) => m.op_id));
    expect(second.mutations).toHaveLength(6);
    expect(new Set(second.mutations.map((m) => m.op_id)).size).toBe(6);
    expect(second.entries[0]?.attempts).toBe(1);
  });

  it("takes at most 200 and leaves the rest for the next batch", async () => {
    const env = await openEnv();
    for (let i = 0; i < BATCH_SIZE + 7; i++) {
      await env.repo.create("tag", { name: `t${i}`, color: "" });
    }
    const batch = await nextBatch(env.db);
    expect(batch.mutations).toHaveLength(BATCH_SIZE);
    expect(batch.entries.map((e) => e.id)).toEqual([...batch.entries.map((e) => e.id)].sort((a, b) => a - b));
    expect(await env.repo.outboxCount()).toBe(BATCH_SIZE + 7);
  });

  it("skips the entities a quota refusal paused", async () => {
    const env = await openEnv();
    await env.repo.create("account", ACCOUNT);
    await env.repo.create("tag", { name: "x", color: "" });
    await pauseEntity(env.db, "account", "The Free plan has one account");
    expect((await nextBatch(env.db)).mutations.map((m) => m.entity)).toEqual(["tag"]);
  });

  it("tells which fields and which lifecycle change wait for a row", async () => {
    const env = await openEnv();
    const t = await env.repo.create("tag", { name: "x", color: "" });
    await env.repo.update("tag", t.uid, { color: "red" });
    expect(await pendingFor(env.db, "tag", t.uid)).toEqual({
      fields: new Set(["name", "color"]),
      lifecycle: null,
      any: true,
    });
    await env.repo.remove("tag", t.uid);
    expect((await pendingFor(env.db, "tag", t.uid)).lifecycle).toBe("delete");
    expect(await pendingFor(env.db, "tag", "nothing")).toEqual({ fields: new Set(), lifecycle: null, any: false });
  });
});

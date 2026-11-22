import { resync } from "../../src/sync/resync";
import { getCursor, setCursor } from "../../src/sync/state";
import type { SnapshotPage } from "../../src/sync/types";
import { ACCOUNT, openEnv } from "../helpers/fixtures";
import { FakeTransport } from "./fake-transport";
import { accountRow, categoryRow, tagRow } from "./wire";

const NOW = (): string => "2026-11-24T12:00:00.000Z";
const A1 = "01JD5Z8K3M0000000000000A01";
const A2 = "01JD5Z8K3M0000000000000A02";
const C1 = "01JD5Z8K3M0000000000000C01";
const G1 = "01JD5Z8K3M0000000000000G01";

function snapshotOf(byEntity: Record<string, Record<string, unknown>[]>, cursor: number, pageSize = 500) {
  return (q: { entity: string; after: string; snapshot_id: string }): SnapshotPage => {
    const all = [...(byEntity[q.entity] ?? [])].sort((a, b) => String(a.uid).localeCompare(String(b.uid)));
    const rows = all.filter((r) => String(r.uid) > q.after).slice(0, pageSize);
    const last = rows[rows.length - 1];
    const more = last !== undefined && all.some((r) => String(r.uid) > String(last.uid));
    return {
      snapshot_id: "s1",
      snapshot_cursor: cursor,
      entity: q.entity,
      rows,
      next_after: more && last !== undefined ? String(last.uid) : "",
      has_more: more,
    };
  };
}

describe("resync", () => {
  it("replaces every clean row with the snapshot, sets the cursor and leaves nothing staged", async () => {
    const env = await openEnv();
    const stale = await env.repo.create("account", ACCOUNT);
    await env.db.run("UPDATE accounts SET dirty = 0, server_seq = 3 WHERE uid = ?", [stale.uid]);
    await env.repo.clearOutbox();
    await setCursor(env.db, 2);

    const transport = new FakeTransport({
      snapshot: snapshotOf({ account: [accountRow(A1, { name: "Fresh" })], category: [categoryRow(C1)] }, 40),
    });
    expect(await resync(env.db, transport, NOW)).toBe(40);

    expect((await env.repo.list("account")).map((a) => [a.uid, a.name])).toEqual([[A1, "Fresh"]]);
    expect(await env.repo.list("category")).toHaveLength(1);
    expect(await getCursor(env.db)).toBe(40);
    expect(await env.db.all("SELECT * FROM sync_staging")).toEqual([]);
    expect(await env.repo.get("account", A1)).toMatchObject({ dirty: false, server_seq: 40 });
  });

  it("keeps dirty rows and the whole outbox, merging the pending fields over the snapshot", async () => {
    const env = await openEnv();
    const mine = await env.repo.create("account", { ...ACCOUNT, name: "Offline only" });
    const edited = await env.repo.create("account", { ...ACCOUNT, name: "Old" }, A1);
    await env.db.run("UPDATE accounts SET dirty = 0 WHERE uid = ?", [A1]);
    await env.repo.update("account", A1, { name: "My edit" });
    const queued = (await env.repo.outbox()).length;

    const transport = new FakeTransport({
      snapshot: snapshotOf({ account: [accountRow(A1, { name: "Server", kind: "checking", version: 7 })] }, 90),
    });
    await resync(env.db, transport, NOW);

    expect(edited.uid).toBe(A1);
    expect(await env.repo.get("account", mine.uid)).toMatchObject({ name: "Offline only", dirty: true });
    expect(await env.repo.get("account", A1)).toMatchObject({
      name: "My edit",
      kind: "checking",
      dirty: true,
      version: 7,
    });
    expect(await env.repo.outboxCount()).toBe(queued);
  });

  it("drops what the snapshot no longer holds (a purged tombstone, a row deleted elsewhere)", async () => {
    const env = await openEnv();
    const t = await env.repo.create("tag", { name: "gone", color: "" });
    await env.db.run("UPDATE tags SET dirty = 0 WHERE uid = ?", [t.uid]);
    await env.repo.clearOutbox();
    await resync(env.db, new FakeTransport({ snapshot: snapshotOf({ tag: [tagRow(G1)] }, 5) }), NOW);
    expect((await env.repo.list("tag", true)).map((x) => x.uid)).toEqual([G1]);
  });

  it("downloads several pages and changes nothing when the download fails half way", async () => {
    const env = await openEnv();
    const old = await env.repo.create("account", ACCOUNT);
    await env.db.run("UPDATE accounts SET dirty = 0 WHERE uid = ?", [old.uid]);
    await env.repo.clearOutbox();

    const rows = [accountRow(A1), accountRow(A2)];
    const paged = snapshotOf({ account: rows }, 8, 1);
    let calls = 0;
    const failing = new FakeTransport({
      snapshot: (q) => {
        calls += 1;
        if (q.entity === "category") throw new Error("connection lost");
        return paged(q);
      },
    });
    await expect(resync(env.db, failing, NOW)).rejects.toThrow("connection lost");
    expect(calls).toBeGreaterThan(2);
    expect((await env.repo.list("account")).map((a) => a.uid)).toEqual([old.uid]);
    expect(await getCursor(env.db)).toBe(0);

    await resync(env.db, new FakeTransport({ snapshot: paged }), NOW);
    expect((await env.repo.list("account")).map((a) => a.uid)).toEqual([A1, A2]);
  });
});

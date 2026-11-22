import { type Batch, nextBatch } from "../../src/sync/outbox";
import { handleResults, listProblems } from "../../src/sync/results";
import { pausedEntities } from "../../src/sync/state";
import type { PushResult } from "../../src/sync/types";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";
import { accountRow, tagRow } from "./wire";

const NOW = "2026-11-24T12:00:00.000Z";
const CATEGORY = { name: "Food", kind: "expense", parent_uid: null, color: "", icon: "", archived_at: null };

function result(batch: Batch, i: number, over: Partial<PushResult>): PushResult {
  const e = batch.entries[i];
  if (e === undefined) throw new Error("no such entry");
  return {
    op_id: e.op_id,
    status: "applied",
    entity: e.entity,
    uid: e.uid,
    row: null,
    version: 1,
    seq: 1,
    conflicts: [],
    effects: [],
    replaced_by_uid: "",
    error: null,
    ...over,
  };
}

const rejected = (batch: Batch, i: number, code: string, message = ""): PushResult =>
  result(batch, i, { status: "rejected", error: { code, message, field: "" } });

async function oneAccount(env: Env): Promise<{ uid: string; batch: Batch }> {
  const a = await env.repo.create("account", ACCOUNT);
  return { uid: a.uid, batch: await nextBatch(env.db) };
}

describe("handling push results", () => {
  it("stores the returned row, clears dirty and removes the entry", async () => {
    const env = await openEnv();
    const { uid, batch } = await oneAccount(env);
    const summary = await handleResults(
      env.db,
      batch,
      [result(batch, 0, { row: accountRow(uid, { name: "Server name", version: 4 }), version: 4, seq: 12 })],
      NOW,
    );

    expect(summary.applied).toBe(1);
    expect(await env.repo.get("account", uid)).toMatchObject({
      name: "Server name",
      version: 4,
      server_seq: 12,
      dirty: false,
    });
    expect(await env.repo.outboxCount()).toBe(0);
  });

  it("keeps the row dirty when a newer edit waits, and rebases that edit onto the accepted seq", async () => {
    const env = await openEnv();
    const { uid, batch } = await oneAccount(env);
    await env.repo.update("account", uid, { name: "Second thoughts" });

    await handleResults(
      env.db,
      batch,
      [
        result(batch, 0, {
          row: accountRow(uid, { name: "Checking", kind: "checking", version: 3 }),
          version: 3,
          seq: 20,
        }),
      ],
      NOW,
    );

    expect(await env.repo.get("account", uid)).toMatchObject({ name: "Second thoughts", dirty: true, server_seq: 20 });
    const left = await env.repo.outbox();
    expect(left).toHaveLength(1);
    expect(left[0]).toMatchObject({ base_seq: 20, fields: { name: "Second thoughts" } });
  });

  it("records the conflicts of a merged result", async () => {
    const env = await openEnv();
    const { uid, batch } = await oneAccount(env);
    const summary = await handleResults(
      env.db,
      batch,
      [
        result(batch, 0, {
          status: "merged",
          row: accountRow(uid, { name: "Theirs" }),
          seq: 5,
          conflicts: [{ field: "name", kind: "field", lost_value: "Mine", applied_value: "Theirs" }],
        }),
      ],
      NOW,
    );
    expect(summary).toMatchObject({ merged: 1, conflicts: 1 });
    expect(await env.db.all("SELECT field, kind, lost_value, applied_value FROM sync_conflicts")).toEqual([
      { field: "name", kind: "field", lost_value: '"Mine"', applied_value: '"Theirs"' },
    ]);
  });

  it("applies a delete result as a tombstone", async () => {
    const env = await openEnv();
    const a = await env.repo.create("account", ACCOUNT);
    await handleResults(env.db, await nextBatch(env.db), [], NOW).catch(() => undefined);
    await env.repo.clearOutbox();
    await env.repo.remove("account", a.uid);
    const batch = await nextBatch(env.db);
    await handleResults(env.db, batch, [result(batch, 0, { row: null, version: 2, seq: 9 })], NOW);
    expect(await env.repo.get("account", a.uid, true)).toMatchObject({ dirty: false, server_seq: 9 });
    expect(await env.repo.outboxCount()).toBe(0);
  });

  it("re-keys the row and every reference to it when the server says it is a duplicate", async () => {
    const env = await openEnv();
    const mine = await env.repo.create("category", CATEGORY);
    const child = await env.repo.create("category", { ...CATEGORY, name: "Snacks", parent_uid: mine.uid });
    const account = await env.repo.create("account", ACCOUNT);
    const tx = await env.repo.create("transaction", txFields(account.uid, { category_uid: mine.uid }));
    const budget = await env.repo.create("budget", { category_uid: mine.uid, limit_cents: 500, rollover: false });
    const batch = await nextBatch(env.db);
    const serverUid = "01JD5Z8K3M0000000000000C77";

    const summary = await handleResults(
      env.db,
      batch,
      batch.entries.map((e, i) =>
        e.uid === mine.uid
          ? result(batch, i, { status: "duplicate", replaced_by_uid: serverUid })
          : result(batch, i, { seq: 3 + i, row: null }),
      ),
      NOW,
    );

    expect(summary).toMatchObject({ duplicates: 1, rekeyed: 1 });
    expect(await env.repo.get("category", mine.uid)).toBeNull();
    expect(await env.repo.get("category", serverUid)).toMatchObject({ name: "Food" });
    expect((await env.repo.get("category", child.uid))?.parent_uid).toBe(serverUid);
    expect((await env.repo.get("transaction", tx.uid))?.category_uid).toBe(serverUid);
    expect((await env.repo.get("budget", budget.uid))?.category_uid).toBe(serverUid);
    expect((await env.repo.outbox()).some((o) => o.uid === mine.uid)).toBe(false);
  });

  it("re-keys the queued mutations that point at the replaced row", async () => {
    const env = await openEnv();
    const mine = await env.repo.create("category", CATEGORY);
    const account = await env.repo.create("account", ACCOUNT);
    const first = await nextBatch(env.db);
    await env.repo.create("transaction", txFields(account.uid, { category_uid: mine.uid }));

    await handleResults(
      env.db,
      first,
      first.entries.map((e, i) =>
        e.uid === mine.uid
          ? result(first, i, { status: "duplicate", replaced_by_uid: "01JD5Z8K3M0000000000000C78" })
          : result(first, i, { row: null, seq: 2 }),
      ),
      NOW,
    );
    expect((await env.repo.outbox()).map((o) => o.fields.category_uid)).toEqual(["01JD5Z8K3M0000000000000C78"]);
  });

  it("retries a missing reference after the next pull, twice, and then lists it as a problem", async () => {
    const env = await openEnv();
    const { uid } = await oneAccount(env);
    for (const expectedLeft of [1, 1, 0]) {
      const batch = await nextBatch(env.db);
      await handleResults(env.db, batch, [rejected(batch, 0, "unknown_reference")], NOW);
      expect(await env.repo.outboxCount()).toBe(expectedLeft);
    }
    expect(await listProblems(env.db)).toMatchObject([{ uid, code: "unknown_reference", op: "upsert" }]);
    expect((await env.repo.get("account", uid))?.dirty).toBe(false);
  });

  it("removes a permanent rejection from the queue and keeps what was lost", async () => {
    const env = await openEnv();
    const { uid, batch } = await oneAccount(env);
    const summary = await handleResults(env.db, batch, [rejected(batch, 0, "unknown_field", "no such field")], NOW);
    expect(summary.problems).toBe(1);
    expect(await env.repo.outboxCount()).toBe(0);
    const [problem] = await listProblems(env.db);
    expect(problem).toMatchObject({ uid, code: "unknown_field", message: "no such field" });
    expect(problem?.fields).toMatchObject({ name: "Checking" });
  });

  it("pauses only the entity the quota refused and keeps its entries", async () => {
    const env = await openEnv();
    await env.repo.create("account", ACCOUNT);
    await env.repo.create("tag", { name: "x", color: "" });
    const batch = await nextBatch(env.db);
    const summary = await handleResults(
      env.db,
      batch,
      [
        rejected(batch, 0, "quota_exceeded", "The Free plan has one account."),
        result(batch, 1, { row: tagRow(batch.entries[1]?.uid ?? "") }),
      ],
      NOW,
    );

    expect(summary.paused.get("account")).toBe("The Free plan has one account.");
    expect([...(await pausedEntities(env.db)).keys()]).toEqual(["account"]);
    expect(await env.repo.outboxCount()).toBe(1);
    expect((await nextBatch(env.db)).mutations).toEqual([]);
  });

  it("drops an edit of a row the server deleted and records the conflict", async () => {
    const env = await openEnv();
    const { uid, batch } = await oneAccount(env);
    await handleResults(env.db, batch, [rejected(batch, 0, "edited_deleted_row")], NOW);
    expect(await env.repo.outboxCount()).toBe(0);
    expect(await listProblems(env.db)).toEqual([]);
    expect(uid).not.toBe("");
  });

  it("treats the two legs of a transfer refused together as permanent, and a lone leg as retryable", async () => {
    const env = await openEnv();
    const account = await env.repo.create("account", ACCOUNT);
    const other = await env.repo.create("account", { ...ACCOUNT, name: "Savings" });
    const group = "01JD5Z8K3M0000000000000X01";
    await env.repo.create("transaction", txFields(account.uid, { transfer_uid: group }));
    await env.repo.create("transaction", txFields(other.uid, { kind: "income", transfer_uid: group }));
    const pair = await nextBatch(env.db);
    await handleResults(
      env.db,
      pair,
      pair.entries.map((_, i) =>
        i < 2 ? result(pair, i, { row: null, seq: i + 1 }) : rejected(pair, i, "transfer_incomplete"),
      ),
      NOW,
    );
    expect(await listProblems(env.db)).toHaveLength(2);

    const lone = await openEnv();
    const a = await lone.repo.create("account", ACCOUNT);
    await lone.repo.create("transaction", txFields(a.uid, { transfer_uid: group }));
    const batch = await nextBatch(lone.db);
    await handleResults(
      lone.db,
      batch,
      batch.entries.map((_, i) =>
        i === 0 ? result(batch, i, { row: null }) : rejected(batch, i, "transfer_incomplete"),
      ),
      NOW,
    );
    expect(await listProblems(lone.db)).toEqual([]);
    expect(await lone.repo.outboxCount()).toBe(1);
  });

  it("refuses an answer that does not match the batch and changes nothing", async () => {
    const env = await openEnv();
    await env.repo.create("account", ACCOUNT);
    await env.repo.create("tag", { name: "x", color: "" });
    const batch = await nextBatch(env.db);
    const first = batch.entries[0]?.uid ?? "";

    await expect(
      handleResults(env.db, batch, [result(batch, 0, { row: accountRow(first), seq: 4 })], NOW),
    ).rejects.toThrow("different number");
    await expect(
      handleResults(
        env.db,
        batch,
        [result(batch, 0, { row: accountRow(first), seq: 4 }), { ...result(batch, 1, {}), op_id: "other" }],
        NOW,
      ),
    ).rejects.toThrow("another order");
    expect(await env.repo.outboxCount()).toBe(2);
    expect((await env.repo.get("account", first))?.server_seq).toBeNull();
  });
});

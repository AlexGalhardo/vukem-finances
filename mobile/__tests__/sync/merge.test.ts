import { connectToServer } from "../../src/sync/connect";
import { describeReplace, hasLocalData, mergeWithServer, replaceWithServer } from "../../src/sync/merge";
import { getCursor, loadConnection } from "../../src/sync/state";
import type { TokenStore } from "../../src/sync/transport";
import type { SnapshotPage } from "../../src/sync/types";
import { ACCOUNT, openEnv, txFields } from "../helpers/fixtures";
import { FakeTransport } from "./fake-transport";
import { accountRow, categoryRow, tagRow } from "./wire";

const NOW = (): string => "2026-11-24T12:00:00.000Z";
const SERVER_FOOD = "01JD5Z8K3M0000000000000C01";
const SERVER_WALLET = "01JD5Z8K3M0000000000000A01";
const SERVER_TAG = "01JD5Z8K3M0000000000000G01";
const CATEGORY = { name: "Food", kind: "expense", parent_uid: null, color: "", icon: "", archived_at: null };

function serverWith(byEntity: Record<string, Record<string, unknown>[]>, cursor = 70): FakeTransport {
  return new FakeTransport({
    snapshot: (q): SnapshotPage => ({
      snapshot_id: "s",
      snapshot_cursor: cursor,
      entity: q.entity,
      rows: byEntity[q.entity] ?? [],
      next_after: "",
      has_more: false,
    }),
  });
}

describe("the first connection", () => {
  it("merges equal names: two categories named alike become one, and the transactions follow the adopted uid", async () => {
    const env = await openEnv();
    const food = await env.repo.create("category", CATEGORY);
    const pets = await env.repo.create("category", { ...CATEGORY, name: "Pets" });
    const account = await env.repo.create("account", { ...ACCOUNT, name: "Wallet", kind: "cash" });
    const tag = await env.repo.create("tag", { name: "trip", color: "" });
    const tx = await env.repo.create(
      "transaction",
      txFields(account.uid, { category_uid: food.uid, tag_uids: [tag.uid] }),
    );

    const server = serverWith({
      account: [accountRow(SERVER_WALLET, { name: "Wallet", kind: "cash" })],
      category: [categoryRow(SERVER_FOOD)],
      tag: [tagRow(SERVER_TAG)],
    });
    const result = await mergeWithServer({ db: env.db, transport: server, now: NOW });

    expect(result.merged).toBe(3);
    expect((await env.repo.list("category")).map((c) => [c.uid, c.name]).sort()).toEqual([
      [SERVER_FOOD, "Food"],
      [pets.uid, "Pets"],
    ]);
    expect(await env.repo.list("account")).toHaveLength(1);
    expect(await env.repo.get("account", SERVER_WALLET)).toMatchObject({ dirty: false });
    expect(await env.repo.get("tag", SERVER_TAG)).not.toBeNull();
    expect(await env.repo.get("transaction", tx.uid)).toMatchObject({
      account_uid: SERVER_WALLET,
      category_uid: SERVER_FOOD,
      tag_uids: [SERVER_TAG],
    });
    expect(await getCursor(env.db)).toBe(70);
  });

  it("queues everything else as creates, once each, parents first, with no create for an adopted row", async () => {
    const env = await openEnv();
    env.repo.setKeepOutbox(false);
    await env.repo.create("category", CATEGORY);
    const pets = await env.repo.create("category", { ...CATEGORY, name: "Pets" });
    const account = await env.repo.create("account", ACCOUNT);
    await env.repo.create("transaction", txFields(account.uid, { category_uid: pets.uid }));
    expect(await env.repo.outboxCount()).toBe(0);

    await mergeWithServer({ db: env.db, transport: serverWith({ category: [categoryRow(SERVER_FOOD)] }), now: NOW });

    const queued = await env.repo.outbox();
    expect(queued.map((q) => [q.entity, q.op, q.base_seq])).toEqual([
      ["account", "upsert", 0],
      ["category", "upsert", 0],
      ["transaction", "upsert", 0],
    ]);
    expect(queued.find((q) => q.entity === "transaction")?.fields).toMatchObject({
      category_uid: pets.uid,
      amount_cents: 1000,
    });
    expect(new Set(queued.map((q) => q.op_id)).size).toBe(3);
  });

  it("replaces a partial queued edit of a never-synced row with one full create", async () => {
    const env = await openEnv();
    const a = await env.repo.create("account", ACCOUNT);
    await env.repo.update("account", a.uid, { name: "Renamed" });
    expect(await env.repo.outboxCount()).toBe(2);
    await mergeWithServer({ db: env.db, transport: serverWith({}), now: NOW });
    const queued = await env.repo.outbox();
    expect(queued).toHaveLength(1);
    expect(queued[0]?.fields).toMatchObject({ name: "Renamed", kind: "checking" });
  });

  it("keeps rows of both sides when the names differ", async () => {
    const env = await openEnv();
    await env.repo.create("category", { ...CATEGORY, name: "Mine" });
    await mergeWithServer({ db: env.db, transport: serverWith({ category: [categoryRow(SERVER_FOOD)] }), now: NOW });
    expect((await env.repo.list("category")).map((c) => c.name).sort()).toEqual(["Food", "Mine"]);
  });

  it("does not need a prompt on an empty phone", async () => {
    const env = await openEnv();
    expect(await hasLocalData(env.db)).toBe(false);
    await env.repo.create("tag", { name: "x", color: "" });
    expect(await hasLocalData(env.db)).toBe(true);
  });

  describe("Replace", () => {
    it("says what it will delete and needs the literal confirmation (the screen offers the export first)", async () => {
      const env = await openEnv();
      await env.repo.create("account", ACCOUNT);
      await env.repo.create("tag", { name: "x", color: "" });
      expect(await describeReplace(env.db)).toEqual({ rows: 2, pending: 2 });

      const server = serverWith({ tag: [tagRow(SERVER_TAG)] });
      await expect(replaceWithServer({ db: env.db, transport: server, now: NOW }, false as never)).rejects.toThrow(
        "confirmation",
      );
      expect(await env.repo.list("account")).toHaveLength(1);
    });

    it("deletes the local data and the queue and takes the server's", async () => {
      const env = await openEnv();
      await env.repo.create("account", ACCOUNT);
      await replaceWithServer({ db: env.db, transport: serverWith({ tag: [tagRow(SERVER_TAG)] }, 12), now: NOW }, true);
      expect(await env.repo.list("account")).toEqual([]);
      expect((await env.repo.list("tag")).map((t) => t.uid)).toEqual([SERVER_TAG]);
      expect(await env.repo.outboxCount()).toBe(0);
      expect(await getCursor(env.db)).toBe(12);
    });

    it("deletes nothing when the download fails", async () => {
      const env = await openEnv();
      await env.repo.create("account", ACCOUNT);
      const broken = new FakeTransport({
        snapshot: () => {
          throw new Error("connection lost");
        },
      });
      await expect(replaceWithServer({ db: env.db, transport: broken, now: NOW }, true)).rejects.toThrow("lost");
      expect(await env.repo.list("account")).toHaveLength(1);
      expect(await env.repo.outboxCount()).toBe(1);
    });
  });
});

describe("connecting", () => {
  const SECRET = ["tok", "en", "-", "from", "-", "pairing"].join("");
  const CODE = ["ABCD", "EFGH"].join("-");

  function store(): TokenStore & { value: string | null } {
    const s = {
      value: null as string | null,
      get: async () => s.value,
      set: async (t: string) => {
        s.value = t;
      },
      clear: async () => {
        s.value = null;
      },
    };
    return s;
  }

  const paired = new FakeTransport({
    pair: () => ({
      token: SECRET,
      device_id: "DEVICE9",
      workspace: { uid: "7", name: "Ana household", currency: "USD", timezone: "UTC" },
      server_time: "",
    }),
  });
  const input = {
    serverUrl: "https://money.example.com/",
    confirmedInsecure: false,
    code: CODE,
    deviceName: "Ana's phone",
    platform: "android" as const,
  };

  it("keeps the token only in secure storage and never writes the code anywhere", async () => {
    const env = await openEnv();
    const tokens = store();
    const built: [string, boolean][] = [];
    const result = await connectToServer(
      { db: env.db, tokens, makeTransport: (url, insecure) => (built.push([url, insecure]), paired) },
      input,
    );

    expect(result).toEqual({ ok: true, workspaceName: "Ana household" });
    expect(built).toEqual([["https://money.example.com", false]]);
    expect(tokens.value).toBe(SECRET);
    expect(await loadConnection(env.db)).toEqual({
      serverUrl: "https://money.example.com",
      insecure: false,
      deviceId: "DEVICE9",
      deviceName: "Ana's phone",
      workspaceUid: "7",
      workspaceName: "Ana household",
    });
    const everything = JSON.stringify(await env.db.all("SELECT * FROM meta"));
    expect(everything).not.toContain(SECRET);
    expect(everything).not.toContain(CODE);
  });

  it("refuses http to a public host, and asks for a confirmation before http to a private address", async () => {
    const env = await openEnv();
    const fresh = new FakeTransport({ pair: () => paired.handlers.pair?.("", "", "ios") as never });
    const deps = { db: env.db, tokens: store(), makeTransport: () => fresh };
    expect(await connectToServer(deps, { ...input, serverUrl: "http://money.example.com" })).toEqual({
      ok: false,
      reason: "public_http",
    });
    expect(await connectToServer(deps, { ...input, serverUrl: "http://192.168.1.20:3000" })).toEqual({
      ok: false,
      reason: "needs_insecure_confirmation",
    });
    expect(fresh.calls).toEqual([]);
    expect(
      await connectToServer(deps, { ...input, serverUrl: "http://192.168.1.20:3000", confirmedInsecure: true }),
    ).toMatchObject({ ok: true });
    expect((await loadConnection(env.db))?.insecure).toBe(true);
  });

  it("stores nothing when the pairing fails", async () => {
    const env = await openEnv();
    const tokens = store();
    const failing = new FakeTransport({
      pair: () => {
        throw new Error("invalid code");
      },
    });
    await expect(connectToServer({ db: env.db, tokens, makeTransport: () => failing }, input)).rejects.toThrow(
      "invalid code",
    );
    expect(tokens.value).toBeNull();
    expect(await loadConnection(env.db)).toBeNull();
  });
});

import type { Database, SqlValue } from "../../src/db/database";
import { applyPage } from "../../src/sync/apply";
import { getCursor } from "../../src/sync/state";
import { ACCOUNT, openEnv, txFields } from "../helpers/fixtures";
import { accountRow, categoryRow, page, tagRow, tombstone, transactionRow, upsert } from "./wire";

const NOW = "2026-11-24T12:00:00.000Z";

const A1 = "01JD5Z8K3M0000000000000A01";
const C1 = "01JD5Z8K3M0000000000000C01";
const C2 = "01JD5Z8K3M0000000000000C02";
const T1 = "01JD5Z8K3M0000000000000T01";
const G1 = "01JD5Z8K3M0000000000000G01";

async function dump(db: Database): Promise<unknown> {
  const out: Record<string, unknown> = {};
  for (const t of ["accounts", "categories", "tags", "transactions", "transaction_tags", "outbox", "sync_conflicts"]) {
    out[t] = await db.all(`SELECT * FROM ${t} ORDER BY 1, 2`);
  }
  out.cursor = await getCursor(db);
  return out;
}

describe("applying a pull page", () => {
  it("inserts new rows, replaces clean ones and saves the cursor", async () => {
    const { db, repo } = await openEnv();
    await applyPage(db, page([upsert(5, "account", accountRow(A1))]), NOW);
    expect(await repo.get("account", A1)).toMatchObject({ name: "Wallet", version: 1, server_seq: 5, dirty: false });

    await applyPage(db, page([upsert(9, "account", accountRow(A1, { name: "Purse", version: 2 }))]), NOW);
    expect(await repo.get("account", A1)).toMatchObject({ name: "Purse", version: 2, server_seq: 9, dirty: false });
    expect(await getCursor(db)).toBe(9);
    expect(await repo.outboxCount()).toBe(0);
  });

  it("keeps a pending local edit and takes the server values of the other fields", async () => {
    const { db, repo } = await openEnv();
    await applyPage(db, page([upsert(1, "account", accountRow(A1))]), NOW);
    await repo.update("account", A1, { name: "Mine" });

    await applyPage(
      db,
      page([upsert(4, "account", accountRow(A1, { name: "Theirs", kind: "checking", version: 2 }))]),
      NOW,
    );
    expect(await repo.get("account", A1)).toMatchObject({ name: "Mine", kind: "checking", dirty: true, server_seq: 4 });
    const queued = await repo.outbox();
    expect(queued).toHaveLength(1);
    expect(queued[0]).toMatchObject({ op: "upsert", fields: { name: "Mine" } });
  });

  it("lets a server delete win over a pending edit and records the conflict", async () => {
    const { db, repo } = await openEnv();
    await applyPage(
      db,
      page([upsert(1, "account", accountRow(A1)), upsert(2, "transaction", transactionRow(T1, A1))]),
      NOW,
    );
    await repo.update("transaction", T1, { description: "late edit", amount_cents: 5 });

    await applyPage(db, page([tombstone(7, "transaction", T1)]), NOW);

    expect(await repo.get("transaction", T1)).toBeNull();
    expect(await repo.get("transaction", T1, true)).toMatchObject({ dirty: false, server_seq: 7 });
    expect(await repo.outboxCount()).toBe(0);
    const conflicts = await db.all<{ field: string; kind: string; lost_value: string }>(
      "SELECT field, kind, lost_value FROM sync_conflicts ORDER BY field",
    );
    expect(conflicts).toEqual([
      { field: "amount_cents", kind: "edited_deleted_row", lost_value: "5" },
      { field: "description", kind: "edited_deleted_row", lost_value: '"late edit"' },
    ]);
  });

  it("deletes a clean row without a conflict and ignores a tombstone of a row it never had", async () => {
    const { db, repo } = await openEnv();
    await applyPage(db, page([upsert(1, "tag", tagRow(G1))]), NOW);
    await applyPage(db, page([tombstone(3, "tag", G1), tombstone(4, "tag", "01JD5Z8K3M0000000000000G09")]), NOW);
    expect(await repo.list("tag")).toEqual([]);
    expect(await repo.list("tag", true)).toHaveLength(1);
    expect(await db.all("SELECT * FROM sync_conflicts")).toEqual([]);
  });

  it("writes a row whose references arrive in a later page, and ignores entities it does not know", async () => {
    const { db, repo } = await openEnv();
    await applyPage(
      db,
      page(
        [
          upsert(8, "transaction", transactionRow(T1, A1, { tag_uids: [G1] })),
          upsert(9, "goal", { uid: "x", version: 1 }),
        ],
        9,
        true,
      ),
      NOW,
    );
    expect(await repo.get("transaction", T1)).toMatchObject({ account_uid: A1, tag_uids: [G1] });
    expect(await db.all("SELECT * FROM transaction_tags")).toEqual([{ transaction_uid: T1, tag_uid: G1 }]);
    await applyPage(db, page([upsert(10, "account", accountRow(A1))]), NOW);
    expect(await getCursor(db)).toBe(10);
  });

  it("gives a never-synced local row the uid of the server row that holds the same name", async () => {
    const { db, repo } = await openEnv();
    const local = await repo.create("category", {
      name: "Food",
      kind: "expense",
      parent_uid: null,
      color: "",
      icon: "",
      archived_at: null,
    });
    const spend = await repo.create("account", ACCOUNT);
    const tx = await repo.create("transaction", txFields(spend.uid, { category_uid: local.uid }));

    await applyPage(db, page([upsert(3, "category", categoryRow(C1))]), NOW);

    expect((await repo.list("category")).map((c) => c.uid)).toEqual([C1]);
    expect((await repo.get("transaction", tx.uid))?.category_uid).toBe(C1);
    const queued = await repo.outbox();
    expect(queued.some((q) => q.entity === "category")).toBe(false);
    expect(queued.find((q) => q.entity === "transaction")?.fields.category_uid).toBe(C1);
  });

  it("parks a synced row that collides for a moment until its own change arrives", async () => {
    const { db, repo } = await openEnv();
    await applyPage(db, page([upsert(1, "category", categoryRow(C1, { name: "Food" }))]), NOW);
    await applyPage(
      db,
      page([
        upsert(5, "category", categoryRow(C2, { name: "Food" })),
        upsert(6, "category", categoryRow(C1, { name: "Meals", version: 2 })),
      ]),
      NOW,
    );
    expect((await repo.list("category")).map((c) => [c.uid, c.name]).sort()).toEqual([
      [C1, "Meals"],
      [C2, "Food"],
    ]);
  });

  describe("a crash between the page and its cursor", () => {
    const changes = [
      upsert(1, "account", accountRow(A1)),
      upsert(2, "category", categoryRow(C1)),
      upsert(3, "transaction", transactionRow(T1, A1, { category_uid: C1 })),
    ];

    function crashOnCursor(db: Database): Database {
      return new Proxy(db, {
        get(target, prop) {
          if (prop === "run") {
            return async (sql: string, params?: readonly SqlValue[]): Promise<void> => {
              if (params?.[0] === "sync.cursor") {
                throw new Error("the app was killed");
              }
              return target.run(sql, params);
            };
          }
          const value = Reflect.get(target, prop, target) as unknown;
          return typeof value === "function" ? (value as (...a: unknown[]) => unknown).bind(target) : value;
        },
      });
    }

    it("rolls the whole page back, and the replay ends in the same state as a clean run", async () => {
      const crashed = await openEnv();
      await expect(applyPage(crashOnCursor(crashed.db), page(changes), NOW)).rejects.toThrow("killed");
      expect(await crashed.repo.list("account")).toEqual([]);
      expect(await getCursor(crashed.db)).toBe(0);
      await applyPage(crashed.db, page(changes), NOW);

      const clean = await openEnv();
      await applyPage(clean.db, page(changes), NOW);
      expect(await dump(crashed.db)).toEqual(await dump(clean.db));
    });

    it("applies the same page twice to the same state: nothing lost, nothing doubled", async () => {
      const env = await openEnv();
      await applyPage(env.db, page(changes), NOW);
      const once = await dump(env.db);
      await applyPage(env.db, page(changes), NOW);
      expect(await dump(env.db)).toEqual(once);
    });
  });
});

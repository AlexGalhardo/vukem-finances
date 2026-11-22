import type { AccountFields, TransactionFields } from "../../src/db/entities";
import { checkIntegrity, OUTBOX_SOFT_CAP, Repository } from "../../src/db/repository";
import { type MemoryDatabase, openMemoryDatabase } from "../helpers/memory-db";

const ACCOUNT: AccountFields = {
  name: "Checking",
  kind: "checking",
  opening_cents: 10000,
  archived_at: null,
  position: 0,
};

function tx(account: string, over: Partial<TransactionFields> = {}): TransactionFields {
  return {
    account_uid: account,
    category_uid: null,
    kind: "expense",
    amount_cents: 1250,
    description: "lunch",
    occurred_on: "2026-11-21",
    transfer_uid: null,
    tag_uids: [],
    statement_ref: null,
    recurrence_uid: null,
    occurrence_on: null,
    installment_group_uid: null,
    installment_no: null,
    installment_total: null,
    ...over,
  };
}

let db: MemoryDatabase;
let repo: Repository;
let n: number;

beforeEach(async () => {
  db = await openMemoryDatabase();
  n = 0;
  repo = new Repository(db, {
    now: () => `2026-11-21T10:00:${String(n).padStart(2, "0")}.000Z`,
    uid: () => `UID${String(++n).padStart(23, "0")}`,
  });
});

describe("create", () => {
  it("stores the row dirty at version 1 and queues the whole create", async () => {
    const a = await repo.create("account", ACCOUNT);
    expect(a).toMatchObject({ ...ACCOUNT, version: 1, dirty: true, server_seq: null, deleted_at: null });

    const [entry, ...rest] = await repo.outbox();
    expect(rest).toHaveLength(0);
    expect(entry).toMatchObject({ entity: "account", uid: a.uid, op: "upsert", base_seq: 0, attempts: 0 });
    expect(entry?.fields).toEqual(ACCOUNT);
    expect(entry?.op_id).toHaveLength(26);
  });

  it("keeps money as integer cents and refuses a float or an overflow", async () => {
    await expect(repo.create("account", { ...ACCOUNT, opening_cents: 10.5 })).rejects.toThrow(RangeError);
    await expect(repo.create("account", { ...ACCOUNT, opening_cents: Number.MAX_SAFE_INTEGER + 2 })).rejects.toThrow(
      RangeError,
    );
    expect(await repo.list("account")).toHaveLength(0);
    expect(await repo.outbox()).toHaveLength(0);
  });

  it("rolls back the row when the outbox write fails", async () => {
    await db.exec("DROP TABLE outbox");
    await expect(repo.create("account", ACCOUNT)).rejects.toThrow();
    expect(await repo.list("account")).toHaveLength(0);
  });
});

describe("update", () => {
  it("bumps the version and queues only the changed fields, based on the seen seq", async () => {
    const a = await repo.create("account", ACCOUNT);
    await db.run("UPDATE accounts SET server_seq = 41 WHERE uid = ?", [a.uid]);

    const edited = await repo.update("account", a.uid, { name: "Main", kind: "checking" });
    expect(edited).toMatchObject({ name: "Main", version: 2, dirty: true });

    const entries = await repo.outbox();
    expect(entries).toHaveLength(2);
    expect(entries[1]).toMatchObject({ op: "upsert", base_seq: 41, fields: { name: "Main" } });
  });

  it("writes nothing when no value changed", async () => {
    const a = await repo.create("account", ACCOUNT);
    const same = await repo.update("account", a.uid, { name: "Checking" });
    expect(same.version).toBe(1);
    expect(await repo.outbox()).toHaveLength(1);
  });

  it("refuses an unknown field", async () => {
    const a = await repo.create("account", ACCOUNT);
    await expect(repo.update("account", a.uid, { nope: 1 } as Partial<AccountFields>)).rejects.toThrow("no field");
  });
});

describe("delete and restore", () => {
  it("soft-deletes with an empty field set and brings the row back", async () => {
    const a = await repo.create("account", ACCOUNT);
    await repo.remove("account", a.uid);

    expect(await repo.get("account", a.uid)).toBeNull();
    const gone = await repo.get("account", a.uid, true);
    expect(gone).toMatchObject({ version: 2 });
    expect(gone?.deleted_at).not.toBeNull();

    await repo.restore("account", a.uid);
    expect(await repo.get("account", a.uid)).toMatchObject({ version: 3, deleted_at: null });

    const ops = (await repo.outbox()).map((e) => [e.op, e.fields]);
    expect(ops).toEqual([
      ["upsert", ACCOUNT],
      ["delete", {}],
      ["restore", {}],
    ]);
  });

  it("restoring a live row is a no-op", async () => {
    const a = await repo.create("account", ACCOUNT);
    await repo.restore("account", a.uid);
    expect(await repo.outbox()).toHaveLength(1);
  });
});

describe("transactions and tags", () => {
  it("keeps the join rows in step with tag_uids", async () => {
    const a = await repo.create("account", ACCOUNT);
    const t1 = await repo.create("tag", { name: "work", color: "" });
    const t2 = await repo.create("tag", { name: "trip", color: "" });
    const row = await repo.create("transaction", tx(a.uid, { tag_uids: [t1.uid, t2.uid] }));

    const joins = async (): Promise<string[]> =>
      (
        await db.all<{ tag_uid: string }>(
          "SELECT tag_uid FROM transaction_tags WHERE transaction_uid = ? ORDER BY tag_uid",
          [row.uid],
        )
      ).map((r) => r.tag_uid);
    expect(await joins()).toEqual([t1.uid, t2.uid].sort());

    await repo.update("transaction", row.uid, { tag_uids: [t2.uid] });
    expect(await joins()).toEqual([t2.uid]);
    expect((await repo.get("transaction", row.uid))?.tag_uids).toEqual([t2.uid]);
  });

  it("enforces the natural key of a recurrence occurrence, deleted rows included", async () => {
    const a = await repo.create("account", ACCOUNT);
    const base = tx(a.uid, { recurrence_uid: "R1", occurrence_on: "2026-11-01" });
    const first = await repo.create("transaction", base);
    await repo.remove("transaction", first.uid);
    await expect(repo.create("transaction", base)).rejects.toThrow();
  });

  it("allows a deleted category name to be reused but not a live one", async () => {
    const food = { name: "Food", kind: "expense", parent_uid: null, color: "", icon: "", archived_at: null };
    const c = await repo.create("category", food);
    await expect(repo.create("category", food)).rejects.toThrow();
    await repo.remove("category", c.uid);
    await repo.create("category", food);
  });
});

describe("checkIntegrity", () => {
  it("reports references to missing or deleted rows", async () => {
    const a = await repo.create("account", ACCOUNT);
    const row = await repo.create("transaction", tx(a.uid));
    expect(await checkIntegrity(db)).toEqual([]);

    await repo.remove("account", a.uid);
    expect(await checkIntegrity(db)).toEqual([
      { entity: "transaction", uid: row.uid, field: "account_uid", missing: a.uid },
    ]);
  });
});

describe("outbox cap", () => {
  it("never drops silently: it only reports, and drops after a confirmed call", async () => {
    await repo.create("account", ACCOUNT);
    const rows = Array.from(
      { length: OUTBOX_SOFT_CAP + 2 },
      (_, i) => `('op${i}', 'tag', 'u${i}', 'upsert', '{}', 'x')`,
    );
    await db.exec(`INSERT INTO outbox (op_id, entity, uid, op, fields, created_at) VALUES ${rows.join(",")}`);

    expect(await repo.outboxOverCap()).toBe(true);
    expect(await repo.outboxCount()).toBe(OUTBOX_SOFT_CAP + 3);
    expect(await repo.dropOldestOutbox(true)).toBe(3);
    expect(await repo.outboxCount()).toBe(OUTBOX_SOFT_CAP);
    expect(await repo.outboxOverCap()).toBe(false);
  });
});

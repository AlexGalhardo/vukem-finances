import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
beforeEach(async () => {
  env = await openEnv();
});

const cat = (name: string, over: Record<string, unknown> = {}) => ({
  name,
  kind: "expense",
  parent_uid: null,
  color: "",
  icon: "",
  ...over,
});

describe("categories", () => {
  it("gives a subcategory the kind of its parent", async () => {
    const food = await env.ledger.createCategory(cat("Food"));
    const sub = await env.ledger.createCategory(cat("Lunch", { kind: "income", parent_uid: food.uid }));
    expect(sub.kind).toBe("expense");
  });

  it("refuses a third level and an archived parent", async () => {
    const food = await env.ledger.createCategory(cat("Food"));
    const sub = await env.ledger.createCategory(cat("Lunch", { parent_uid: food.uid }));
    await expect(env.ledger.createCategory(cat("Deep", { parent_uid: sub.uid }))).rejects.toMatchObject({
      code: "parent",
    });

    await env.repo.update("category", food.uid, { archived_at: "2026-11-01T00:00:00.000Z" });
    await expect(env.ledger.createCategory(cat("More", { parent_uid: food.uid }))).rejects.toMatchObject({
      code: "parent_archived",
    });
  });

  it("does not let a category with children become a subcategory", async () => {
    const a = await env.ledger.createCategory(cat("A"));
    const b = await env.ledger.createCategory(cat("B"));
    await env.ledger.createCategory(cat("A1", { parent_uid: a.uid }));
    await expect(env.ledger.moveCategory(a.uid, b.uid)).rejects.toMatchObject({ code: "nested" });
    expect((await env.ledger.moveCategory(b.uid, a.uid)).parent_uid).toBe(a.uid);
  });

  it("refuses to delete a category with subcategories", async () => {
    const a = await env.ledger.createCategory(cat("A"));
    await env.ledger.createCategory(cat("A1", { parent_uid: a.uid }));
    await expect(env.ledger.deleteCategory(a.uid)).rejects.toMatchObject({ code: "children" });
  });

  it("deleting clears the category of transactions and deletes the budget, one entry each", async () => {
    const acc = await env.repo.create("account", ACCOUNT);
    const food = await env.ledger.createCategory(cat("Food"));
    const t = await env.repo.create("transaction", txFields(acc.uid, { category_uid: food.uid }));
    const b = await env.repo.create("budget", { category_uid: food.uid, limit_cents: 5000, rollover: false });
    const before = (await env.repo.outbox()).length;

    await env.ledger.deleteCategory(food.uid);

    expect((await env.repo.get("transaction", t.uid))?.category_uid).toBeNull();
    expect(await env.repo.get("budget", b.uid)).toBeNull();
    expect(await env.repo.get("category", food.uid)).toBeNull();
    const added = (await env.repo.outbox()).slice(before);
    expect(added.map((e) => [e.entity, e.op, e.fields])).toEqual([
      ["transaction", "upsert", { category_uid: null }],
      ["budget", "delete", {}],
      ["category", "delete", {}],
    ]);
  });

  it("reassigning moves transactions and the budget to a category of the same kind", async () => {
    const acc = await env.repo.create("account", ACCOUNT);
    const a = await env.ledger.createCategory(cat("A"));
    const b = await env.ledger.createCategory(cat("B"));
    const income = await env.ledger.createCategory(cat("Pay", { kind: "income" }));
    const t = await env.repo.create("transaction", txFields(acc.uid, { category_uid: a.uid }));
    const budget = await env.repo.create("budget", { category_uid: a.uid, limit_cents: 100, rollover: true });

    await expect(env.ledger.deleteCategory(a.uid, income.uid)).rejects.toMatchObject({ code: "reassign" });
    await env.ledger.deleteCategory(a.uid, b.uid);

    expect((await env.repo.get("transaction", t.uid))?.category_uid).toBe(b.uid);
    expect((await env.repo.get("budget", budget.uid))?.category_uid).toBe(b.uid);
  });
});

describe("tags", () => {
  it("deleting a tag strips it from every transaction and keeps the join rows in step", async () => {
    const acc = await env.repo.create("account", ACCOUNT);
    const t1 = await env.repo.create("tag", { name: "work", color: "" });
    const t2 = await env.repo.create("tag", { name: "trip", color: "" });
    const tx = await env.repo.create("transaction", txFields(acc.uid, { tag_uids: [t1.uid, t2.uid] }));

    await env.ledger.deleteTag(t1.uid);

    expect((await env.repo.get("transaction", tx.uid))?.tag_uids).toEqual([t2.uid]);
    const joins = await env.db.all<{ tag_uid: string }>(
      "SELECT tag_uid FROM transaction_tags WHERE transaction_uid = ?",
      [tx.uid],
    );
    expect(joins.map((j) => j.tag_uid)).toEqual([t2.uid]);
    expect(await env.repo.get("tag", t1.uid)).toBeNull();
  });
});

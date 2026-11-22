import { budgetStatuses, previousMonth, saveBudget, validMonth } from "../../src/domain/budgets";
import { newUid } from "../../src/domain/ids";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

// The scenarios below are those of tests/budgets_spending_test.go (TestBudgetSpending, TestParentBudget,
// TestOverallBudget, TestBudgetRollover) with the same amounts.

let env: Env;
let acc: string;
let food: string;

beforeEach(async () => {
  env = await openEnv();
  acc = (await env.repo.create("account", { ...ACCOUNT, name: "Main" })).uid;
  food = (
    await env.ledger.createCategory({ name: "Groceries", kind: "expense", parent_uid: null, color: "", icon: "" })
  ).uid;
});

async function spend(category: string | null, cents: number, date: string): Promise<string> {
  return (
    await env.repo.create(
      "transaction",
      txFields(acc, { category_uid: category, amount_cents: cents, occurred_on: date }),
    )
  ).uid;
}

describe("budgetStatuses", () => {
  it("counts the month only and ignores deleted spending", async () => {
    await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 10000 });
    await spend(food, 1000, "2027-03-02");
    const gone = await spend(food, 4000, "2027-03-03");
    await spend(food, 500, "2027-02-28");
    await env.ledger.deleteTransaction(gone);

    const [b] = await budgetStatuses(env.db, "2027-03");
    expect(b).toMatchObject({
      spent_cents: 1000,
      limit_cents: 10000,
      percent: 10,
      category: "Groceries",
      overall: false,
    });
  });

  it("refuses a month that is not YYYY-MM", async () => {
    await expect(budgetStatuses(env.db, "2027-3")).rejects.toMatchObject({ code: "month" });
    expect(validMonth("2027-13")).toBe(false);
    expect(validMonth("2027-12")).toBe(true);
  });

  it("rolls a child's spending up into the parent budget", async () => {
    const kid = await env.ledger.createCategory({
      name: "Snacks",
      kind: "expense",
      parent_uid: food,
      color: "",
      icon: "",
    });
    await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 10000 });
    await saveBudget(env.db, env.repo, { category_uid: kid.uid, limit_cents: 10000 });
    await spend(food, 1000, "2027-03-02");
    await spend(kid.uid, 2500, "2027-03-02");

    const all = await budgetStatuses(env.db, "2027-03");
    expect(all.find((b) => b.category_uid === food)?.spent_cents).toBe(3500);
    expect(all.find((b) => b.category_uid === kid.uid)?.spent_cents).toBe(2500);
  });

  it("keeps one overall budget, counting uncategorized spending and never a transfer", async () => {
    await saveBudget(env.db, env.repo, { category_uid: null, limit_cents: 20000 });
    await saveBudget(env.db, env.repo, { category_uid: null, limit_cents: 5000 });
    await spend(food, 1000, "2027-03-02");
    await spend(null, 1500, "2027-03-02");
    const second = (await env.repo.create("account", { ...ACCOUNT, name: "Second", kind: "savings" })).uid;
    await env.ledger.createTransfer({
      from_account_uid: acc,
      to_account_uid: second,
      amount_cents: 9900,
      date: "2027-03-03",
      description: "",
    });

    const all = await budgetStatuses(env.db, "2027-03");
    expect(all).toHaveLength(1);
    expect(all[0]).toMatchObject({ overall: true, limit_cents: 5000, spent_cents: 2500, percent: 50 });
  });

  it("lists the overall budget first", async () => {
    await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 100 });
    await saveBudget(env.db, env.repo, { category_uid: null, limit_cents: 900 });
    expect((await budgetStatuses(env.db, "2027-03")).map((b) => b.overall)).toEqual([true, false]);
  });

  it("carries the unspent part of last month for a rollover budget made earlier, never chained", async () => {
    await env.repo.create(
      "budget",
      { category_uid: food, limit_cents: 10000, rollover: true },
      newUid(Date.UTC(2027, 1, 10, 10)),
    );
    await spend(food, 3000, "2027-02-11");
    await spend(food, 15000, "2027-03-02");

    const mar = (await budgetStatuses(env.db, "2027-03"))[0];
    expect(mar).toMatchObject({ carry_cents: 7000, available_cents: 17000, percent: 88 });
    const apr = (await budgetStatuses(env.db, "2027-04"))[0];
    expect(apr).toMatchObject({ carry_cents: 0, available_cents: 10000 });
    const feb = (await budgetStatuses(env.db, "2027-02"))[0];
    expect(feb?.carry_cents).toBe(0);
  });

  it("computes a percentage from a sum beyond 2^53 / 100 without a float", async () => {
    await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: Number.MAX_SAFE_INTEGER });
    await spend(food, Number.MAX_SAFE_INTEGER, "2027-03-02");
    expect((await budgetStatuses(env.db, "2027-03"))[0]?.percent).toBe(100);
  });
});

describe("saveBudget", () => {
  it("edits the existing budget and queues only the changed field", async () => {
    const first = await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 100 });
    const second = await saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 200 });
    expect(second.uid).toBe(first.uid);
    expect(second.rollover).toBe(false);
    const last = (await env.repo.outbox()).at(-1);
    expect(last).toMatchObject({ entity: "budget", op: "upsert", fields: { limit_cents: 200 } });
  });

  it("refuses a bad limit and a non-expense category", async () => {
    await expect(saveBudget(env.db, env.repo, { category_uid: food, limit_cents: 0 })).rejects.toMatchObject({
      code: "limit",
    });
    const pay = await env.ledger.createCategory({ name: "Pay", kind: "income", parent_uid: null, color: "", icon: "" });
    await expect(saveBudget(env.db, env.repo, { category_uid: pay.uid, limit_cents: 100 })).rejects.toMatchObject({
      code: "expense",
    });
  });

  it("previousMonth crosses the year", () => {
    expect(previousMonth("2027-01")).toBe("2026-12");
  });
});

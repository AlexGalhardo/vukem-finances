import vectors from "../../../testdata/statement-vectors.json";
import {
  cardStatements,
  closingDate,
  dueDate,
  statementFor,
  statementRefFor,
  statementRefIn,
  statementStatus,
} from "../../src/domain/statements";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

// testdata/statement-vectors.json is written by the Go test TestStatementVectorsExport from the epic 09 functions.
describe("shared statement vectors", () => {
  it("has every group", () => {
    expect(vectors.closing_date.length).toBeGreaterThan(200);
    expect(vectors.statement_for.length).toBeGreaterThan(400);
    expect(vectors.due_date.length).toBeGreaterThan(50);
    expect(vectors.status.length).toBeGreaterThan(30);
  });

  it("closingDate", () => {
    for (const v of vectors.closing_date) {
      expect([v, closingDate(v.year, v.month, v.closing_day)]).toEqual([v, v.expect]);
    }
  });

  it("statementFor", () => {
    for (const v of vectors.statement_for) {
      expect([v, statementFor(v.date, v.closing_day)]).toEqual([v, v.expect]);
    }
  });

  it("dueDate", () => {
    for (const v of vectors.due_date) {
      expect([v, dueDate(v.closing, v.due_day)]).toEqual([v, v.expect]);
    }
  });

  it("statementStatus", () => {
    for (const v of vectors.status) {
      expect([v, statementStatus(v.today, v.closing, v.due, v.total_cents, v.paid_cents)]).toEqual([v, v.expect]);
    }
  });
});

describe("statementRefFor", () => {
  it("is empty for text that is not a date", () => {
    expect(statementRefFor("2027-02-30", 10)).toBe("");
    expect(statementRefFor("2027-02-27", 31)).toBe("2027-02-28");
  });
});

let env: Env;
let cardAccount: string;
let source: string;

beforeEach(async () => {
  env = await openEnv();
  cardAccount = (await env.repo.create("account", { ...ACCOUNT, name: "Card", kind: "credit_card" })).uid;
  source = (await env.repo.create("account", { ...ACCOUNT, name: "Bank", opening_cents: 100000 })).uid;
  await env.repo.create("credit_card", {
    account_uid: cardAccount,
    limit_cents: 500000,
    closing_day: 10,
    due_day: 20,
    payment_account_uid: source,
    brand: "visa",
    last4: "1234",
  });
});

async function buy(kind: string, cents: number, date: string): Promise<void> {
  const ref = await statementRefIn(env.db, cardAccount, date);
  await env.repo.create(
    "transaction",
    txFields(cardAccount, { kind, amount_cents: cents, occurred_on: date, statement_ref: ref }),
  );
}

describe("cardStatements", () => {
  it("tags a purchase with its statement only on a card account", async () => {
    expect(await statementRefIn(env.db, cardAccount, "2026-11-11")).toBe("2026-12-10");
    expect(await statementRefIn(env.db, source, "2026-11-11")).toBeNull();
  });

  it("sums purchases, refunds and payments per statement and derives the status", async () => {
    await buy("expense", 10000, "2026-10-05");
    await buy("expense", 5000, "2026-11-09");
    await buy("income", 1500, "2026-11-09");
    await buy("expense", 700, "2026-11-12");
    await env.ledger.createTransfer({
      from_account_uid: source,
      to_account_uid: cardAccount,
      amount_cents: 3000,
      date: "2026-11-15",
      description: "pay",
    });
    const payment = (await env.repo.list("transaction")).find(
      (t) => t.account_uid === cardAccount && t.transfer_uid !== null,
    );
    await env.repo.update("transaction", payment?.uid ?? "", { statement_ref: "2026-11-10" });

    const list = await cardStatements(env.db, { account_uid: cardAccount, closing_day: 10, due_day: 20 }, "2026-11-21");
    expect(list.map((s) => [s.ref, s.status])).toEqual([
      ["2026-12-10", "open"],
      ["2026-11-10", "overdue"],
      ["2026-10-10", "overdue"],
    ]);
    expect(list[1]).toMatchObject({
      due_on: "2026-11-20",
      purchases_cents: 5000,
      credits_cents: 1500,
      total_cents: 3500,
      paid_cents: 3000,
      remaining_cents: 500,
    });
    expect(list[0]).toMatchObject({ purchases_cents: 700, total_cents: 700 });
  });

  it("lists the open statement even with no rows and ignores deleted rows", async () => {
    const empty = await cardStatements(
      env.db,
      { account_uid: cardAccount, closing_day: 10, due_day: 20 },
      "2026-11-21",
    );
    expect(empty).toHaveLength(1);
    expect(empty[0]).toMatchObject({ ref: "2026-12-10", status: "open", total_cents: 0 });

    await buy("expense", 999, "2026-11-12");
    const tx = (await env.repo.list("transaction"))[0];
    await env.ledger.deleteTransaction(tx?.uid ?? "");
    const after = await cardStatements(
      env.db,
      { account_uid: cardAccount, closing_day: 10, due_day: 20 },
      "2026-11-21",
    );
    expect(after[0]?.purchases_cents).toBe(0);
  });
});

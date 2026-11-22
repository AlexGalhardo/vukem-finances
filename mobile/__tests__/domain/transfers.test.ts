import { accountBalances, LedgerError, netWorthOf, type TransferInput } from "../../src/domain/ledger";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
let from: string;
let to: string;

beforeEach(async () => {
  env = await openEnv();
  from = (await env.repo.create("account", { ...ACCOUNT, name: "A", opening_cents: 10000 })).uid;
  to = (await env.repo.create("account", { ...ACCOUNT, name: "B", opening_cents: 500 })).uid;
});

function input(over: Partial<TransferInput> = {}): TransferInput {
  return {
    from_account_uid: from,
    to_account_uid: to,
    amount_cents: 2500,
    date: "2026-11-12",
    description: " rent ",
    ...over,
  };
}

async function totals(): Promise<{ income: number; expense: number }> {
  const rows = await env.db.all<{ kind: string; c: number }>(
    "SELECT kind, COALESCE(SUM(amount_cents), 0) AS c FROM transactions WHERE deleted_at IS NULL AND transfer_uid IS NULL GROUP BY kind",
  );
  return {
    income: rows.find((r) => r.kind === "income")?.c ?? 0,
    expense: rows.find((r) => r.kind === "expense")?.c ?? 0,
  };
}

describe("createTransfer", () => {
  it("writes two legs sharing a transfer_uid, expense on the source and income on the destination", async () => {
    const r = await env.ledger.createTransfer(input());
    const src = await env.repo.get("transaction", r.source);
    const dst = await env.repo.get("transaction", r.destination);
    expect(src).toMatchObject({
      kind: "expense",
      account_uid: from,
      transfer_uid: r.transfer_uid,
      category_uid: null,
      amount_cents: 2500,
      description: "rent",
    });
    expect(dst).toMatchObject({ kind: "income", account_uid: to, transfer_uid: r.transfer_uid, amount_cents: 2500 });
    expect((await env.repo.outbox()).filter((e) => e.entity === "transaction")).toHaveLength(2);
  });

  it("changes two balances and no income or expense total", async () => {
    const before = await totals();
    await env.ledger.createTransfer(input());
    const balances = await accountBalances(env.db, "2026-11-12");
    expect(balances.map((b) => b.balance_cents)).toEqual([7500, 3000]);
    expect(netWorthOf(balances)).toBe(10500);
    expect(await totals()).toEqual(before);
  });

  it("refuses bad input and writes nothing", async () => {
    const bad = [
      input({ amount_cents: 0 }),
      input({ amount_cents: -5 }),
      input({ amount_cents: 1.5 }),
      input({ to_account_uid: from }),
      input({ to_account_uid: "nope" }),
    ];
    for (const b of bad) {
      await expect(env.ledger.createTransfer(b)).rejects.toBeInstanceOf(LedgerError);
    }
    await env.repo.update("account", to, { archived_at: "2026-11-01T00:00:00.000Z" });
    await expect(env.ledger.createTransfer(input())).rejects.toMatchObject({ code: "archived" });
    expect(await env.repo.list("transaction")).toHaveLength(0);
  });

  it("rolls both legs back when the second fails", async () => {
    await env.db.exec(
      "CREATE TRIGGER boom BEFORE INSERT ON transactions WHEN NEW.kind = 'income' BEGIN SELECT RAISE(ABORT, 'boom'); END",
    );
    await expect(env.ledger.createTransfer(input())).rejects.toThrow("boom");
    expect(await env.repo.list("transaction")).toHaveLength(0);
  });
});

describe("deleting and restoring", () => {
  it("deleting one leg deletes the other, restoring one brings both back", async () => {
    const r = await env.ledger.createTransfer(input());
    await env.ledger.deleteTransaction(r.source);
    expect(await env.repo.list("transaction")).toHaveLength(0);
    await env.ledger.restoreTransaction(r.destination);
    expect(await env.repo.list("transaction")).toHaveLength(2);
  });

  it("deleting an account takes its transactions and the other leg of its transfers", async () => {
    await env.repo.create("transaction", txFields(from, { amount_cents: 1 }));
    await env.ledger.createTransfer(input());
    await env.ledger.deleteAccount(from);
    expect(await env.repo.list("transaction")).toHaveLength(0);

    await env.ledger.restoreAccount(from);
    expect(await env.repo.list("transaction")).toHaveLength(3);
  });

  it("refuses to restore a leg while its account is deleted", async () => {
    const r = await env.ledger.createTransfer(input());
    await env.ledger.deleteTransaction(r.source);
    await env.ledger.deleteAccount(to);
    await expect(env.ledger.restoreTransaction(r.source)).rejects.toMatchObject({ code: "account_deleted" });
  });
});

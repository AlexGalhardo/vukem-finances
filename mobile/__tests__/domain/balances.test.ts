import { accountBalances, netWorthOf } from "../../src/domain/ledger";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
beforeEach(async () => {
	env = await openEnv();
});

describe("accountBalances", () => {
	it("is opening plus signed transactions up to the date, inclusive", async () => {
		const a = await env.repo.create("account", { ...ACCOUNT, opening_cents: 10000 });
		await env.repo.create("transaction", txFields(a.uid, { kind: "income", amount_cents: 5050, occurred_on: "2026-11-05" }));
		await env.repo.create("transaction", txFields(a.uid, { kind: "expense", amount_cents: 1234, occurred_on: "2026-11-10" }));
		await env.repo.create("transaction", txFields(a.uid, { kind: "expense", amount_cents: 999, occurred_on: "2026-11-11" }));

		const [on10] = await accountBalances(env.db, "2026-11-10");
		expect(on10).toMatchObject({ opening_cents: 10000, movement_cents: 3816, balance_cents: 13816, archived: false });
		const [on11] = await accountBalances(env.db, "2026-11-11");
		expect(on11?.balance_cents).toBe(12817);
		const [early] = await accountBalances(env.db, "2026-11-01");
		expect(early?.balance_cents).toBe(10000);
	});

	it("ignores deleted transactions and accounts", async () => {
		const a = await env.repo.create("account", ACCOUNT);
		const t = await env.repo.create("transaction", txFields(a.uid, { amount_cents: 700 }));
		await env.ledger.deleteTransaction(t.uid);
		const b = await env.repo.create("account", { ...ACCOUNT, name: "Gone" });
		await env.repo.remove("account", b.uid);

		const balances = await accountBalances(env.db, "2026-12-31");
		expect(balances.map((x) => [x.account_uid, x.balance_cents])).toEqual([[a.uid, 0]]);
	});

	it("leaves archived accounts out of net worth but still lists them", async () => {
		await env.repo.create("account", { ...ACCOUNT, opening_cents: 100 });
		const b = await env.repo.create("account", {
			...ACCOUNT,
			name: "Old",
			opening_cents: 900,
			archived_at: "2026-11-01T00:00:00.000Z",
		});
		const balances = await accountBalances(env.db, "2026-11-21");
		expect(balances.find((x) => x.account_uid === b.uid)?.archived).toBe(true);
		expect(netWorthOf(balances)).toBe(100);
	});

	it("refuses a balance beyond the safe range", async () => {
		const a = await env.repo.create("account", { ...ACCOUNT, opening_cents: Number.MAX_SAFE_INTEGER });
		await env.repo.create("transaction", txFields(a.uid, { kind: "income", amount_cents: 1 }));
		await expect(accountBalances(env.db, "2026-12-31")).rejects.toThrow("range");
	});
});

import { bps, byCategory, cashFlow, periodMonths, previousPeriod, resolvePeriod } from "../../src/domain/reports";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

describe("resolvePeriod", () => {
	it("reads the presets as whole calendar months", () => {
		expect(resolvePeriod("", "", "", "2028-03-15")).toEqual({ preset: "this_month", from: "2028-03-01", to: "2028-03-31" });
		expect(resolvePeriod("last_month", "", "", "2028-03-15")).toMatchObject({ from: "2028-02-01", to: "2028-02-29" });
		expect(resolvePeriod("last_3_months", "", "", "2028-03-15")).toMatchObject({ from: "2028-01-01", to: "2028-03-31" });
		expect(resolvePeriod("last_12_months", "", "", "2028-03-15")).toMatchObject({ from: "2027-04-01", to: "2028-03-31" });
		expect(resolvePeriod("this_year", "", "", "2028-03-15")).toMatchObject({ from: "2028-01-01", to: "2028-12-31" });
	});

	it("validates a custom range", () => {
		expect(resolvePeriod("custom", "2028-01-05", "2028-01-09", "2028-03-15")).toMatchObject({ preset: "custom" });
		for (const [f, t] of [
			["2028-01-09", "2028-01-05"],
			["2028-02-30", "2028-03-01"],
			["x", "2028-03-01"],
			["2000-01-01", "2030-01-01"],
		] as const) {
			expect(() => resolvePeriod("custom", f, t, "2028-03-15")).toThrow();
		}
		expect(() => resolvePeriod("nope", "", "", "2028-03-15")).toThrow();
	});

	it("compares whole months with the months before and others with the same days", () => {
		const march = resolvePeriod("this_month", "", "", "2028-03-15");
		expect(previousPeriod(march)).toEqual({ preset: "custom", from: "2028-02-01", to: "2028-02-29" });
		const odd = resolvePeriod("custom", "2028-03-10", "2028-03-19", "2028-03-15");
		expect(previousPeriod(odd)).toEqual({ preset: "custom", from: "2028-02-29", to: "2028-03-09" });
	});

	it("lists the months a period touches", () => {
		expect(periodMonths({ preset: "custom", from: "2027-11-20", to: "2028-02-02" })).toEqual(["2027-11", "2027-12", "2028-01", "2028-02"]);
	});
});

describe("bps", () => {
	it("rounds half up with integers", () => {
		expect(bps(1, 3)).toBe(3333);
		expect(bps(2, 3)).toBe(6667);
		expect(bps(1, 8)).toBe(1250);
		expect(bps(0, 5)).toBe(0);
		expect(bps(5, 0)).toBe(0);
		expect(bps(Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER)).toBe(10000);
	});
});

let env: Env;
let acc: string;
let other: string;

beforeEach(async () => {
	env = await openEnv();
	acc = (await env.repo.create("account", { ...ACCOUNT, opening_cents: 100000 })).uid;
	other = (await env.repo.create("account", { ...ACCOUNT, name: "Second" })).uid;
});

describe("cashFlow", () => {
	it("sums income and expense per month and leaves transfers out", async () => {
		await env.repo.create("transaction", txFields(acc, { kind: "income", amount_cents: 300000, occurred_on: "2027-01-05" }));
		await env.repo.create("transaction", txFields(acc, { amount_cents: 1250, occurred_on: "2027-01-06" }));
		await env.repo.create("transaction", txFields(acc, { amount_cents: 1250, occurred_on: "2027-01-06" }));
		await env.repo.create("transaction", txFields(acc, { amount_cents: 999, occurred_on: "2027-02-01" }));
		await env.repo.create("transaction", txFields(acc, { amount_cents: 7, occurred_on: "2026-12-31" }));
		await env.ledger.createTransfer({ from_account_uid: acc, to_account_uid: other, amount_cents: 5000, date: "2027-01-07", description: "" });
		const gone = await env.repo.create("transaction", txFields(acc, { amount_cents: 4242, occurred_on: "2027-01-08" }));
		await env.ledger.deleteTransaction(gone.uid);

		const report = await cashFlow(env.db, resolvePeriod("custom", "2027-01-01", "2027-02-28", "2027-02-15"));
		expect(report.months).toEqual([
			{ month: "2027-01", income_cents: 300000, expense_cents: 2500, net_cents: 297500 },
			{ month: "2027-02", income_cents: 0, expense_cents: 999, net_cents: -999 },
		]);
		expect(report.totals).toEqual({ income_cents: 300000, expense_cents: 3499, net_cents: 296501 });
	});
});

describe("byCategory", () => {
	it("rolls children into parents, adds an uncategorized row and sorts by amount", async () => {
		const mk = (name: string, parent: string | null = null) =>
			env.ledger.createCategory({ name, kind: "expense", parent_uid: parent, color: "", icon: "" });
		const food = await mk("Food");
		const snacks = await mk("Snacks", food.uid);
		const fun = await mk("Fun");
		const spend = (category: string | null, cents: number) =>
			env.repo.create("transaction", txFields(acc, { category_uid: category, amount_cents: cents, occurred_on: "2027-03-02" }));
		await spend(food.uid, 1000);
		await spend(snacks.uid, 2500);
		await spend(fun.uid, 4000);
		await spend(null, 500);
		await env.repo.create("transaction", txFields(acc, { kind: "income", amount_cents: 99999, occurred_on: "2027-03-02" }));

		const r = await byCategory(env.db, resolvePeriod("this_month", "", "", "2027-03-15"), "expense");
		expect(r.total_cents).toBe(8000);
		expect(r.rows.map((x) => [x.name, x.cents, x.share_bps])).toEqual([
			["Fun", 4000, 5000],
			["Food", 3500, 4375],
			["", 500, 625],
		]);
		const foodRow = r.rows.find((x) => x.uid === food.uid);
		expect(foodRow).toMatchObject({ own_cents: 1000, cents: 3500 });
		expect(foodRow?.children).toMatchObject([{ name: "Snacks", cents: 2500, share_bps: 3125 }]);
		expect(r.rows.find((x) => x.uncategorized)?.cents).toBe(500);
	});

	it("keeps a deleted category's spending on its row", async () => {
		const c = await env.ledger.createCategory({ name: "Old", kind: "expense", parent_uid: null, color: "", icon: "" });
		await env.repo.create("transaction", txFields(acc, { category_uid: c.uid, amount_cents: 100, occurred_on: "2027-03-02" }));
		await env.repo.remove("category", c.uid);
		const r = await byCategory(env.db, resolvePeriod("this_month", "", "", "2027-03-15"), "expense");
		expect(r.rows.map((x) => [x.name, x.cents])).toEqual([["Old", 100]]);
	});
});

import vectors from "../../../testdata/recurrence-vectors.json";
import {
	createInstallments,
	generateDue,
	GENERATE_CAP,
	installmentDates,
	installmentLabel,
	occurrences,
	type Frequency,
	splitInstallments,
	upcoming,
	validateRule,
} from "../../src/domain/recurrence";
import { ACCOUNT, type Env, openEnv } from "../helpers/fixtures";

describe("shared recurrence vectors", () => {
	it("has vectors", () => {
		expect(vectors.length).toBeGreaterThanOrEqual(70);
	});

	it.each(vectors.map((v) => [v.name, v] as const))("%s", (_name, v) => {
		const rule = { ...v.rule, frequency: v.rule.frequency as Frequency };
		expect(occurrences(rule, v.from, v.to)).toEqual(v.expect);
	});
});

describe("occurrences", () => {
	const monthly = { frequency: "monthly", interval: 1, start_date: "2027-01-31", end_date: null, max_count: null } as const;

	it("never drifts after a clamped month", () => {
		expect(occurrences(monthly, "2027-01-01", "2027-05-31")).toEqual(["2027-01-31", "2027-02-28", "2027-03-31", "2027-04-30", "2027-05-31"]);
	});

	it("counts max_count from the start, not from the window", () => {
		const rule = { ...monthly, start_date: "2027-01-10", max_count: 4 };
		expect(occurrences(rule, "2027-04-01", "2027-12-31")).toEqual(["2027-04-10"]);
	});

	it("honors the limit, an empty window and malformed input", () => {
		expect(occurrences(monthly, "2027-01-01", "2030-01-01", 2)).toHaveLength(2);
		expect(occurrences(monthly, "2027-06-01", "2027-01-01")).toEqual([]);
		expect(occurrences({ ...monthly, interval: 0 }, "2027-01-01", "2027-12-31")).toEqual([]);
		expect(occurrences({ ...monthly, frequency: "hourly" as Frequency }, "2027-01-01", "2027-12-31")).toEqual([]);
		expect(occurrences(monthly, "2027-02-30", "2027-12-31")).toEqual([]);
	});

	it("walks a far window without starting at k = 0 and stops at year 9999", () => {
		const daily = { frequency: "daily", interval: 1, start_date: "2000-01-01", end_date: null, max_count: null } as const;
		expect(occurrences(daily, "2900-05-05", "2900-05-07")).toEqual(["2900-05-05", "2900-05-06", "2900-05-07"]);
		expect(occurrences(daily, "9999-12-30", "9999-12-31")).toEqual(["9999-12-30", "9999-12-31"]);
		const yearly = { frequency: "yearly", interval: 1, start_date: "2999-02-28", end_date: null, max_count: null } as const;
		expect(occurrences(yearly, "9998-01-01", "9999-12-31")).toEqual(["9998-02-28", "9999-02-28"]);
	});

	it("looks at no more than 10000 occurrences per call", () => {
		const daily = { frequency: "daily", interval: 1, start_date: "2000-01-01", end_date: null, max_count: null } as const;
		expect(occurrences(daily, "2000-01-01", "2099-12-31")).toHaveLength(10000);
	});
});

describe("validateRule", () => {
	const ok = { frequency: "monthly", interval: 1, start_date: "2027-01-31", end_date: null, max_count: null } as const;

	it("accepts a good rule and refuses what the server refuses", () => {
		expect(() => validateRule(ok)).not.toThrow();
		expect(() => validateRule({ ...ok, interval: 0 })).toThrow();
		expect(() => validateRule({ ...ok, max_count: 0 })).toThrow();
		expect(() => validateRule({ ...ok, start_date: "1899-12-31" })).toThrow();
		expect(() => validateRule({ ...ok, end_date: "3000-01-01" })).toThrow();
		expect(() => validateRule({ ...ok, start_date: "2027-02-29" })).toThrow();
		expect(() => validateRule({ ...ok, frequency: "hourly" as Frequency })).toThrow();
	});
});

let env: Env;
let acc: string;

beforeEach(async () => {
	env = await openEnv();
	acc = (await env.repo.create("account", ACCOUNT)).uid;
});

async function rule(over: Record<string, unknown> = {}): Promise<string> {
	return (
		await env.repo.create("recurrence", {
			account_uid: acc,
			category_uid: null,
			kind: "expense",
			amount_cents: 1500,
			description: "rent",
			frequency: "monthly",
			interval: 1,
			start_date: "2027-01-31",
			end_date: null,
			max_count: null,
			generated_through: null,
			active: true,
			...over,
		})
	).uid;
}

const generated = (): Promise<{ occurrence_on: string; deleted_at: string | null }[]> =>
	env.db.all("SELECT occurrence_on, deleted_at FROM transactions WHERE recurrence_uid IS NOT NULL ORDER BY occurrence_on");

describe("generateDue", () => {
	it("makes each due occurrence with its natural key and moves the watermark", async () => {
		const uid = await rule();
		const r = await generateDue(env.db, env.repo, "2027-03-15");
		expect(r).toEqual({ created: 2, failed: [] });
		expect((await generated()).map((g) => g.occurrence_on)).toEqual(["2027-01-31", "2027-02-28"]);
		expect((await env.repo.get("recurrence", uid))?.generated_through).toBe("2027-02-28");

		const tx = (await env.repo.list("transaction"))[0];
		expect(tx).toMatchObject({ recurrence_uid: uid, kind: "expense", amount_cents: 1500, description: "rent" });
		expect(tx?.occurred_on).toBe(tx?.occurrence_on);
	});

	it("is idempotent: a second run creates nothing and queues nothing", async () => {
		await rule();
		await generateDue(env.db, env.repo, "2027-03-15");
		const outbox = (await env.repo.outbox()).length;
		expect(await generateDue(env.db, env.repo, "2027-03-15")).toEqual({ created: 0, failed: [] });
		expect((await env.repo.list("transaction")).length).toBe(2);
		expect((await env.repo.outbox()).length).toBe(outbox);
	});

	it("never recreates an occurrence the person deleted, even with the watermark lost", async () => {
		const uid = await rule();
		await generateDue(env.db, env.repo, "2027-03-15");
		const first = (await env.repo.list("transaction")).find((t) => t.occurrence_on === "2027-01-31");
		await env.ledger.deleteTransaction(first?.uid ?? "");
		await env.repo.update("recurrence", uid, { generated_through: null });

		expect((await generateDue(env.db, env.repo, "2027-03-15")).created).toBe(0);
		const rows = await generated();
		expect(rows.filter((g) => g.occurrence_on === "2027-01-31")).toHaveLength(1);
		expect(rows.find((g) => g.occurrence_on === "2027-01-31")?.deleted_at).not.toBeNull();
	});

	it("skips inactive rules and caps one run at 500 occurrences", async () => {
		await rule({ active: false });
		expect((await generateDue(env.db, env.repo, "2027-03-15")).created).toBe(0);

		const daily = await rule({ frequency: "daily", start_date: "2020-01-01" });
		expect((await generateDue(env.db, env.repo, "2027-03-15")).created).toBe(GENERATE_CAP);
		expect((await env.repo.get("recurrence", daily))?.generated_through).toBe("2021-05-14");
		expect((await generateDue(env.db, env.repo, "2027-03-15")).created).toBe(GENERATE_CAP);
	});

	it("stops before an occurrence whose account is archived, retries later, and does not block others", async () => {
		const blocked = await env.repo.create("account", { ...ACCOUNT, name: "Old", archived_at: "2027-01-01T00:00:00.000Z" });
		const stuck = await rule({ account_uid: blocked.uid });
		await rule({ description: "other" });

		const r = await generateDue(env.db, env.repo, "2027-02-10");
		expect(r.created).toBe(1);
		expect(r.failed).toEqual([{ recurrence_uid: stuck, reason: "account_archived" }]);
		expect((await env.repo.get("recurrence", stuck))?.generated_through).toBeNull();

		await env.repo.update("account", blocked.uid, { archived_at: null });
		expect((await generateDue(env.db, env.repo, "2027-02-10")).created).toBe(1);
	});

	it("clears a deleted category instead of failing", async () => {
		const cat = await env.ledger.createCategory({ name: "Home", kind: "expense", parent_uid: null, color: "", icon: "" });
		await rule({ category_uid: cat.uid });
		await env.repo.remove("category", cat.uid);
		await generateDue(env.db, env.repo, "2027-01-31");
		expect((await env.repo.list("transaction"))[0]?.category_uid).toBeNull();
	});
});

describe("installments", () => {
	it("splits 100.00 in 3 as 33.34, 33.33, 33.33 and always sums to the total", () => {
		expect(splitInstallments(10000, 3)).toEqual([3334, 3333, 3333]);
		for (const [total, parts] of [[7, 3], [60, 60], [1000001, 7], [Number.MAX_SAFE_INTEGER, 13]] as const) {
			const split = splitInstallments(total, parts);
			expect(split).toHaveLength(parts);
			expect(split.reduce((a, b) => a + b, 0)).toBe(total);
			expect(Math.max(...split) - Math.min(...split)).toBeLessThanOrEqual(1);
		}
	});

	it("dates the parts from the first date with month-end clamping", () => {
		expect(installmentDates("2027-01-31", 3)).toEqual(["2027-01-31", "2027-02-28", "2027-03-31"]);
		expect(installmentLabel("TV", 2, 6)).toBe("TV (2/6)");
	});

	it("writes every part in one go, linked by a group", async () => {
		const parts = await createInstallments(env.db, env.repo, {
			account_uid: acc,
			category_uid: null,
			total_cents: 10000,
			parts: 3,
			first_date: "2027-01-31",
			description: "TV",
		});
		expect(parts.map((p) => [p.amount_cents, p.occurred_on, p.installment_no, p.installment_total])).toEqual([
			[3334, "2027-01-31", 1, 3],
			[3333, "2027-02-28", 2, 3],
			[3333, "2027-03-31", 3, 3],
		]);
		expect(new Set(parts.map((p) => p.installment_group_uid)).size).toBe(1);
		expect(parts[0]?.description).toBe("TV");
	});

	it("refuses bad input and writes nothing", async () => {
		const base = { account_uid: acc, category_uid: null, total_cents: 10000, parts: 3, first_date: "2027-01-31", description: "TV" };
		for (const bad of [
			{ parts: 1 },
			{ parts: 61 },
			{ total_cents: 2, parts: 3 },
			{ total_cents: 0 },
			{ first_date: "2027-02-30" },
			{ first_date: "3000-01-01" },
			{ account_uid: "nope" },
		]) {
			await expect(createInstallments(env.db, env.repo, { ...base, ...bad })).rejects.toThrow();
		}
		expect(await env.repo.list("transaction")).toHaveLength(0);
	});
});

describe("upcoming", () => {
	it("merges future rows with virtual occurrences and hides what a row already holds", async () => {
		const uid = await rule({ start_date: "2027-03-10", frequency: "weekly" });
		await createInstallments(env.db, env.repo, {
			account_uid: acc,
			category_uid: null,
			total_cents: 3000,
			parts: 3,
			first_date: "2027-03-12",
			description: "TV",
		});

		const items = await upcoming(env.db, env.repo, "2027-03-10", 7);
		expect(items.map((i) => [i.date, i.source])).toEqual([
			["2027-03-10", "recurrence"],
			["2027-03-12", "installment"],
			["2027-03-17", "recurrence"],
		]);
		expect(items[0]).toMatchObject({ recurrence_uid: uid, transaction_uid: null, amount_cents: 1500 });
		expect(items[1]).toMatchObject({ installment_no: 1, installment_total: 3, amount_cents: 1000 });

		await generateDue(env.db, env.repo, "2027-03-10");
		const after = await upcoming(env.db, env.repo, "2027-03-10", 7);
		expect(after.map((i) => i.date)).toEqual(["2027-03-12", "2027-03-17"]);
	});

	it("never shows a deleted occurrence again", async () => {
		await rule({ start_date: "2027-03-10", frequency: "weekly" });
		await generateDue(env.db, env.repo, "2027-03-10");
		const made = (await env.repo.list("transaction"))[0];
		await env.ledger.deleteTransaction(made?.uid ?? "");
		const items = await upcoming(env.db, env.repo, "2027-03-10", 1);
		expect(items).toEqual([]);
	});
});

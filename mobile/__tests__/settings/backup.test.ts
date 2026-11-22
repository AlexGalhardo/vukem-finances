import schema from "../../../docs/export-format.schema.json";
import { backupFileName, buildBackup } from "../../src/backup/export";
import { createCard } from "../../src/domain/cards";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";
import { type Schema, validate } from "../helpers/json-schema";

const WORKSPACE = { name: "Vukem Finances", currency: "USD", timezone: "UTC" };
const NOW = new Date(Date.UTC(2027, 2, 15, 15, 0, 0, 123));
const root = schema as unknown as Schema;

type Doc = Record<string, Record<string, unknown>[]> & Record<string, unknown>;

async function seeded(): Promise<Env> {
	const env = await openEnv();
	const acc = (await env.repo.create("account", { ...ACCOUNT, name: "Main", opening_cents: 123456 })).uid;
	const food = await env.ledger.createCategory({ name: "Food", kind: "expense", parent_uid: null, color: "green", icon: "" });
	await env.ledger.createCategory({ name: "Snacks", kind: "expense", parent_uid: food.uid, color: "", icon: "" });
	const tag = await env.repo.create("tag", { name: "work", color: "" });
	await env.repo.create("transaction", txFields(acc, { category_uid: food.uid, amount_cents: 1250, tag_uids: [tag.uid], description: "lunch" }));
	const second = (await env.repo.create("account", { ...ACCOUNT, name: "Second", position: 1 })).uid;
	await env.ledger.createTransfer({ from_account_uid: acc, to_account_uid: second, amount_cents: 500, date: "2027-03-12", description: "" });
	await env.repo.create("budget", { category_uid: null, limit_cents: 100000, rollover: true });
	await env.repo.create("recurrence", {
		account_uid: acc,
		category_uid: null,
		kind: "expense",
		amount_cents: 999,
		description: "gym",
		frequency: "weekly",
		interval: 1,
		start_date: "2027-03-01",
		end_date: null,
		max_count: 4,
		generated_through: null,
		active: true,
	});
	await createCard(env.db, env.repo, { name: "Visa", limit_cents: 500000, closing_day: 10, due_day: 20, payment_account_uid: acc });
	return env;
}

describe("backup", () => {
	it("validates against docs/export-format.schema.json", async () => {
		const env = await seeded();
		const backup = await buildBackup(env.db, WORKSPACE, NOW);
		expect(validate(root, backup)).toEqual([]);
		expect(backup).toMatchObject({ format: "vukem-finances-export", version: 1, exported_at: "2027-03-15T15:00:00Z" });
	});

	it("validates an empty database too", async () => {
		const env = await openEnv();
		expect(validate(root, await buildBackup(env.db, WORKSPACE, NOW))).toEqual([]);
	});

	it("holds every live row with integer cents, uid references and decoded tags", async () => {
		const env = await seeded();
		const b = (await buildBackup(env.db, WORKSPACE, NOW)) as Doc;
		expect(b.accounts?.map((a) => a.name).sort()).toEqual(["Main", "Second", "Visa"]);
		expect(b.accounts?.[0]).toMatchObject({ opening_cents: 123456 });
		expect(b.categories).toHaveLength(2);
		expect(b.transactions).toHaveLength(3);
		const lunch = b.transactions?.find((t) => t.description === "lunch");
		expect(lunch).toMatchObject({ amount_cents: 1250, tag_uids: [b.tags?.[0]?.uid] });
		const legs = b.transactions?.filter((t) => t.transfer_uid !== null);
		expect(legs).toHaveLength(2);
		expect(legs?.[0]?.transfer_uid).toBe(legs?.[1]?.transfer_uid);
		expect(b.budgets?.[0]).toMatchObject({ category_uid: null, rollover: true });
		expect(b.cards?.[0]).toMatchObject({ limit_cents: 500000, closing_day: 10 });
		expect(b.goals).toEqual([]);
	});

	it("leaves deleted rows out", async () => {
		const env = await seeded();
		const first = (await env.repo.list("transaction")).find((t) => t.description === "lunch");
		await env.ledger.deleteTransaction(first?.uid ?? "");
		const b = (await buildBackup(env.db, WORKSPACE, NOW)) as Doc;
		expect(b.transactions).toHaveLength(2);
		expect(JSON.stringify(b)).not.toContain(first?.uid ?? "never");
	});

	it("carries no sync bookkeeping or secrets", async () => {
		const env = await seeded();
		const text = JSON.stringify(await buildBackup(env.db, WORKSPACE, NOW));
		for (const word of ["server_seq", "dirty", "updated_local_at", "lock", "secret", "token", "password"]) {
			expect(text).not.toContain(word);
		}
	});

	it("names the file by the date", () => {
		expect(backupFileName("2027-03-15")).toBe("vukem-finances-backup-2027-03-15.json");
	});
});

describe("the validator is strict (so the test above means something)", () => {
	it("rejects a float amount, an extra field, a missing field and a wrong format", async () => {
		const env = await seeded();
		const good = (await buildBackup(env.db, WORKSPACE, NOW)) as Doc;
		const clone = (): Doc => JSON.parse(JSON.stringify(good)) as Doc;

		const float = clone();
		float.transactions = [{ ...float.transactions?.[0], amount_cents: 12.5 }];
		expect(validate(root, float).join()).toContain("amount_cents");

		const extra = clone();
		extra.accounts = [{ ...extra.accounts?.[0], secret: "x" }];
		expect(validate(root, extra).join()).toContain("unexpected secret");

		const missing = clone();
		delete missing.goals;
		expect(validate(root, missing).join()).toContain("missing goals");

		const wrong = clone();
		wrong.format = "other" as never;
		expect(validate(root, wrong).join()).toContain("format");
	});
});

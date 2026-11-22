import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { AddScreen } from "../../src/screens/AddScreen";
import { ACCOUNT, type Env, txFields } from "../helpers/fixtures";
import { renderApp } from "../helpers/app";
import { openEnv } from "../helpers/fixtures";

let env: Env;
let checking: string;
let savings: string;
let groceries: string;

beforeEach(async () => {
	env = await openEnv();
	checking = (await env.repo.create("account", { ...ACCOUNT, name: "Checking", opening_cents: 100000 })).uid;
	savings = (await env.repo.create("account", { ...ACCOUNT, name: "Savings", kind: "savings", position: 1 })).uid;
	groceries = (
		await env.ledger.createCategory({ name: "Groceries", kind: "expense", parent_uid: null, color: "", icon: "" })
	).uid;
	await env.ledger.createCategory({ name: "Salary", kind: "income", parent_uid: null, color: "", icon: "" });
});

async function rows(): Promise<{ amount_cents: number; kind: string; category_uid: string | null; occurred_on: string; description: string; account_uid: string }[]> {
	return env.db.all("SELECT amount_cents, kind, category_uid, occurred_on, description, account_uid FROM transactions WHERE deleted_at IS NULL ORDER BY uid");
}

describe("Add", () => {
	it('stores "1.234,56" as 123456 cents and queues one outbox entry', async () => {
		await renderApp(<AddScreen />, { env });
		await screen.findByRole("radio", { name: "Checking" });

		await fireEvent.changeText(screen.getByLabelText("Amount"), "1.234,56");
		await fireEvent.press(screen.getByRole("radio", { name: "Groceries" }));
		await fireEvent.changeText(screen.getByLabelText("Note"), "weekly shop");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		await waitFor(async () => expect(await rows()).toHaveLength(1));
		expect((await rows())[0]).toEqual({
			amount_cents: 123456,
			kind: "expense",
			category_uid: groceries,
			occurred_on: "2027-03-15",
			description: "weekly shop",
			account_uid: checking,
		});
		const entry = (await env.repo.outbox()).filter((e) => e.entity === "transaction");
		expect(entry).toHaveLength(1);
		expect(entry[0]?.fields).toMatchObject({ amount_cents: 123456, kind: "expense" });
		expect(await screen.findByText("Transaction saved")).toBeTruthy();
		expect(screen.getByLabelText("Amount").props.value).toBe("");
	});

	it("refuses an amount it cannot read and writes nothing", async () => {
		await renderApp(<AddScreen />, { env });
		await screen.findByRole("radio", { name: "Checking" });

		await fireEvent.changeText(screen.getByLabelText("Amount"), "1.005");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByRole("alert")).toHaveTextContent("Enter an amount like 12.50 or 12,50.");
		expect(await rows()).toHaveLength(0);
	});

	it("refuses an impossible date", async () => {
		await renderApp(<AddScreen />, { env });
		await screen.findByRole("radio", { name: "Checking" });
		await fireEvent.changeText(screen.getByLabelText("Amount"), "5");
		await fireEvent.changeText(screen.getByLabelText("Date"), "2027-02-30");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findByText("Enter a valid date.")).toBeTruthy();
		expect(await rows()).toHaveLength(0);
	});

	it("records income with the income categories", async () => {
		await renderApp(<AddScreen />, { env });
		await screen.findByRole("radio", { name: "Checking" });
		await fireEvent.press(screen.getByRole("radio", { name: "Income" }));
		expect(await screen.findByRole("radio", { name: "Salary" })).toBeTruthy();
		expect(screen.queryByRole("radio", { name: "Groceries" })).toBeNull();

		await fireEvent.changeText(screen.getByLabelText("Amount"), "2500");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		await waitFor(async () => expect(await rows()).toHaveLength(1));
		expect((await rows())[0]).toMatchObject({ kind: "income", amount_cents: 250000 });
	});

	it("offers the most recently used categories first", async () => {
		const dining = await env.ledger.createCategory({ name: "Dining", kind: "expense", parent_uid: null, color: "", icon: "" });
		await env.repo.create("transaction", txFields(checking, { category_uid: dining.uid, occurred_on: "2027-03-10" }));
		await renderApp(<AddScreen />, { env });
		const chips = await screen.findAllByRole("radio");
		const names = chips.map((c) => c.props.accessibilityLabel as string);
		expect(names.indexOf("Dining")).toBeLessThan(names.indexOf("Groceries"));
	});

	it("makes a transfer between two accounts that moves balances and no income or expense", async () => {
		await renderApp(<AddScreen />, { env });
		await screen.findByRole("radio", { name: "Checking" });
		await fireEvent.press(screen.getByRole("radio", { name: "Transfer" }));
		expect(await screen.findByText("To account")).toBeTruthy();

		await fireEvent.changeText(screen.getByLabelText("Amount"), "100");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		await waitFor(async () => expect(await rows()).toHaveLength(2));
		const legs = await rows();
		expect(legs.find((l) => l.kind === "expense")?.account_uid).toBe(checking);
		expect(legs.find((l) => l.kind === "income")?.account_uid).toBe(savings);
		expect(legs.every((l) => l.category_uid === null && l.amount_cents === 10000)).toBe(true);
		expect(await screen.findByText("Transfer saved")).toBeTruthy();
	});

	it("edits a stored transaction and queues only the changed fields", async () => {
		const tx = await env.repo.create("transaction", txFields(checking, { amount_cents: 1250, description: "lunch", occurred_on: "2027-03-12" }));
		await renderApp(<AddScreen editUid={tx.uid} />, { env });
		await waitFor(() => expect(screen.getByLabelText("Amount").props.value).toBe("12.50"));

		await fireEvent.changeText(screen.getByLabelText("Amount"), "13,00");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		await waitFor(async () => expect((await rows())[0]?.amount_cents).toBe(1300));
		const last = (await env.repo.outbox()).at(-1);
		expect(last).toMatchObject({ op: "upsert", fields: { amount_cents: 1300 } });
		expect(Object.keys(last?.fields ?? {})).toEqual(["amount_cents"]);
	});
});

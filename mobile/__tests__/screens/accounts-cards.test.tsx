import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { createCard } from "../../src/domain/cards";
import { statementRefIn } from "../../src/domain/statements";
import { AccountsScreen } from "../../src/screens/AccountsScreen";
import { CardsScreen } from "../../src/screens/CardsScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
beforeEach(async () => {
	env = await openEnv();
});

describe("Accounts", () => {
	it("shows the empty state with a way to start", async () => {
		await renderApp(<AccountsScreen />, { env });
		expect(await screen.findByText("No accounts yet.")).toBeTruthy();
		expect(screen.getByRole("button", { name: "New account" })).toBeTruthy();
	});

	it("lists accounts with their balances", async () => {
		const acc = (await env.repo.create("account", { ...ACCOUNT, name: "Main", opening_cents: 10000 })).uid;
		await env.repo.create("transaction", txFields(acc, { amount_cents: 2550, occurred_on: "2027-03-01" }));
		await renderApp(<AccountsScreen />, { env });
		expect(await screen.findByLabelText("74 dollars 50 cents")).toHaveTextContent("$74.50");
		expect(screen.getByRole("button", { name: "Main, Checking" })).toBeTruthy();
	});

	it("creates, edits, archives and deletes an account, deleting its transactions", async () => {
		await renderApp(<AccountsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: "New account" }));
		await fireEvent.changeText(screen.getByLabelText("Name"), "Wallet");
		await fireEvent.press(screen.getByRole("radio", { name: "Cash" }));
		await fireEvent.changeText(screen.getByLabelText("Opening balance"), "1.234,56");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		const row = await screen.findByRole("button", { name: /^Wallet, Cash$/ });
		const [account] = await env.repo.list("account");
		expect(account).toMatchObject({ name: "Wallet", kind: "cash", opening_cents: 123456 });

		await fireEvent.press(row);
		await fireEvent.changeText(screen.getByLabelText("Name"), "Pocket");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		await screen.findByRole("button", { name: /^Pocket, / });
		const edit = (await env.repo.outbox()).at(-1);
		expect(edit).toMatchObject({ op: "upsert", fields: { name: "Pocket" } });

		await env.repo.create("transaction", txFields(account?.uid ?? "", { amount_cents: 100 }));
		await fireEvent.press(screen.getByRole("button", { name: /^Pocket, / }));
		await fireEvent.press(screen.getByRole("button", { name: "Archive" }));
		await screen.findByRole("button", { name: /^Pocket \(Archived\), / });
		expect((await env.repo.get("account", account?.uid ?? ""))?.archived_at).not.toBeNull();

		await fireEvent.press(screen.getByRole("button", { name: /^Pocket \(Archived\), / }));
		await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
		expect(screen.getByText("Delete this item?")).toBeTruthy();
		await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
		await waitFor(async () => expect(await env.repo.list("account")).toHaveLength(0));
		expect(await env.repo.list("transaction")).toHaveLength(0);
	});

	it("refuses a missing name and an unreadable balance", async () => {
		await renderApp(<AccountsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: "New account" }));
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findByText("Required")).toBeTruthy();
		await fireEvent.changeText(screen.getByLabelText("Name"), "X");
		await fireEvent.changeText(screen.getByLabelText("Opening balance"), "1.005");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findAllByText("Enter an amount like 12.50 or 12,50.")).not.toHaveLength(0);
		expect(await env.repo.list("account")).toHaveLength(0);
	});

	it("opens the transfer form from two accounts and moves the money", async () => {
		const a = (await env.repo.create("account", { ...ACCOUNT, name: "A", opening_cents: 5000 })).uid;
		await env.repo.create("account", { ...ACCOUNT, name: "B", position: 1 });
		await renderApp(<AccountsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: "Transfer between accounts" }));
		expect(await screen.findByRole("header", { name: "New transfer" })).toBeTruthy();
		await screen.findByRole("radio", { name: "A" });
		await fireEvent.changeText(screen.getByLabelText("Amount"), "20");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));
		await waitFor(async () => expect(await env.repo.list("transaction")).toHaveLength(2));
		const legs = await env.repo.list("transaction");
		expect(legs.find((l) => l.kind === "expense")?.account_uid).toBe(a);
	});
});

describe("Cards", () => {
	const input = { name: "Visa", limit_cents: 500000, closing_day: 10, due_day: 20, payment_account_uid: null as string | null };

	it("creates a card with its account and shows the empty state first", async () => {
		await env.repo.create("account", { ...ACCOUNT, name: "Bank" });
		await renderApp(<CardsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: "New card" }));
		await fireEvent.changeText(screen.getByLabelText("Name"), "Visa");
		await fireEvent.changeText(screen.getByLabelText("Limit"), "5.000,00");
		await fireEvent.changeText(screen.getByLabelText("Closing day"), "10");
		await fireEvent.changeText(screen.getByLabelText("Due day"), "20");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		await screen.findByRole("button", { name: /^Visa, / });
		const [card] = await env.repo.list("credit_card");
		expect(card).toMatchObject({ limit_cents: 500000, closing_day: 10, due_day: 20 });
		const cardAccount = await env.repo.get("account", card?.account_uid ?? "");
		expect(cardAccount).toMatchObject({ name: "Visa", kind: "credit_card" });
	});

	it("refuses a closing day outside 1 to 31", async () => {
		await expect(createCard(env.db, env.repo, { ...input, closing_day: 32 })).rejects.toMatchObject({ code: "day" });
		await expect(createCard(env.db, env.repo, { ...input, limit_cents: 0 })).rejects.toMatchObject({ code: "limit" });
		await expect(createCard(env.db, env.repo, { ...input, name: " " })).rejects.toMatchObject({ code: "name" });
		expect(await env.repo.list("account")).toHaveLength(0);
	});

	it("shows the limit used, the statements with their status in words, and pays one as a transfer", async () => {
		const bank = (await env.repo.create("account", { ...ACCOUNT, name: "Bank", opening_cents: 100000 })).uid;
		const card = await createCard(env.db, env.repo, { ...input, payment_account_uid: bank });
		const ref = await statementRefIn(env.db, card.account_uid, "2027-02-20");
		await env.repo.create("transaction", txFields(card.account_uid, { amount_cents: 30000, occurred_on: "2027-02-20", statement_ref: ref }));

		await renderApp(<CardsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: /^Visa, / }));
		expect(await screen.findByText("Closed · 2027-03-10")).toBeTruthy();
		expect(screen.getByText("Limit: $5,000.00")).toBeTruthy();
		expect(screen.getAllByLabelText("300 dollars")).toHaveLength(2); // limit used and the statement total

		await fireEvent.press(screen.getByRole("button", { name: "Pay statement 2027-03-10" }));
		await fireEvent.press(await screen.findByRole("button", { name: "Pay statement" }));

		expect(await screen.findByText("Statement paid")).toBeTruthy();
		const legs = await env.repo.list("transaction");
		const payment = legs.find((l) => l.account_uid === card.account_uid && l.transfer_uid !== null);
		expect(payment).toMatchObject({ kind: "income", amount_cents: 30000, statement_ref: "2027-03-10" });
		expect(legs.find((l) => l.account_uid === bank)).toMatchObject({ kind: "expense", amount_cents: 30000 });
		expect(await screen.findByText("Paid · 2027-03-10")).toBeTruthy();
	});
});

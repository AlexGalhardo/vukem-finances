import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { AddScreen } from "../../src/screens/AddScreen";
import { TransactionsScreen } from "../../src/screens/TransactionsScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
let checking: string;
let food: string;

beforeEach(async () => {
	env = await openEnv();
	checking = (await env.repo.create("account", { ...ACCOUNT, name: "Checking" })).uid;
	food = (await env.ledger.createCategory({ name: "Groceries", kind: "expense", parent_uid: null, color: "", icon: "" })).uid;
	await env.repo.create("transaction", txFields(checking, { description: "Coffee beans", amount_cents: 1250, category_uid: food, occurred_on: "2027-03-10" }));
	await env.repo.create("transaction", txFields(checking, { description: "Rent", amount_cents: 90000, occurred_on: "2027-03-01" }));
	await env.repo.create("transaction", txFields(checking, { description: "Old fee 50%", amount_cents: 500, occurred_on: "2027-01-05" }));
});

describe("Transactions", () => {
	it("lists the newest first with signed, spoken amounts", async () => {
		await renderApp(<TransactionsScreen />, { env });
		expect(await screen.findByRole("button", { name: /^Coffee beans, / })).toBeTruthy();
		const titles = screen.getAllByRole("button").map((b) => b.props.accessibilityLabel as string);
		const order = ["Coffee beans", "Rent", "Old fee 50%"].map((t) => titles.findIndex((l) => l.startsWith(t)));
		expect(order).toEqual([...order].sort((a, b) => a - b));
		expect(screen.getByLabelText("Expense minus 12 dollars 50 cents")).toHaveTextContent("-$12.50");
	});

	it("filters as the person types in the search field", async () => {
		await renderApp(<TransactionsScreen />, { env });
		await screen.findByRole("button", { name: /^Rent, / });

		await fireEvent.changeText(screen.getByLabelText("Search transactions"), "coffee");
		await waitFor(() => expect(screen.queryByRole("button", { name: /^Rent, / })).toBeNull());
		expect(screen.getByRole("button", { name: /^Coffee beans, / })).toBeTruthy();

		await fireEvent.changeText(screen.getByLabelText("Search transactions"), "Groceries");
		expect(await screen.findByRole("button", { name: /^Coffee beans, / })).toBeTruthy();

		await fireEvent.changeText(screen.getByLabelText("Search transactions"), "50%");
		await screen.findByRole("button", { name: /^Old fee 50%, / });
		expect(screen.queryByRole("button", { name: /^Rent, / })).toBeNull();

		await fireEvent.changeText(screen.getByLabelText("Search transactions"), "zzz");
		expect(await screen.findByText("No transactions match.")).toBeTruthy();
	});

	it("filters by category and by period", async () => {
		await renderApp(<TransactionsScreen />, { env });
		await screen.findByRole("button", { name: /^Rent, / });

		await fireEvent.press(screen.getByRole("radio", { name: "Groceries" }));
		await waitFor(() => expect(screen.queryByRole("button", { name: /^Rent, / })).toBeNull());
		expect(screen.getByRole("button", { name: /^Coffee beans, / })).toBeTruthy();

		await fireEvent.press(screen.getByRole("radio", { name: "Category: All" }));
		await fireEvent.press(screen.getByRole("radio", { name: "This month" }));
		await waitFor(() => expect(screen.queryByRole("button", { name: /^Old fee/ })).toBeNull());
		expect(screen.getByRole("button", { name: /^Rent, / })).toBeTruthy();
	});

	it("updates when a transaction is added from the Add form", async () => {
		await renderApp(
			<>
				<TransactionsScreen />
				<AddScreen />
			</>,
			{ env },
		);
		await screen.findByRole("button", { name: /^Rent, / });
		expect(screen.queryByRole("button", { name: /^Gym, / })).toBeNull();

		await fireEvent.changeText(screen.getByLabelText("Amount"), "30");
		await fireEvent.changeText(screen.getByLabelText("Note"), "Gym");
		await fireEvent.press(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByRole("button", { name: /^Gym, / })).toBeTruthy();
	});

	it("deletes a transaction with a visible button and brings it back with undo", async () => {
		await renderApp(<TransactionsScreen />, { env });
		await fireEvent.press(await screen.findByRole("button", { name: /^Rent, / }));
		await fireEvent.press(screen.getByRole("button", { name: "Delete" }));

		await waitFor(() => expect(screen.queryByRole("button", { name: /^Rent, / })).toBeNull());
		expect(screen.getByText("Deleted")).toBeTruthy();

		await fireEvent.press(screen.getByRole("button", { name: "Undo" }));
		expect(await screen.findByRole("button", { name: /^Rent, / })).toBeTruthy();
	});

	it("opens a row to edit it, and a transfer leg offers delete only", async () => {
		const savings = (await env.repo.create("account", { ...ACCOUNT, name: "Savings", position: 1 })).uid;
		await env.ledger.createTransfer({ from_account_uid: checking, to_account_uid: savings, amount_cents: 700, date: "2027-03-14", description: "move" });
		await renderApp(<TransactionsScreen />, { env });

		await fireEvent.press((await screen.findAllByRole("button", { name: /^move, / }))[0] as never);
		expect(screen.queryByRole("button", { name: "Edit" })).toBeNull();
		expect(screen.getByRole("button", { name: "Delete" })).toBeTruthy();
		await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(screen.queryByRole("button", { name: /^move, / })).toBeNull());
		// Both legs are gone.
		expect(await env.repo.list("transaction")).toHaveLength(3);
	});

	// FlatList renders its rows in windows, so the test checks the page size through the button, which shows only while a full page came back.
	it("loads a page at a time with a visible Load more button", async () => {
		for (let i = 0; i < 35; i++) {
			await env.repo.create("transaction", txFields(checking, { description: `bulk ${i}`, occurred_on: "2027-02-01" }));
		}
		await renderApp(<TransactionsScreen />, { env });
		// FlatList renders rows in windows, so the page size is checked through the button: it shows only while a full page came back.
		await screen.findByRole("button", { name: "Load more" });
		await fireEvent.press(screen.getByRole("button", { name: "Load more" }));
		await waitFor(() => expect(screen.queryByRole("button", { name: "Load more" })).toBeNull());
	});
});

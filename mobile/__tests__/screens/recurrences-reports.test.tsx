import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { RecurrencesScreen } from "../../src/screens/RecurrencesScreen";
import { ReportsScreen } from "../../src/screens/ReportsScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";

let env: Env;
let acc: string;
beforeEach(async () => {
  env = await openEnv();
  acc = (await env.repo.create("account", ACCOUNT)).uid;
});

describe("Recurrences", () => {
  it("creates a rule, generates what is due and lists the upcoming occurrences", async () => {
    await renderApp(<RecurrencesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "New recurring transaction" }));
    await screen.findByRole("radio", { name: "Monthly" });

    await fireEvent.changeText(screen.getByLabelText("Amount"), "1.234,56");
    await fireEvent.changeText(screen.getByLabelText("Description"), "Rent");
    await fireEvent.changeText(screen.getByLabelText("Starts on"), "2027-01-31");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));

    await screen.findByRole("button", { name: /^Rent, Monthly · 2027-01-31/ });
    const [rule] = await env.repo.list("recurrence");
    expect(rule).toMatchObject({
      amount_cents: 123456,
      frequency: "monthly",
      interval: 1,
      generated_through: "2027-02-28",
    });
    const made = await env.db.all<{ occurrence_on: string }>(
      "SELECT occurrence_on FROM transactions ORDER BY occurrence_on",
    );
    expect(made.map((m) => m.occurrence_on)).toEqual(["2027-01-31", "2027-02-28"]);
    expect(await screen.findByText("2027-03-31")).toBeTruthy();
  });

  it("refuses a rule with a bad interval or date", async () => {
    await renderApp(<RecurrencesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "New recurring transaction" }));
    await fireEvent.changeText(screen.getByLabelText("Amount"), "10");
    await fireEvent.changeText(screen.getByLabelText("Every"), "0");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();

    await fireEvent.changeText(screen.getByLabelText("Every"), "1");
    await fireEvent.changeText(screen.getByLabelText("Starts on"), "2027-02-30");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Enter a valid date.")).toBeTruthy();
    expect(await env.repo.list("recurrence")).toHaveLength(0);
  });

  it("pauses and deletes a rule", async () => {
    const rule = await env.repo.create("recurrence", {
      account_uid: acc,
      category_uid: null,
      kind: "expense",
      amount_cents: 500,
      description: "Gym",
      frequency: "weekly",
      interval: 1,
      start_date: "2027-03-20",
      end_date: null,
      max_count: null,
      generated_through: null,
      active: true,
    });
    await renderApp(<RecurrencesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: /^Gym, / }));
    await fireEvent.press(screen.getByRole("button", { name: "Archive" }));
    await screen.findByRole("button", { name: /^Gym, .*Archived/ });
    expect((await env.repo.get("recurrence", rule.uid))?.active).toBe(false);

    await fireEvent.press(screen.getByRole("button", { name: /^Gym, / }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await waitFor(async () => expect(await env.repo.list("recurrence")).toHaveLength(0));
  });

  it("records an installment purchase in parts, the remainder on the first", async () => {
    await renderApp(<RecurrencesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Installment purchase" }));
    await screen.findByRole("radio", { name: "Checking" });
    await fireEvent.changeText(screen.getByLabelText("Total"), "100");
    await fireEvent.changeText(screen.getByLabelText("Number of parts"), "3");
    await fireEvent.changeText(screen.getByLabelText("Description"), "TV");
    await fireEvent.changeText(screen.getByLabelText("Date"), "2027-03-31");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));

    await waitFor(async () => expect(await env.repo.list("transaction")).toHaveLength(3));
    const parts = (await env.repo.list("transaction")).sort(
      (a, b) => (a.installment_no ?? 0) - (b.installment_no ?? 0),
    );
    expect(parts.map((p) => [p.amount_cents, p.occurred_on])).toEqual([
      [3334, "2027-03-31"],
      [3333, "2027-04-30"],
      [3333, "2027-05-31"],
    ]);
  });
});

describe("Reports", () => {
  beforeEach(async () => {
    const food = await env.ledger.createCategory({
      name: "Food",
      kind: "expense",
      parent_uid: null,
      color: "",
      icon: "",
    });
    await env.repo.create(
      "transaction",
      txFields(acc, { kind: "income", amount_cents: 300000, occurred_on: "2027-03-01" }),
    );
    await env.repo.create(
      "transaction",
      txFields(acc, { category_uid: food.uid, amount_cents: 7500, occurred_on: "2027-03-05" }),
    );
    await env.repo.create("transaction", txFields(acc, { amount_cents: 2500, occurred_on: "2027-03-06" }));
    await env.repo.create("transaction", txFields(acc, { amount_cents: 999, occurred_on: "2027-02-06" }));
  });

  it("shows this month's cash flow with signed, spoken amounts", async () => {
    await renderApp(<ReportsScreen />, { env });
    expect(await screen.findByRole("header", { name: "2027-03" })).toBeTruthy();
    expect(screen.getByLabelText("Income 3000 dollars")).toHaveTextContent("+$3,000.00");
    expect(screen.getByLabelText("Expense minus 100 dollars")).toHaveTextContent("-$100.00");
    expect(screen.getAllByLabelText("2900 dollars").length).toBeGreaterThan(0);
  });

  it("has a table view with the same numbers and no bars", async () => {
    await renderApp(<ReportsScreen />, { env });
    await screen.findByRole("header", { name: "2027-03" });
    const bars = (): number => (JSON.stringify(screen.toJSON()).match(/no-hide-descendants/gu) ?? []).length;
    expect(bars()).toBeGreaterThan(0);

    await fireEvent.press(screen.getByRole("radio", { name: "Table" }));
    expect(screen.getByLabelText("Income 3000 dollars")).toBeTruthy();
    expect(bars()).toBe(0);
  });

  it("changes the period", async () => {
    await renderApp(<ReportsScreen />, { env });
    await screen.findByRole("header", { name: "2027-03" });
    await fireEvent.press(screen.getByRole("radio", { name: "Last 3 months" }));
    expect(await screen.findByRole("header", { name: "2027-01" })).toBeTruthy();
    expect(screen.getByRole("header", { name: "2027-02" })).toBeTruthy();
    expect(screen.getByLabelText("Expense minus 9 dollars 99 cents")).toBeTruthy();
  });

  it("breaks spending down by category with the share and an uncategorized row", async () => {
    await renderApp(<ReportsScreen />, { env });
    await fireEvent.press(await screen.findByRole("radio", { name: "By category" }));
    expect(await screen.findByText("Food")).toBeTruthy();
    expect(screen.getByText("Share 75%")).toBeTruthy();
    expect(screen.getByText("Uncategorized")).toBeTruthy();
    expect(screen.getByText("Share 25%")).toBeTruthy();
    expect(screen.getByLabelText("Expense minus 100 dollars")).toBeTruthy();

    await fireEvent.press(screen.getByRole("radio", { name: "Income" }));
    expect(await screen.findByText("Uncategorized")).toBeTruthy();
    expect(screen.getAllByLabelText("Income 3000 dollars")).toHaveLength(2); // the row and the total
  });

  it("shows an empty state for a period without data", async () => {
    const empty = await openEnv();
    await renderApp(<ReportsScreen />, { env: empty });
    expect(await screen.findByText("No data in this period.")).toBeTruthy();
  });
});

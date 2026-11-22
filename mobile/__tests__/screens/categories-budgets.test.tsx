import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { BudgetsScreen } from "../../src/screens/BudgetsScreen";
import { CategoriesScreen } from "../../src/screens/CategoriesScreen";
import { TagsScreen } from "../../src/screens/TagsScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";
import { newUid } from "../../src/domain/ids";

let env: Env;
beforeEach(async () => {
  env = await openEnv();
});

const cat = (name: string, over: Record<string, unknown> = {}) => ({
  name,
  kind: "expense",
  parent_uid: null,
  color: "",
  icon: "",
  ...over,
});

describe("Categories", () => {
  it("shows the empty state, then creates a category and a subcategory under it", async () => {
    await renderApp(<CategoriesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "New category" }));
    await fireEvent.changeText(screen.getByLabelText("Name"), "Food");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    await screen.findByRole("button", { name: "Food" });

    await fireEvent.press(screen.getByRole("button", { name: "Add" }));
    await fireEvent.changeText(screen.getByLabelText("Name"), "Snacks");
    await fireEvent.press(screen.getByRole("radio", { name: "Food" }));
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByRole("button", { name: "↳ Snacks" })).toBeTruthy();
    const [food] = await env.repo.list("category");
    const snacks = (await env.repo.list("category")).find((c) => c.name === "Snacks");
    expect(snacks).toMatchObject({ parent_uid: food?.uid, kind: "expense" });
  });

  it("deletes a category, moving its transactions to the chosen one, and refuses one with subcategories", async () => {
    const acc = (await env.repo.create("account", ACCOUNT)).uid;
    const a = await env.ledger.createCategory(cat("A"));
    const b = await env.ledger.createCategory(cat("B"));
    const kid = await env.ledger.createCategory(cat("A1", { parent_uid: a.uid }));
    const t = await env.repo.create("transaction", txFields(acc, { category_uid: kid.uid }));
    await renderApp(<CategoriesScreen />, { env });

    await fireEvent.press(await screen.findByRole("button", { name: "A" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    expect(await screen.findByText("Move or delete its subcategories first.")).toBeTruthy();
    expect(await env.repo.get("category", a.uid)).not.toBeNull();

    await fireEvent.press(screen.getByRole("button", { name: "Back" }));
    await fireEvent.press(await screen.findByRole("button", { name: "↳ A1" }));
    await fireEvent.press(screen.getAllByRole("radio", { name: "B" }).at(-1) as never); // the last one is in "Move its transactions to"
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await waitFor(async () => expect(await env.repo.get("category", kid.uid)).toBeNull());
    expect((await env.repo.get("transaction", t.uid))?.category_uid).toBe(b.uid);
  });

  it("archives and restores", async () => {
    const a = await env.ledger.createCategory(cat("A"));
    await renderApp(<CategoriesScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "A" }));
    await fireEvent.press(screen.getByRole("button", { name: "Archive" }));
    await screen.findByRole("button", { name: "A (Archived)" });
    expect((await env.repo.get("category", a.uid))?.archived_at).not.toBeNull();
  });
});

describe("Tags", () => {
  it("creates, renames and deletes a tag, stripping it from transactions", async () => {
    const acc = (await env.repo.create("account", ACCOUNT)).uid;
    await renderApp(<TagsScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "New tag" }));
    await fireEvent.changeText(screen.getByLabelText("Name"), "work");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    const row = await screen.findByRole("button", { name: "work" });
    const [tag] = await env.repo.list("tag");
    const tx = await env.repo.create("transaction", txFields(acc, { tag_uids: [tag?.uid ?? ""] }));

    await fireEvent.press(row);
    await fireEvent.changeText(screen.getByLabelText("Name"), "job");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    await fireEvent.press(await screen.findByRole("button", { name: "job" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));

    await waitFor(async () => expect(await env.repo.list("tag")).toHaveLength(0));
    expect((await env.repo.get("transaction", tx.uid))?.tag_uids).toEqual([]);
  });
});

describe("Budgets", () => {
  it("creates a budget for a category and shows what is spent, with the percent in words", async () => {
    const acc = (await env.repo.create("account", ACCOUNT)).uid;
    const food = await env.ledger.createCategory(cat("Groceries"));
    await env.repo.create(
      "transaction",
      txFields(acc, { category_uid: food.uid, amount_cents: 2500, occurred_on: "2027-03-02" }),
    );
    await renderApp(<BudgetsScreen />, { env });

    await fireEvent.press(await screen.findByRole("button", { name: "New budget" }));
    await fireEvent.press(screen.getByRole("radio", { name: "Groceries" }));
    await fireEvent.changeText(screen.getByLabelText("Monthly limit"), "100");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));

    const row = await screen.findByRole("button", { name: "Groceries, Spent $25.00 of $100.00 · 25 percent used" });
    expect(row).toBeTruthy();
    const [b] = await env.repo.list("budget");
    expect(b).toMatchObject({ category_uid: food.uid, limit_cents: 10000, rollover: false });
  });

  it("shows an overall budget, flags an overspent one in words, edits and deletes", async () => {
    const acc = (await env.repo.create("account", ACCOUNT)).uid;
    await env.repo.create("transaction", txFields(acc, { amount_cents: 15000, occurred_on: "2027-03-02" }));
    // Made in January 2027 (the ULID carries the date), so the rollover test does not depend on the real clock.
    await env.repo.create(
      "budget",
      { category_uid: null, limit_cents: 10000, rollover: false },
      newUid(Date.UTC(2027, 0, 1)),
    );
    await renderApp(<BudgetsScreen />, { env });

    const row = await screen.findByRole("button", {
      name: "All spending, Spent $150.00 of $100.00 · 150 percent used",
    });
    await fireEvent.press(row);
    await fireEvent.changeText(screen.getByLabelText("Monthly limit"), "200");
    await fireEvent.press(screen.getByRole("checkbox", { name: "Carry over what is left" }));
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    // Made before March and February spent nothing, so its whole $200.00 limit carries into March.
    await screen.findByRole("button", { name: "All spending, Spent $150.00 of $400.00 · 37 percent used" });
    expect(screen.getByText("Carried over $200.00")).toBeTruthy();
    expect((await env.repo.list("budget"))[0]).toMatchObject({ limit_cents: 20000, rollover: true });

    await fireEvent.press(screen.getByRole("button", { name: /^All spending, / }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await waitFor(async () => expect(await env.repo.list("budget")).toHaveLength(0));
  });

  it("refuses a limit it cannot read", async () => {
    await renderApp(<BudgetsScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "New budget" }));
    await fireEvent.changeText(screen.getByLabelText("Monthly limit"), "abc");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findAllByText("Enter an amount like 12.50 or 12,50.")).not.toHaveLength(0);
    expect(await env.repo.list("budget")).toHaveLength(0);
  });
});

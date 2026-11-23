import { expect, test } from "@playwright/test";
import { categorizationRequest, setupAI } from "./ai-helpers";
import { type FakeProvider, startFakeProvider } from "./fake-provider.mjs";

let provider: FakeProvider;

test.beforeAll(async () => {
  provider = await startFakeProvider("e2e");
});

test.afterAll(async () => {
  await provider.close();
});

test("category suggestions are reviewed with the keyboard only", async ({ page }) => {
  const stamp = Date.now();
  const first = `e2e-sugg-alpha-${stamp}`;
  const second = `e2e-sugg-beta-${stamp}`;

  // The fake model files every transaction under Groceries (or the first category it was offered).
  provider.setReply(({ user }) => {
    const request = categorizationRequest(user);
    if (!request) return "pong";
    const groceries = request.categories.find((c) => c.name === "Groceries") ?? request.categories[0];
    return JSON.stringify({
      items: request.transactions.map((t) => ({ id: t.id, category_uid: groceries?.uid ?? null, confidence: 0.8 })),
    });
  });
  await setupAI(page, provider);

  // Two uncategorized expenses, made through the dialog.
  for (const description of [first, second]) {
    await page.goto("/transactions");
    await page.getByRole("heading", { level: 1 }).waitFor();
    await page.getByRole("button", { name: "Add transaction" }).click();
    const dialog = page.getByRole("dialog", { name: "New expense" });
    await dialog.getByLabel("Amount").fill("7.50");
    await dialog.getByLabel("Description").fill(description);
    await dialog.getByLabel("Description").press("Enter");
    await expect(dialog).toBeHidden();
  }

  await page.goto("/suggestions");
  await expect(page.getByRole("heading", { level: 1, name: "Suggestions" })).toBeVisible();
  const categorize = page.getByRole("button", { name: "Categorize uncategorized" });
  await categorize.click();

  const alpha = page.getByRole("option", { name: new RegExp(first) });
  const beta = page.getByRole("option", { name: new RegExp(second) });
  await expect(alpha).toBeVisible();
  await expect(beta).toBeVisible();
  await expect(alpha).toContainText("From AI");
  // The rows show up before the request ends, and the page ignores keys until then.
  await expect(categorize).toBeEnabled();

  // Keyboard only: focus the first row, accept it with Enter, reject the second with Backspace.
  await alpha.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("option", { name: new RegExp(first) })).toHaveCount(0);
  await expect(page.getByRole("status").filter({ hasText: "Category applied." }).first()).toBeAttached();

  await beta.focus();
  await page.keyboard.press("Backspace");
  await expect(page.getByRole("option", { name: new RegExp(second) })).toHaveCount(0);

  // The accepted transaction now has its category; the rejected one has none.
  await page.goto(`/transactions?q=${first}`);
  await expect(page.getByRole("row").filter({ hasText: first })).toContainText("Groceries");
  await page.goto(`/transactions?q=${second}`);
  await expect(page.getByRole("row").filter({ hasText: second })).not.toContainText("Groceries");
});

test("Shift+A accepts every suggestion at or above the chosen confidence", async ({ page }) => {
  const description = `e2e-sugg-gamma-${Date.now()}`;
  provider.setReply(({ user }) => {
    const request = categorizationRequest(user);
    if (!request) return "pong";
    const category = request.categories.find((c) => c.name === "Groceries") ?? request.categories[0];
    return JSON.stringify({
      items: request.transactions.map((t) => ({ id: t.id, category_uid: category?.uid ?? null, confidence: 0.95 })),
    });
  });
  await setupAI(page, provider);

  await page.goto("/transactions");
  await page.getByRole("button", { name: "Add transaction" }).click();
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("3.20");
  await dialog.getByLabel("Description").fill(description);
  await dialog.getByLabel("Description").press("Enter");
  await expect(dialog).toBeHidden();

  await page.goto("/suggestions");
  const categorize = page.getByRole("button", { name: "Categorize uncategorized" });
  await categorize.click();
  const row = page.getByRole("option", { name: new RegExp(description) });
  await expect(row).toBeVisible();
  await expect(categorize).toBeEnabled();

  await row.focus();
  await page.keyboard.press("Shift+A");
  await expect(page.getByRole("option", { name: new RegExp(description) })).toHaveCount(0);
});

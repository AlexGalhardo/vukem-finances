import { expect, test } from "@playwright/test";
import { categorizationRequest, setupAI } from "./ai-helpers";
import { type FakeProvider, startFakeProvider } from "./fake-provider.mjs";
import { blurActive } from "./login";

let provider: FakeProvider;

test.beforeAll(async () => {
  provider = await startFakeProvider("e2e-screens-a");
});

test.afterAll(async () => {
  await provider.close();
});

test("a card statement carries the targets of its tutorial steps and the card-day hints", async ({ page }) => {
  const name = `E2E Tour Visa ${Date.now()}`;
  await page.goto("/cards");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Card name").fill(name);
  await page.getByLabel("Credit limit").fill("1500.00");
  await page.getByLabel("Closing day", { exact: true }).fill("10");
  await page.getByLabel("Due day", { exact: true }).fill("25");
  await page.getByRole("button", { name: "Add card" }).click();
  const tile = page.getByRole("listitem").filter({ hasText: name });
  await expect(tile).toBeVisible();
  await expect(page.locator('[data-tour="cards-list"]')).toBeAttached();
  await expect(page.locator('[data-tour="cards-new"]')).toBeAttached();

  await page.getByRole("button", { name: "About Closing day" }).click();
  await expect(page.getByRole("dialog").getByText("Purchases after this day go to the next statement.")).toBeVisible();
  await page.keyboard.press("Escape");

  await tile.getByRole("link", { name }).click();
  await expect(page.getByRole("heading", { level: 1, name })).toBeVisible();
  for (const target of ["statement-picker", "statement-totals", "statement-pay"]) {
    await expect(page.locator(`[data-tour="${target}"]`), target).toBeAttached();
  }
  await expect(page.getByRole("button", { name: "About Limit used" })).toBeVisible();

  await page.getByRole("button", { name: `Delete ${name}` }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page).toHaveURL(/\/cards$/);
});

test("the suggestions page carries its targets and the tutorial anchors to the list", async ({ page }) => {
  const description = `e2e-tour-sugg-${Date.now()}`;
  provider.setReply(({ user }) => {
    const request = categorizationRequest(user);
    if (!request) return "pong";
    const category = request.categories.find((c) => c.name === "Groceries") ?? request.categories[0];
    return JSON.stringify({
      items: request.transactions.map((t) => ({ id: t.id, category_uid: category?.uid ?? null, confidence: 0.9 })),
    });
  });
  await setupAI(page, provider);

  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByRole("button", { name: "Add transaction" }).click();
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("4.10");
  await dialog.getByLabel("Description").fill(description);
  await dialog.getByLabel("Description").press("Enter");
  await expect(dialog).toBeHidden();

  await page.goto("/suggestions");
  await page.getByRole("button", { name: "Categorize uncategorized" }).click();
  const row = page.getByRole("option", { name: new RegExp(description) });
  await expect(row).toBeVisible();
  for (const target of ["suggestions-list", "suggestions-actions", "suggestions-bulk"]) {
    await expect(page.locator(`[data-tour="${target}"]`), target).toBeAttached();
  }

  await page.getByRole("button", { name: "Help", exact: true }).click();
  await page.getByRole("menuitem", { name: /Help for this page/ }).click();
  await page.getByRole("button", { name: "Show me this screen", exact: true }).click();
  await expect(page.locator('[data-tour="suggestions-list"]')).toHaveAttribute("data-tour-active", "");
  await page.keyboard.press("Escape");

  await row.focus();
  await page.keyboard.press("Shift+A");
  await expect(page.getByRole("option", { name: new RegExp(description) })).toHaveCount(0);
});

test.describe("on a phone", () => {
  test.use({ viewport: { width: 390, height: 800 } });

  test("the transaction filters are a sheet, / and n keep working and the list reads as rows", async ({ page }) => {
    await page.goto("/transactions");
    await page.getByRole("heading", { level: 1 }).waitFor();
    await expect(page.getByLabel("Category", { exact: true })).toHaveCount(0);

    await page.getByRole("button", { name: "Filters", exact: true }).click();
    const sheet = page.getByRole("dialog", { name: "Filters" });
    await expect(sheet).toBeVisible();
    await sheet.getByLabel("Type", { exact: true }).selectOption("expense");
    await sheet.getByRole("button", { name: "Apply filters" }).click();
    await expect(sheet).toHaveCount(0);
    await expect(page).toHaveURL(/kind=expense/);
    await expect(page.getByRole("button", { name: "Filters 1" })).toBeVisible();

    await blurActive(page);
    await page.keyboard.press("/");
    await expect(page.getByRole("searchbox")).toBeFocused();
    await blurActive(page);
    await page.keyboard.press("n");
    await page.keyboard.press("e");
    await expect(page.getByRole("dialog", { name: "New expense" })).toBeVisible();
    await page.keyboard.press("Escape");

    // Below md the table is one list: the header row is only for screen readers.
    const header = page.locator('[data-tour="transactions-table"] thead');
    await expect(header).toHaveCSS("position", "absolute");
  });
});

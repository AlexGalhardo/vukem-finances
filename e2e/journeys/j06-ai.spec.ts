import { expect, test } from "@playwright/test";
import { categorizationRequest, setupAI } from "../ai-helpers";
import { type FakeProvider, startFakeProvider } from "../fake-provider.mjs";
import { blurActive } from "../login";

// Journey 6: AI categorization and the assistant, against the fake provider only (never a real one).
let provider: FakeProvider;

test.beforeAll(async () => {
  provider = await startFakeProvider("e2e-j6");
});

test.afterAll(async () => {
  await provider.close();
});

test("a suggestion is reviewed and applied, then the assistant answers from the server tables", async ({ page }) => {
  const description = `j6-uncategorized-${Date.now()}`;
  provider.setReply(({ system, user }) => {
    if (system.startsWith("You plan")) {
      return JSON.stringify({ queries: [{ report: "cash_flow", from: "2026-01-01", to: "2026-12-31" }] });
    }
    const request = categorizationRequest(user);
    if (request) {
      const category = request.categories.find((c) => c.name === "Groceries") ?? request.categories[0];
      return JSON.stringify({
        items: request.transactions.map((t) => ({ id: t.id, category_uid: category?.uid ?? null, confidence: 0.8 })),
      });
    }
    return JSON.stringify({ answer: "Your cash flow is in the table below.", figures: [] });
  });
  await setupAI(page, provider);

  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByRole("button", { name: "Add transaction" }).click();
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("12.00");
  await dialog.getByLabel("Description").fill(description);
  await dialog.getByLabel("Description").press("Enter");
  await expect(dialog).toBeHidden();

  await page.goto("/suggestions");
  const categorize = page.getByRole("button", { name: "Categorize uncategorized" });
  await categorize.click();
  const row = page.getByRole("option", { name: new RegExp(description) });
  await expect(row).toBeVisible();
  // The row shows up before the request ends, and the page ignores keys until then.
  await expect(categorize).toBeEnabled();
  await row.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("option", { name: new RegExp(description) })).toHaveCount(0);

  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("a");
  await page.keyboard.press("s");
  const assistant = page.getByRole("dialog", { name: "Finance assistant" });
  await expect(assistant).toBeVisible();
  await assistant.getByLabel("Your question").fill("How is my cash flow this year?");
  await assistant.getByRole("button", { name: "Ask" }).click();
  await expect(assistant.getByText("Your cash flow is in the table below.")).toBeVisible({ timeout: 20_000 });
});

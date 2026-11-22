import { expect, test } from "@playwright/test";
import { setupAI } from "./ai-helpers";
import { type FakeProvider, startFakeProvider } from "./fake-provider.mjs";
import { blurActive } from "./login";

let provider: FakeProvider;

test.beforeAll(async () => {
  provider = await startFakeProvider("e2e");
});

test.afterAll(async () => {
  await provider.close();
});

test("a s opens the assistant outside a field, answers in plain text and shows the server's tables", async ({
  page,
}) => {
  provider.setReply(({ system }) =>
    system.startsWith("You plan")
      ? JSON.stringify({ queries: [{ report: "cash_flow", from: "2026-01-01", to: "2026-12-31" }] })
      : JSON.stringify({ answer: "Your cash flow is below. <b>not bold</b> [link](http://evil.test)", figures: [] }),
  );
  await setupAI(page, provider);

  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("a");
  await page.keyboard.press("s");

  const dialog = page.getByRole("dialog", { name: "Finance assistant" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText(/sent to your configured AI provider/)).toBeVisible();

  const input = dialog.getByLabel("Your question");
  await expect(input).toBeFocused();
  await input.fill("How is my cash flow this year?");
  await dialog.getByRole("button", { name: "Ask" }).click();

  const log = dialog.getByRole("log");
  // The model's text is plain text: no markup is rendered, and no link is made of it.
  await expect(log).toContainText("<b>not bold</b> [link](http://evil.test)");
  await expect(log.locator("b")).toHaveCount(0);
  await expect(log.getByRole("link")).toHaveCount(0);
  // The tables and the "Based on" line come from the server.
  await expect(log.getByRole("table")).toBeVisible();
  await expect(log.getByText(/Based on: cash flow, 2026-01-01 to 2026-12-31/)).toBeVisible();
  await expect(log.getByText(/authoritative/)).toBeVisible();
});

test("a s typed in a field does nothing", async ({ page }) => {
  await page.goto("/accounts");
  const field = page.locator("main input[type=text], main input:not([type])").first();
  await field.focus();
  await page.keyboard.type("a s");
  await expect(field).toHaveValue("a s");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

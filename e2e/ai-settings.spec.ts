import { expect, test } from "@playwright/test";
import { type FakeProvider, startFakeProvider } from "./fake-provider.mjs";

// A made-up key; the word "placeholder" keeps secret scanners quiet. No real provider is ever called:
// the app talks to a fake OpenAI-compatible server started by this spec (personal edition, AI_ALLOW_PRIVATE_HOSTS).
const KEY = "sk-placeholder-e2e-0123456789-ABCD";

let provider: FakeProvider;

test.beforeAll(async () => {
  provider = await startFakeProvider(KEY);
});

test.afterAll(async () => {
  await provider.close();
});

test("the key is saved without showing it, tested, and AI is turned on only after consent", async ({ page }) => {
  await page.goto("/ai");
  await expect(page.getByRole("heading", { level: 1, name: "AI" })).toBeVisible();
  await expect(page.getByText("AI is off for this workspace.")).toBeVisible();

  // Save a key: the field is empty afterwards and only the last four characters are shown.
  await page.getByLabel("Provider").selectOption("openai_compatible");
  await page.getByLabel("Base URL").fill(provider.baseUrl);
  const keyField = page.getByLabel("API key");
  await expect(keyField).toHaveAttribute("type", "password");
  await expect(keyField).toHaveAttribute("autocomplete", "off");
  await keyField.fill(KEY);
  await page.getByRole("button", { name: "Save key" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Key saved." })).toBeVisible();
  await expect(page.getByLabel("API key")).toHaveValue("");
  await expect(page.getByText("ending in")).toBeVisible();
  await expect(page.locator("code", { hasText: "ABCD" })).toBeVisible();
  expect(await page.content()).not.toContain(KEY);

  // Test key: the result is announced in a live region.
  provider.setStatus(200);
  await page.getByRole("button", { name: "Test key" }).click();
  await expect(page.getByRole("status").filter({ hasText: "The key works." })).toBeVisible();

  provider.setStatus(401);
  await page.getByRole("button", { name: "Test key" }).click();
  const alert = page.getByRole("alert").filter({ hasText: "The provider rejected the key." });
  await expect(alert).toBeVisible();
  expect(await page.content()).not.toContain(KEY);
  expect(await page.content()).not.toContain("authentication_error");

  // Enabling needs the consent checkbox.
  await page.getByLabel("Turn on AI for this workspace").check();
  await page.getByRole("button", { name: "Save settings" }).click();
  const dialog = page.getByRole("dialog", { name: "What is sent to your provider" });
  await expect(dialog).toBeVisible();
  const confirm = dialog.getByRole("button", { name: "Turn on AI" });
  await expect(confirm).toBeDisabled();
  await dialog.getByLabel("I understand what is sent and want to turn AI on.").check();
  await expect(confirm).toBeEnabled();
  await confirm.click();
  await expect(page.getByRole("status").filter({ hasText: "AI settings saved." })).toBeVisible();
  await expect(page.getByText("AI is on for this workspace.")).toBeVisible();
});

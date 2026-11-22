import { expect, type Page } from "@playwright/test";
import type { FakeProvider } from "./fake-provider.mjs";

// A made-up key; the word "placeholder" keeps secret scanners quiet. No real provider is ever called.
export const KEY = "sk-placeholder-e2e-features-0123456789-WXYZ";

/**
 * Points the workspace at the fake provider and turns on AI with both features, auto-apply off. Safe to run when an
 * earlier spec already turned AI on: the consent dialog only appears when AI is being turned on.
 */
export async function setupAI(page: Page, provider: FakeProvider): Promise<void> {
  await page.goto("/ai");
  await page.getByRole("heading", { level: 1, name: "AI" }).waitFor();
  await page.getByLabel("Provider").selectOption("openai_compatible");
  await page.getByLabel("Base URL").fill(provider.baseUrl);
  await page.getByLabel("API key").fill(KEY);
  await page.getByRole("button", { name: "Save key" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Key saved." })).toBeVisible();

  await page.getByLabel("Turn on AI for this workspace").check();
  await page.getByLabel("Suggest categories for transactions").check();
  await page.getByLabel("Assistant", { exact: true }).check();
  await page.getByLabel(/Apply suggestions with confidence/).uncheck();
  await page.getByRole("button", { name: "Save settings" }).click();

  const dialog = page.getByRole("dialog", { name: "What is sent to your provider" });
  if (await dialog.isVisible().catch(() => false)) {
    await dialog.getByLabel("I understand what is sent and want to turn AI on.").check();
    await dialog.getByRole("button", { name: "Turn on AI" }).click();
  }
  await expect(page.getByRole("status").filter({ hasText: "AI settings saved." })).toBeVisible();
  await expect(page.getByText("AI is on for this workspace.")).toBeVisible();
}

/** The category list and the transactions of a categorization request, as the server sent them. */
export function categorizationRequest(user: string): {
  categories: { uid: string; name: string }[];
  transactions: { id: string; description: string }[];
} | null {
  try {
    const doc = JSON.parse(user);
    return Array.isArray(doc.categories) && Array.isArray(doc.transactions) ? doc : null;
  } catch {
    return null;
  }
}

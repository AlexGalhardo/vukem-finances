import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("mod+k opens a dialog whose input has focus and role combobox", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("ControlOrMeta+k");
  const dialog = page.getByRole("dialog", { name: "Command palette" });
  await expect(dialog).toBeVisible();
  const combobox = dialog.getByRole("combobox");
  await expect(combobox).toBeFocused();
  await expect(combobox).toHaveAttribute("aria-expanded", "true");
  await expect(dialog.getByRole("listbox").getByRole("option").first()).toBeVisible();
});

test("Escape closes the palette and focus returns to the opener", async ({ page }) => {
  const opener = page.getByRole("button", { name: "Search or run a command" });
  await opener.click();
  await expect(page.getByRole("dialog", { name: "Command palette" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(opener).toBeFocused();
});

test("typing filters, Enter runs the action and focus ends on the new heading", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("ControlOrMeta+k");
  await page.keyboard.type("budg");
  await expect(page.getByRole("option")).toHaveCount(1);
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/budgets$/);
  await expect(page.getByRole("heading", { level: 1 })).toBeFocused();
});

test("a query without a match offers a transaction search", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("ControlOrMeta+k");
  await page.keyboard.type("zzqq");
  await page.getByRole("option", { name: /Search transactions for/ }).click();
  await expect(page).toHaveURL(/\/transactions\?q=zzqq/);
});

test("the palette traps focus while it is open", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("ControlOrMeta+k");
  for (let i = 0; i < 5; i++) await page.keyboard.press("Tab");
  await expect(page.getByRole("dialog", { name: "Command palette" }).locator(":focus")).toHaveCount(1);
});

test("the personal palette does not offer billing", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("ControlOrMeta+k");
  await expect(page.getByRole("option", { name: /^Billing/ })).toHaveCount(0);
});

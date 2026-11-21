import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("g t outside a field navigates to /transactions", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("g");
  await page.keyboard.press("t");
  await expect(page).toHaveURL(/\/transactions$/);
});

test("typing n e in a text field changes nothing", async ({ page }) => {
  await page.goto("/accounts");
  const field = page.locator("main input[type=text], main input:not([type])").first();
  await field.focus();
  await page.keyboard.type("n e g t");
  await expect(field).toHaveValue("n e g t");
  await expect(page).toHaveURL(/\/accounts$/);
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("no shortcut fires on a select", async ({ page }) => {
  await page.goto("/transactions");
  await page.locator("main select").first().focus();
  await page.keyboard.press("g");
  await page.keyboard.press("d");
  await page.keyboard.press("?");
  // A select may change its own value by type-ahead; the shortcuts must not navigate away or open anything.
  await expect(page).toHaveURL(/\/transactions/);
  await expect(page).not.toHaveURL(/\/dashboard/);
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("turning off single-key shortcuts disables g t but not mod+k", async ({ page }) => {
  const toggle = page.getByRole("button", { name: "Single-key shortcuts" });
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await blurActive(page);
  await page.keyboard.press("g");
  await page.keyboard.press("t");
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.keyboard.press("ControlOrMeta+k");
  await expect(page.getByRole("dialog", { name: "Command palette" })).toBeVisible();
});

test("[ toggles the sidebar", async ({ page }) => {
  const sidebar = page.locator('[data-slot="sidebar"]').first();
  await expect(sidebar).toHaveAttribute("data-state", "expanded");
  await blurActive(page);
  await page.keyboard.press("[");
  await expect(sidebar).toHaveAttribute("data-state", "collapsed");
});

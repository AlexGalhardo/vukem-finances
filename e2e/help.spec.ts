import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("? opens the shortcut help as tables and states the typing rule", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("Shift+?");
  const dialog = page.getByRole("dialog", { name: "Keyboard shortcuts" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("table")).toHaveCount(2);
  await expect(dialog.getByRole("row", { name: /Transactions/ })).toContainText("T");
  await expect(dialog.getByText("Shortcuts do not work while you type in a field.")).toBeVisible();
  await expect(dialog.getByRole("checkbox", { name: /Single-key shortcuts/ })).toBeChecked();
});

test("Escape closes the help and focus returns to the opener", async ({ page }) => {
  const opener = page.getByRole("button", { name: "Keyboard shortcuts" }).first();
  await opener.click();
  await expect(page.getByRole("dialog", { name: "Keyboard shortcuts" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(opener).toBeFocused();
});

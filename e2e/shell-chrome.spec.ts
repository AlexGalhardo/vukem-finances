import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("both theme buttons sit at the far right of the top bar and switch the theme", async ({ page }) => {
  const header = page.locator("header").first();
  const group = header.getByRole("group", { name: "Theme" });
  const light = group.getByRole("button", { name: "Switch to light mode" });
  const dark = group.getByRole("button", { name: "Switch to dark mode" });
  await expect(light).toBeVisible();
  await expect(dark).toBeVisible();

  const lastControl = header.getByRole("button").last();
  await expect(lastControl).toHaveAccessibleName("Switch to dark mode");

  await dark.click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await expect(dark).toHaveAttribute("aria-pressed", "true");
  await light.click();
  await expect(page.locator("html")).not.toHaveClass(/dark/);
  await expect(light).toHaveAttribute("aria-pressed", "true");
});

test("the sidebar footer holds only Account and Documentation, and the menu keeps the rest", async ({ page }) => {
  const footer = page.locator('[data-slot="sidebar-footer"]').first();
  await expect(footer.getByRole("listitem")).toHaveCount(2);

  const docs = footer.getByRole("link", { name: "Documentation" });
  await expect(docs).toHaveAttribute("target", "_blank");

  await footer.getByRole("button", { name: "Account" }).click();
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem", { name: "Account settings" })).toBeVisible();
  await expect(menu.getByRole("menuitem", { name: "Log out" })).toBeVisible();
  await page.screenshot({ path: "test-results/shell-chrome.png" });
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
});

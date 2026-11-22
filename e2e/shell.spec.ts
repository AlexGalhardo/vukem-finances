import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("the skip link is the first Tab stop and moves focus to #content", async ({ page }) => {
  await page.keyboard.press("Tab");
  await expect(page.getByRole("link", { name: "Skip to content" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("#content")).toBeFocused();
});

test("the sidebar is a navigation landmark and marks the current page", async ({ page }) => {
  const nav = page.getByRole("navigation", { name: "Main navigation" });
  await expect(nav.getByRole("link", { name: "Overview" })).toHaveAttribute("aria-current", "page");
  await expect(nav.getByRole("link", { name: "Transactions" })).not.toHaveAttribute("aria-current", "page");
});

test("the personal edition has no billing entry and no workspace switcher", async ({ page }) => {
  const nav = page.getByRole("navigation", { name: "Main navigation" });
  await expect(nav.getByRole("link", { name: "Billing" })).toHaveCount(0);
  await expect(page.getByLabel("Workspace", { exact: true })).toHaveCount(0);
});

test("focus moves to the page heading after a navigation", async ({ page }) => {
  await page.getByRole("navigation", { name: "Main navigation" }).getByRole("link", { name: "Accounts" }).click();
  await expect(page).toHaveURL(/\/accounts/);
  await expect(page.getByRole("heading", { level: 1 })).toBeFocused();
});

test("the sidebar groups its items and the header shows the breadcrumb", async ({ page }) => {
  const nav = page.getByRole("navigation", { name: "Main navigation" });
  for (const label of ["Money", "Planning", "Insights", "Data", "Settings"]) {
    await expect(nav.getByText(label, { exact: true })).toBeVisible();
  }
  const crumb = page.getByRole("navigation", { name: "Breadcrumb" });
  await expect(crumb.getByText("Overview")).toHaveAttribute("aria-current", "page");
});

test("h p and the Help menu open the help of the page", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("h");
  await page.keyboard.press("p");
  const sheet = page.getByRole("dialog", { name: "Overview" });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByRole("heading", { name: "Keyboard shortcuts" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(sheet).toHaveCount(0);

  await page.getByRole("button", { name: "Help", exact: true }).click();
  await page.getByRole("menuitem", { name: /Help for this page/ }).click();
  await expect(page.getByRole("dialog", { name: "Overview" })).toBeVisible();
});

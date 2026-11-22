import { expect, test } from "@playwright/test";

const screens: { path: string; targets: string[] }[] = [
  { path: "/dashboard", targets: ["dashboard-balance", "dashboard-cashflow", "dashboard-upcoming"] },
  { path: "/transactions", targets: ["transactions-search", "transactions-filters", "transactions-transfer"] },
  { path: "/accounts", targets: ["accounts-networth", "accounts-table", "accounts-new"] },
  { path: "/cards", targets: [] },
  { path: "/categories", targets: ["categories-kind", "categories-tags"] },
  { path: "/budgets", targets: ["budgets-month", "budgets-form"] },
  { path: "/goals", targets: ["goals-new"] },
  { path: "/recurrences", targets: ["recurrences-upcoming"] },
  { path: "/reports", targets: ["reports-period", "reports-tabs", "reports-export"] },
  { path: "/import", targets: ["import-upload"] },
  { path: "/suggestions", targets: [] },
];

const viewports = [
  { width: 320, height: 700 },
  { width: 768, height: 900 },
  { width: 1440, height: 900 },
];

for (const screen of screens) {
  test(`${screen.path} has its tour targets and no horizontal page overflow in both themes`, async ({ page }) => {
    for (const theme of ["light", "dark"] as const) {
      await page.addInitScript((value) => localStorage.setItem("theme", value), theme);
      for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.goto(screen.path);
        await page.getByRole("heading", { level: 1 }).waitFor();
        for (const target of screen.targets) {
          await expect(page.locator(`[data-tour="${target}"]`).first(), target).toBeAttached();
        }
        const overflow = await page.evaluate(
          () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
        );
        expect(overflow, `${screen.path} ${theme} ${viewport.width}px`).toBeLessThanOrEqual(0);
      }
    }
  });
}

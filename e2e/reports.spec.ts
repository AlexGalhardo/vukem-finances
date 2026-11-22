import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test("g r opens the reports, a period change reloads them and every chart has a table", async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("g");
  await page.keyboard.press("r");
  await expect(page).toHaveURL(/\/reports$/);
  await expect(page.getByRole("heading", { level: 1, name: "Reports" })).toBeVisible();

  await page.getByLabel("Period", { exact: true }).selectOption("last_month");
  await expect(page).toHaveURL(/period=last_month/);
  await expect(page.getByLabel("Period", { exact: true })).toHaveValue("last_month");

  // The chart is hidden from assistive technology and its table is in the page, same numbers.
  const panel = page.getByRole("tabpanel");
  await expect(panel.getByRole("button", { name: "Show as table" })).toBeVisible();
  await panel.getByRole("button", { name: "Show as table" }).click();
  await expect(panel.getByRole("table")).toBeVisible();
  await expect(panel.getByRole("columnheader", { name: "Income" })).toBeVisible();
  await panel.getByRole("button", { name: "Show as chart" }).click();

  for (const [tab, heading] of [
    ["Categories", "Category"],
    ["Net worth", "Balance"],
    ["Compare", "Previous period"],
  ] as const) {
    await page.getByRole("tab", { name: tab }).click();
    await expect(page.getByRole("tabpanel").getByRole("columnheader", { name: heading }).first()).toBeAttached();
  }
});

test("a custom period is applied with its dates", async ({ page }) => {
  await page.goto("/reports");
  await page.getByLabel("Period", { exact: true }).selectOption("custom");
  await page.getByLabel("From", { exact: true }).fill("2026-01-01");
  await page.getByLabel("To", { exact: true }).fill("2026-03-31");
  await page.getByRole("button", { name: "Apply" }).click();
  await expect(page).toHaveURL(/period=custom&from=2026-01-01&to=2026-03-31/);
  await page.getByRole("tabpanel").getByRole("button", { name: "Show as table" }).click();
  await expect(page.getByRole("tabpanel").getByRole("row")).toHaveCount(5); // header, three months, total
});

test("x downloads the transactions CSV, in a field it does nothing", async ({ page }) => {
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);

  const download = page.waitForEvent("download");
  await page.keyboard.press("x");
  const file = await download;
  expect(file.suggestedFilename()).toMatch(/^vukem-finances-transactions-\d{4}-\d{2}-\d{2}\.csv$/);

  let downloaded = false;
  page.on("download", () => {
    downloaded = true;
  });
  const search = page.getByRole("searchbox").first();
  await search.focus();
  await page.keyboard.press("x");
  await expect(search).toHaveValue("x");
  await page.waitForTimeout(500);
  expect(downloaded).toBe(false);
});

test("the Export menu offers the downloads and x works on the reports page too", async ({ page }) => {
  await page.goto("/reports");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByRole("button", { name: "Export" }).click();
  await expect(page.getByRole("menuitem", { name: "CSV of transactions" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "CSV for Excel" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Full JSON export" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Print or save as PDF" })).toBeVisible();
  await page.keyboard.press("Escape");
  await blurActive(page);

  const download = page.waitForEvent("download");
  await page.keyboard.press("x");
  expect((await download).suggestedFilename()).toMatch(/^vukem-finances-transactions-.*\.csv$/);
});

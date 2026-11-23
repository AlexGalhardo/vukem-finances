import { expect, test } from "@playwright/test";

// Journey 1 (epic 24, docs/qa/journeys.md): on a personal install, accounts, categories, a CSV import, a budget, a
// report and the exports. The install, setup and the backup and restore parts run without a browser
// (scripts/qa/journeys-nonweb.sh, step 1).
test("accounts, categories, a CSV import, a budget, a report and the exports", async ({ page }) => {
  const stamp = Date.now();
  const account = `J1 account ${stamp}`;
  const category = `J1 category ${stamp}`;

  await page.goto("/accounts");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Name", { exact: true }).first().fill(account);
  await page.getByRole("button", { name: "Add account" }).click();
  await expect(page.getByRole("row").filter({ hasText: account })).toBeVisible();

  await page.goto("/categories");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Name", { exact: true }).first().fill(category);
  await page.getByRole("button", { name: "Add category" }).click();
  await expect(page.locator("main li").filter({ hasText: category }).first()).toBeVisible();

  await page.goto("/import");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Statement file").setInputFiles("tests/fixtures/csv/journey-statement.csv");
  await page.getByRole("button", { name: "Upload" }).click();
  await page.getByRole("button", { name: "Preview the import" }).click();
  await expect(page.getByText("First rows, as they will be stored:")).toBeVisible();
  await page.getByRole("button", { name: /Import rows/ }).click();
  await expect(page.getByText(/2 imported, 0 skipped as duplicates/).first()).toBeVisible({ timeout: 15_000 });

  await page.goto("/budgets");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Monthly limit").fill("500.00");
  await page.getByRole("button", { name: "Save budget" }).click();
  await expect(page.getByText(/500[.,]00/).first()).toBeVisible();

  await page.goto("/reports?period=custom&from=2026-03-01&to=2026-03-31");
  await expect(page.getByRole("heading", { level: 1, name: "Reports" })).toBeVisible();

  const csvExport = await page.request.get("/transactions/export.csv");
  expect(csvExport.status()).toBe(200);
  expect(await csvExport.text()).toContain(`J1 rent`);
  const jsonExport = await page.request.get("/workspace/export.json");
  expect(jsonExport.status()).toBe(200);
  expect(await jsonExport.text()).toContain(`J1 market`);

  // Leave the demo data as it was: other specs count the accounts.
  await page.goto("/accounts");
  await page.getByRole("button", { name: `Delete ${account}` }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page.getByRole("row").filter({ hasText: account })).toHaveCount(0);
});

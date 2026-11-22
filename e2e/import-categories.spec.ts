import { expect, test } from "@playwright/test";

test("an OFX statement shows a preview, imports once, and the second time is all duplicates", async ({ page }) => {
  for (const round of [1, 2]) {
    await page.goto("/import");
    await page.getByRole("heading", { level: 1 }).waitFor();
    await page.getByLabel("Statement file").setInputFiles("tests/fixtures/ofx/xml.ofx");
    await page.getByRole("button", { name: "Upload" }).click();

    await expect(page.getByText("First rows, as they will be stored:")).toBeVisible();
    await expect(page.getByRole("row").filter({ hasText: "Refund été" })).toBeVisible();
    const status = round === 1 ? "New" : "Duplicate";
    await expect(
      page.getByRole("row").filter({ hasText: "Refund été" }).getByText(status, { exact: true }),
    ).toBeVisible();

    const importButton = page.getByRole("button", { name: /Import rows/ });
    if (round === 2) {
      await expect(importButton).toBeDisabled();
      await page.getByLabel("Import duplicates anyway").check();
      await expect(importButton).toBeEnabled();
      await page.getByLabel("Import duplicates anyway").uncheck();
      break;
    }
    await importButton.click();
    await expect(page.getByText("2 imported, 0 skipped as duplicates, 0 unreadable").first()).toBeVisible({
      timeout: 15_000,
    });
  }
});

test("a category gets a parent, a color and an icon, and can be archived", async ({ page }) => {
  const name = `E2E parent ${Date.now()}`;
  const child = `E2E child ${Date.now()}`;
  await page.goto("/categories");
  await page.getByRole("heading", { level: 1 }).waitFor();

  await page.getByLabel("Name", { exact: true }).first().fill(name);
  await page.getByText("Blue", { exact: true }).first().click({ force: true });
  await page.getByLabel("Icon").first().selectOption({ label: "Gift" });
  await page.getByRole("button", { name: "Add category" }).click();
  await expect(page.locator("main li").filter({ hasText: name }).first()).toBeVisible();

  await page.getByLabel("Name", { exact: true }).first().fill(child);
  await page.getByLabel("Parent category", { exact: true }).first().selectOption({ label: name });
  await page.getByRole("button", { name: "Add category" }).click();
  await expect(page.getByRole("list", { name: `Subcategories: ${name}` }).getByText(child)).toBeVisible();

  await page.getByRole("button", { name: `Archive ${name}` }).click();
  await expect(page.locator("main li").filter({ hasText: name }).first()).toHaveCount(0);
  await page.getByRole("button", { name: "Show archived" }).click();
  await expect(page.locator("main li").filter({ hasText: name }).first()).toBeVisible();
});

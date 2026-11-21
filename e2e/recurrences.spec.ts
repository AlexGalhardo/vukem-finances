import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

// A year that has not started: nothing is posted at once, so the demo data stays as it was.
const year = new Date().getFullYear() + 1;
const leap = (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
const february = leap ? 29 : 28;

test("a monthly rule that starts on the 31st shows the clamped February date and can be deleted", async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("g");
  await page.keyboard.press("u");
  await expect(page).toHaveURL(/\/recurrences$/);
  await expect(page.getByRole("heading", { level: 1, name: "Recurring" })).toBeVisible();

  const description = `E2E rent ${Date.now()}`;
  await page.getByLabel("Amount").fill("800.00");
  await page.getByLabel("Description").fill(description);
  await page.getByLabel("Starts on").fill(`${year}-01-31`);

  const preview = page.locator("p", { has: page.getByText("Next dates:") });
  await expect(preview).toContainText(`Jan 31, ${year}`);
  await expect(preview).toContainText(`Feb ${february}, ${year}`);
  await expect(preview).toContainText(`Mar 31, ${year}`);
  await expect(preview).toContainText(`Apr 30, ${year}`);
  await expect(preview).not.toContainText("Mar 3,");

  await page.getByRole("button", { name: "Add rule" }).click();
  const row = page.getByRole("row").filter({ hasText: description });
  await expect(row).toBeVisible();
  await expect(row).toContainText("Every 1 month(s)");
  await expect(row).toContainText(`Jan 31, ${year}`);

  await row.getByRole("button", { name: "Delete rule", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(row).toBeHidden();
});

test("an installment purchase on a card lands on the right statements", async ({ page }) => {
  const name = `E2E Installments ${Date.now()}`;
  await page.goto("/cards");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Card name").fill(name);
  await page.getByLabel("Credit limit").fill("2000.00");
  await page.getByLabel("Closing day").fill("10");
  await page.getByLabel("Due day").fill("25");
  await page.getByRole("button", { name: "Add card" }).click();
  await expect(page.getByRole("listitem").filter({ hasText: name })).toBeVisible();

  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("e");
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("100.00");
  const account = dialog.getByRole("combobox", { name: "Account" });
  await account.fill(name);
  await account.press("ArrowDown");
  await account.press("Enter");
  await dialog.getByLabel("Date").fill(`${year}-01-31`);
  await dialog.getByLabel("Description").fill("E2E sofa");
  await dialog.getByLabel("Pay in installments").check();
  await dialog.getByLabel("Number of parts (2 to 60)").fill("3");

  // 100.00 in 3 is 33.34, 33.33, 33.33, and the 31st keeps month ends.
  await expect(dialog.getByText(`Part 1 of 3: Jan 31, ${year} · $33.34`)).toBeVisible();
  await expect(dialog.getByText(`Part 2 of 3: Feb ${february}, ${year} · $33.33`)).toBeVisible();
  await expect(dialog.getByText(`Part 3 of 3: Mar 31, ${year} · $33.33`)).toBeVisible();

  await dialog.getByRole("button", { name: "Add installments" }).click();
  await expect(dialog).toBeHidden();

  // Closing day 10: Jan 31 closes Feb 10, Feb 28 closes Mar 10, Mar 31 closes Apr 10.
  await page.goto("/cards");
  await page.getByRole("listitem").filter({ hasText: name }).getByRole("link", { name }).click();
  await expect(page.getByRole("heading", { level: 1, name })).toBeVisible();
  await expect(page.getByRole("link", { name: new RegExp(`Feb 10, ${year}`) })).toBeVisible();
  await expect(page.getByRole("link", { name: new RegExp(`Mar 10, ${year}`) })).toBeVisible();
  await expect(page.getByRole("link", { name: new RegExp(`Apr 10, ${year}`) })).toBeVisible();

  // Leave the demo data as it was: other specs count the accounts.
  await page.getByRole("button", { name: `Delete ${name}` }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page).toHaveURL(/\/cards$/);
});

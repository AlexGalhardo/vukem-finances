import { expect, test } from "@playwright/test";
import { blurActive } from "../login";

// Journey 4: a card whose closing day is the 31st, a purchase over the statement boundary and installments that
// keep month ends. The 31st clamps to the last day of shorter months.
test("a card closing on day 31 splits purchases at the boundary and installments keep month ends", async ({ page }) => {
  const name = `J4 card ${Date.now()}`;
  await page.goto("/cards");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Card name").fill(name);
  await page.getByLabel("Credit limit").fill("5000.00");
  await page.getByLabel("Closing day", { exact: true }).fill("31");
  await page.getByLabel("Due day", { exact: true }).fill("10");
  await page.getByRole("button", { name: "Add card" }).click();
  await expect(page.getByRole("listitem").filter({ hasText: name })).toBeVisible();

  const spend = async (date: string, description: string, installments?: string) => {
    await page.goto("/transactions");
    await page.getByRole("heading", { level: 1 }).waitFor();
    await blurActive(page);
    await page.keyboard.press("n");
    await page.keyboard.press("e");
    const dialog = page.getByRole("dialog", { name: "New expense" });
    await dialog.getByLabel("Amount").fill("90.00");
    const account = dialog.getByRole("combobox", { name: "Account" });
    await account.fill(name);
    await account.press("ArrowDown");
    await account.press("Enter");
    await dialog.getByLabel("Date").fill(date);
    await dialog.getByLabel("Description").fill(description);
    if (installments) {
      await dialog.getByLabel("Pay in installments").check();
      await dialog.getByLabel("Number of parts (2 to 60)").fill(installments);
      await dialog.getByRole("button", { name: "Add installments" }).click();
    } else {
      await dialog.getByLabel("Description").press("Enter");
    }
    await expect(dialog).toBeHidden();
  };

  // 31 January is on the closing day: it belongs to the statement that closes that same day. 1 February starts the next.
  await spend("2027-01-31", "J4 on the closing day");
  await spend("2027-02-01", "J4 day after");
  await spend("2027-01-31", "J4 sofa", "3");

  await page.goto("/cards");
  await page.getByRole("listitem").filter({ hasText: name }).getByRole("link", { name }).click();
  await expect(page.getByRole("heading", { level: 1, name })).toBeVisible();
  // Closings: Jan 31, Feb 28 (clamped), Mar 31.
  await expect(page.getByRole("link", { name: /Jan 31, 2027/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Feb 28, 2027/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Mar 31, 2027/ })).toBeVisible();
  await page.getByRole("link", { name: /Jan 31, 2027/ }).click();
  await expect(page.getByRole("row").filter({ hasText: "J4 on the closing day" })).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: "J4 day after" })).toHaveCount(0);

  await page.getByRole("button", { name: `Delete ${name}` }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page).toHaveURL(/\/cards$/);
});

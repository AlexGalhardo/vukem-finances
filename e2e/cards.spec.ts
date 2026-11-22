import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test("g k opens the cards page outside a field and does nothing inside one", async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("g");
  await page.keyboard.press("k");
  await expect(page).toHaveURL(/\/cards$/);
  await expect(page.getByRole("heading", { level: 1, name: "Cards" })).toBeVisible();

  await page.getByLabel("Card name").focus();
  await page.keyboard.type("g k");
  await expect(page.getByLabel("Card name")).toHaveValue("g k");
  await expect(page).toHaveURL(/\/cards$/);
});

test("a card takes a purchase after the closing day onto the next statement, and paying it makes it paid", async ({
  page,
}) => {
  const name = `E2E Visa ${Date.now()}`;
  await page.goto("/cards");
  await page.getByRole("heading", { level: 1 }).waitFor();

  await page.getByLabel("Card name").fill(name);
  await page.getByLabel("Credit limit").fill("2000.00");
  await page.getByLabel("Closing day", { exact: true }).fill("10");
  await page.getByLabel("Due day", { exact: true }).fill("25");
  await page.getByLabel("Last 4 digits").fill("4242");
  await page.getByRole("button", { name: "Add card" }).click();

  const tile = page.getByRole("listitem").filter({ hasText: name });
  await expect(tile).toBeVisible();
  await expect(tile.getByRole("progressbar")).toBeVisible();
  await expect(tile).toContainText("4242");

  // 11 January is the day after the closing day: the purchase belongs to the statement that closes on 10 February.
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("e");
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("300.00");
  const account = dialog.getByRole("combobox", { name: "Account" });
  await account.fill(name);
  await account.press("ArrowDown");
  await account.press("Enter");
  await dialog.getByLabel("Date").fill("2026-01-11");
  const note = `card buy ${Date.now()}`;
  await dialog.getByLabel("Description").fill(note);
  await dialog.getByLabel("Description").press("Enter");
  await expect(dialog).toBeHidden();

  await page.goto("/cards");
  await page.getByRole("listitem").filter({ hasText: name }).getByRole("link", { name: name }).click();
  await expect(page.getByRole("heading", { level: 1, name })).toBeVisible();

  const february = page.getByRole("link", { name: /Feb 10, 2026/ });
  await expect(february).toBeVisible();
  await february.click();
  await expect(page.getByRole("row").filter({ hasText: note })).toBeVisible();
  await expect(february).toContainText("Overdue");

  await page.getByRole("button", { name: "Pay statement" }).click();
  const pay = page.getByRole("dialog", { name: "Pay the statement" });
  await pay.getByRole("button", { name: "Pay", exact: true }).click();
  await expect(pay).toBeHidden();

  await expect(february).toContainText("Paid");
  await expect(page.getByText("Overdue", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("row").filter({ hasText: "Credit card payment 2026-02-10" })).toBeVisible();

  // Leave the demo data as it was: other specs count the accounts.
  await page.getByRole("button", { name: `Delete ${name}` }).click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page).toHaveURL(/\/cards$/);
  await expect(page.getByRole("link", { name })).toHaveCount(0);
});

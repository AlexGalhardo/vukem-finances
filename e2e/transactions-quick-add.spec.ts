import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test.beforeEach(async ({ page }) => {
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("n e outside a field opens the dialog on the amount, Enter saves and the row shows in the list", async ({
  page,
}) => {
  const note = `quick add ${Date.now()}`;
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("e");

  const dialog = page.getByRole("dialog", { name: "New expense" });
  await expect(dialog).toBeVisible();
  const amount = dialog.getByLabel("Amount");
  await expect(amount).toBeFocused();

  await amount.fill("12.34");
  await dialog.getByLabel("Description").fill(note);
  await dialog.getByLabel("Description").press("Enter");

  await expect(dialog).toBeHidden();
  const row = page.getByRole("row").filter({ hasText: note });
  await expect(row).toBeVisible();
  await expect(row).toContainText("12.34");
});

test("n e typed inside the description field does nothing", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("i");

  const dialog = page.getByRole("dialog", { name: "New income" });
  const description = dialog.getByLabel("Description");
  await description.focus();
  await page.keyboard.type("n e");
  await expect(description).toHaveValue("n e");
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect(page.getByRole("dialog", { name: "New income" })).toBeVisible();
});

test("Escape closes the dialog and returns focus to the button that opened it", async ({ page }) => {
  const open = page.getByRole("button", { name: "Add transaction" });
  await open.click();
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await expect(dialog.getByLabel("Amount")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(open).toBeFocused();
});

test("an invalid amount keeps the dialog open with the message", async ({ page }) => {
  await page.getByRole("button", { name: "Add transaction" }).click();
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("abc");
  await dialog.getByLabel("Amount").press("Enter");
  await expect(dialog.getByRole("alert")).toBeVisible();
  await expect(dialog).toBeVisible();
});

test("n t moves money between two accounts and the list marks it as a transfer", async ({ page }) => {
  const note = `move ${Date.now()}`;
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("t");

  const dialog = page.getByRole("dialog", { name: "New transfer" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByLabel("Amount")).toBeFocused();
  await dialog.getByLabel("Amount").fill("25.00");
  await dialog.getByLabel("Description").fill(note);
  await dialog.getByLabel("Description").press("Enter");

  await expect(dialog).toBeHidden();
  const rows = page.getByRole("row").filter({ hasText: note });
  await expect(rows).toHaveCount(2);
  await expect(rows.first()).toContainText("Transfer");
});

test("the same transfer to one account is refused", async ({ page }) => {
  await page.getByRole("button", { name: "New transfer" }).first().click();
  const dialog = page.getByRole("dialog", { name: "New transfer" });
  const to = dialog.getByRole("combobox", { name: "To account" });
  const from = await dialog.getByRole("combobox", { name: "From account" }).inputValue();
  await to.fill(from);
  await to.press("ArrowDown");
  await to.press("Enter");
  await dialog.getByLabel("Amount").fill("1.00");
  await dialog.getByLabel("Amount").press("Enter");
  await expect(dialog.getByRole("alert")).toContainText("two different accounts");
});

test("the accounts page shows balances, the net worth and the archive toggle", async ({ page }) => {
  await page.goto("/accounts");
  await expect(page.getByText("Net worth")).toBeVisible();
  const table = page.getByRole("table", { name: "Accounts and their balances" });
  await expect(table.getByRole("row")).toHaveCount(3);

  const savings = table.getByRole("row").filter({ has: page.getByRole("cell", { name: "Savings", exact: true }) });
  await expect(savings).toHaveCount(1);
  await savings.getByRole("button", { name: "Archive Savings" }).click();
  await expect(table.getByRole("button", { name: "Archive Savings" })).toHaveCount(0);
  await expect(savings).toHaveCount(0);

  await page.getByLabel("Show archived").check();
  await expect(savings).toContainText("Archived");
  await savings.getByRole("button", { name: "Unarchive Savings" }).click();
  await expect(savings).not.toContainText("Archived");
});

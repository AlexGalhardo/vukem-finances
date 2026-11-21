import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test("/ outside a field focuses the search field and does nothing inside one", async ({ page }) => {
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  const search = page.getByRole("searchbox", { name: "Search descriptions" });

  await blurActive(page);
  await expect(search).not.toBeFocused();
  await page.keyboard.press("/");
  await expect(search).toBeFocused();
  // The shortcut did not type its own key into the field it focused.
  await expect(search).toHaveValue("");

  // Inside a field, / is text: it stays in the field and the page does not move.
  await page.keyboard.type("50/50");
  await expect(search).toHaveValue("50/50");
  await expect(page).toHaveURL(/\/transactions$/);

  const description = page.getByLabel("Minimum amount");
  await description.focus();
  await page.keyboard.press("/");
  await expect(description).toBeFocused();
  await expect(description).toHaveValue("/");
});

test("/ on another page takes you to the transactions search", async ({ page }) => {
  await page.goto("/accounts");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("/");
  await expect(page).toHaveURL(/\/transactions/);
  await expect(page.getByRole("searchbox", { name: "Search descriptions" })).toBeFocused();
});

test("a tag typed in the new expense dialog is created, shown, and filters the list", async ({ page }) => {
  const note = `tagged ${Date.now()}`;
  const tag = `e2e-${Date.now()}`;
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("e");

  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("9.99");
  await dialog.getByLabel("Description").fill(note);
  const tags = dialog.getByLabel("Tags", { exact: true });
  await tags.fill(tag);
  await tags.press("Enter");
  // Enter added the tag and did not submit the form.
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("button", { name: `Remove tag ${tag}` })).toBeVisible();
  await dialog.getByRole("button", { name: "Add transaction" }).click();
  await expect(dialog).toBeHidden();

  const row = page.getByRole("row").filter({ hasText: note });
  await expect(row).toContainText(tag);

  // Filter by the tag, then by text: the URL is the state.
  await page.getByLabel("Tag", { exact: true }).selectOption({ label: tag });
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expect(page).toHaveURL(/tag=\d+/);
  await expect(page.getByRole("row").filter({ hasText: note })).toBeVisible();

  await page.goto(`/transactions?q=${encodeURIComponent(`${note.slice(0, 6)}%`)}`);
  await expect(page.getByRole("row").filter({ hasText: note })).toHaveCount(0);
  await page.goto(`/transactions?q=${encodeURIComponent(note.toUpperCase())}`);
  await expect(page.getByRole("row").filter({ hasText: note })).toBeVisible();
});

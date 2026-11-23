import { expect, test } from "@playwright/test";

// Journey 5: a rule that started months ago and was never run (the server was down). Backfill creates the
// occurrences already due, once, and posts them as transactions. The scheduled catch-up itself is proved by the
// Go tests run in scripts/qa/journeys-nonweb.sh (step 5): a browser cannot move the server clock.
test("a monthly rule that started in the past backfills what was due and stays idempotent", async ({ page }) => {
  const description = `J5 internet ${Date.now()}`;
  const start = new Date();
  start.setMonth(start.getMonth() - 3);
  const date = `${start.toISOString().slice(0, 7)}-05`;

  await page.goto("/recurrences");
  await page.getByRole("heading", { level: 1, name: "Recurring" }).waitFor();
  await page.getByLabel("Amount").fill("99.90");
  await page.getByLabel("Description").fill(description);
  await page.getByLabel("Starts on").fill(date);
  await page.getByLabel("Also create the occurrences already due (up to 36)").check();
  await page.getByRole("button", { name: "Add rule" }).click();
  await expect(page.getByRole("row").filter({ hasText: description }).first()).toBeVisible();

  await page.goto(`/transactions?q=${encodeURIComponent(description)}`);
  await page.getByRole("heading", { level: 1 }).waitFor();
  const rows = page.getByRole("row").filter({ hasText: description });
  await expect.poll(async () => rows.count(), { timeout: 15_000 }).toBeGreaterThanOrEqual(3);
  const first = await rows.count();

  await page.reload();
  await expect(page.getByRole("row").filter({ hasText: description })).toHaveCount(first);

  await page.goto("/recurrences");
  await page
    .getByRole("row")
    .filter({ hasText: description })
    .getByRole("button", { name: "Delete rule", exact: true })
    .first()
    .click();
  await page
    .getByRole("dialog", { name: "Delete this item?" })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page.getByRole("row").filter({ hasText: description })).toHaveCount(0);
});

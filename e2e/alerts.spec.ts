import { expect, test } from "@playwright/test";
import { blurActive } from "./login";

test("spending past a budget limit raises the bell count, and the list works by keyboard", async ({ page }) => {
  // An overall budget of one cent: any expense of the month reaches 100%.
  await page.goto("/budgets");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Monthly limit").fill("0.01");
  await page.getByRole("button", { name: "Save budget" }).click();
  await expect(page.getByRole("progressbar", { name: "Spent on Overall" })).toBeVisible();

  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await blurActive(page);
  await page.keyboard.press("n");
  await page.keyboard.press("e");
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await dialog.getByLabel("Amount").fill("5.00");
  const note = `alerts ${Date.now()}`;
  await dialog.getByLabel("Description").fill(note);
  await dialog.getByLabel("Description").press("Enter");
  await expect(dialog).toBeHidden();

  // The alert is made by a job that the write enqueues: give it a moment.
  const bell = page.getByRole("button", { name: /^Notifications, \d+ unread$/ });
  await expect(async () => {
    await page.reload();
    await expect(bell).toBeVisible({ timeout: 1000 });
  }).toPass({ timeout: 15_000 });

  // Open the list with the keyboard alone and mark everything as read.
  await bell.focus();
  await page.keyboard.press("Enter");
  const list = page.getByRole("dialog", { name: "Notifications" });
  await expect(list).toBeVisible();
  await expect(list.getByText(/overall budget is used up/i).first()).toBeVisible();

  const markAll = list.getByRole("button", { name: "Mark all as read" });
  await markAll.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("button", { name: "Notifications, none unread" })).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(list).toBeHidden();

  // The full page lists it as read, and the preferences are there.
  await page.goto("/notifications");
  await expect(page.getByRole("heading", { level: 1, name: "Notifications" })).toBeVisible();
  await expect(page.getByText("Read ·").first()).toBeVisible();
  await expect(page.getByRole("checkbox", { name: "A budget reaches 80%: Email" })).toBeVisible();
});

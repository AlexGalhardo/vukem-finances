import { expect, type Page, test } from "@playwright/test";
import { blurActive } from "./login";

const card = (page: Page) => page.locator('[data-slot="tutorial-step"]');

test.beforeEach(async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).waitFor();
});

test("the welcome card is offered once, never opens the tour by itself and stays dismissed after a reload", async ({
  page,
}) => {
  const welcome = page.getByRole("region", { name: "Welcome. Want a quick tour?" });
  await expect(welcome).toBeVisible();
  await expect(card(page)).toHaveCount(0);

  await welcome.getByRole("button", { name: "Not now" }).click();
  await expect(welcome).toHaveCount(0);
  await page.reload();
  await page.getByRole("heading", { level: 1 }).waitFor();
  await expect(page.getByRole("region", { name: "Welcome. Want a quick tour?" })).toHaveCount(0);
});

test("start, next, back and skip walk the steps and the screens, and focus follows the person", async ({ page }) => {
  await page.getByRole("button", { name: "Take the tour" }).click();
  await expect(card(page)).toBeVisible();
  await expect(card(page).getByText("Step 1 of 5")).toBeVisible();
  await expect(card(page).getByText("Why this screen exists")).toBeVisible();

  await card(page).getByRole("button", { name: "Next", exact: true }).click();
  await expect(card(page).getByText("Step 2 of 5")).toBeVisible();
  await expect(card(page).getByRole("heading", { level: 3 })).toBeFocused();

  await card(page).getByRole("button", { name: "Back" }).click();
  await expect(card(page).getByText("Step 1 of 5")).toBeVisible();

  await card(page).getByRole("button", { name: "Skip screen" }).click();
  await expect(page).toHaveURL(/\/transactions/);
  await expect(card(page).getByText("Step 1 of")).toBeVisible();
});

test("Escape pauses, Help offers Resume, and the position survives a reload", async ({ page }) => {
  await page.getByRole("button", { name: "Take the tour" }).click();
  await card(page).getByRole("button", { name: "Next", exact: true }).click();
  await expect(card(page).getByText("Step 2 of 5")).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(card(page)).toHaveCount(0);
  await expect(page.getByText("Tutorial paused. Resume it from Help.")).toBeVisible();

  await page.reload();
  await page.getByRole("heading", { level: 1 }).waitFor();
  await expect(card(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Help", exact: true }).click();
  await page.getByRole("menuitem", { name: /Resume tutorial/ }).click();
  await expect(card(page).getByText("Step 2 of 5")).toBeVisible();
});

test("the tutorial makes no write request", async ({ page }) => {
  const writes: string[] = [];
  page.on("request", (request) => {
    if (request.method() !== "GET" && request.method() !== "HEAD") writes.push(`${request.method()} ${request.url()}`);
  });
  await page.getByRole("button", { name: "Take the tour" }).click();
  await card(page).getByRole("button", { name: "Next", exact: true }).click();
  await card(page).getByRole("button", { name: "Skip screen" }).click();
  await expect(page).toHaveURL(/\/transactions/);
  await expect(card(page)).toBeVisible();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Help", exact: true }).click();
  await page.getByRole("menuitem", { name: /Resume tutorial/ }).click();
  await expect(card(page)).toBeVisible();
  expect(writes).toEqual([]);
});

test("h t starts the tutorial, and the center restarts it after a confirmation and hides billing", async ({ page }) => {
  await blurActive(page);
  await page.keyboard.press("h");
  await page.keyboard.press("t");
  await expect(card(page)).toBeVisible();
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Help", exact: true }).click();
  await page.getByRole("menuitem", { name: /Help for this page/ }).click();
  const sheet = page.getByRole("dialog", { name: "Overview" });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByText("Billing")).toHaveCount(0);

  await sheet.getByRole("button", { name: "Restart tutorial" }).click();
  const confirm = page.getByRole("dialog", { name: "Restart tutorial" });
  await expect(confirm.getByText("Restart from the beginning? Your data is not affected.")).toBeVisible();
  await confirm.getByRole("button", { name: "Restart tutorial" }).click();
  await expect(confirm).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(sheet).toHaveCount(0);
  await page.getByRole("button", { name: "Help", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: /Start tutorial/ })).toBeVisible();
});

test("a step is anchored to its target: the target is outlined and the card sits beside it", async ({ page }) => {
  await page.getByRole("button", { name: "Take the tour" }).click();
  const balance = page.locator('[data-tour="dashboard-balance"]');
  await expect(balance).toHaveAttribute("data-tour-active", "");
  const box = await balance.boundingBox();
  const cardBox = await card(page).boundingBox();
  expect(box).not.toBeNull();
  expect(cardBox).not.toBeNull();
  // Anchored below its target, not centered on the page.
  expect((cardBox?.y ?? 0) >= (box?.y ?? 0) + (box?.height ?? 0)).toBe(true);
  await card(page).getByRole("button", { name: "Next", exact: true }).click();
  await expect(balance).not.toHaveAttribute("data-tour-active", "");
  await expect(page.locator('[data-tour="dashboard-month"]')).toHaveAttribute("data-tour-active", "");
});

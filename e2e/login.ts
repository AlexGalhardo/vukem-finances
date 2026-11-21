import type { Page } from "@playwright/test";
import { E2E_ADMIN_EMAIL, E2E_ADMIN_PASSWORD } from "./env";

/** Signs in as the seeded administrator and waits for the dashboard. */
export async function login(page: Page): Promise<void> {
  await page.goto("/login");
  await page.getByLabel("Email").fill(E2E_ADMIN_EMAIL);
  await page.locator("input[name=password]").fill(E2E_ADMIN_PASSWORD);
  await page
    .getByRole("button", { name: /log in/i })
    .first()
    .click();
  await page.waitForURL(/\/dashboard/);
  await page.getByRole("heading", { level: 1 }).waitFor();
}

/** Puts focus on the page itself so a shortcut is not typed into a control. */
export async function blurActive(page: Page): Promise<void> {
  await page.evaluate(() => {
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
  });
}

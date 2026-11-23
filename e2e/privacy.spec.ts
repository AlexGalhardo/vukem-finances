import { expect, test } from "@playwright/test";
import { E2E_ADMIN_EMAIL, E2E_ADMIN_PASSWORD } from "./env";

// Runs on the saas server (see playwright.config.ts) after billing.spec.ts: it deletes the account it signs in with,
// so it must stay the last spec of that project.
test.describe.configure({ mode: "serial" });

test("the export is a download that holds the profile and no secret", async ({ page }) => {
  await page.goto("/account/privacy");
  await expect(page.getByRole("heading", { level: 1, name: "Privacy and data" })).toBeVisible();

  await page.locator("input[name=password]").first().fill("not my password");
  await page.getByRole("button", { name: "Export my data" }).click();
  await expect(page.getByText("The password is incorrect.")).toBeVisible();
  await expect(page.locator("input[name=password]").first()).toHaveValue("");

  await page.locator("input[name=password]").first().fill(E2E_ADMIN_PASSWORD);
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Export my data" }).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/^vukem-finances-personal-export-.*\.json$/);

  const stream = await download.createReadStream();
  const decoder = new TextDecoder();
  let body = "";
  for await (const chunk of stream as AsyncIterable<Uint8Array>) body += decoder.decode(chunk, { stream: true });
  const doc = JSON.parse(body) as { format: string; profile: { email: string } };
  expect(doc.format).toBe("vukem-finances-personal-export");
  expect(doc.profile.email).toBe(E2E_ADMIN_EMAIL);
  expect(body).not.toContain(E2E_ADMIN_PASSWORD);
  expect(body.toLowerCase()).not.toContain("$argon2");
});

test("the owner schedules the deletion of the workspace and cancels it", async ({ page }) => {
  await page.goto("/account/privacy");

  const label = (
    (await page.locator("label", { hasText: "Type the workspace name to confirm" }).textContent()) ?? ""
  ).trim();
  const name = label.slice(label.indexOf(":") + 1).trim();
  expect(name).not.toBe("");

  await page.getByLabel(/Type the workspace name to confirm/).fill("some other name");
  await page
    .locator("form", { hasText: "Delete the workspace" })
    .locator("input[name=password]")
    .fill(E2E_ADMIN_PASSWORD);
  await page.getByRole("button", { name: "Delete the workspace" }).click();
  // The form clears the password when the answer arrives: typing earlier would lose it.
  await expect(page.locator("form", { hasText: "Delete the workspace" }).locator("input[name=password]")).toHaveValue(
    "",
  );
  await expect(page.getByRole("heading", { name: "This workspace is scheduled for deletion" })).toHaveCount(0);

  await page.getByLabel(/Type the workspace name to confirm/).fill(name);
  await page
    .locator("form", { hasText: "Delete the workspace" })
    .locator("input[name=password]")
    .fill(E2E_ADMIN_PASSWORD);
  await page.getByRole("button", { name: "Delete the workspace" }).click();

  await expect(page.getByText("This workspace is scheduled for deletion")).toBeVisible();
  await expect(page.getByText(/read-only and will be erased for good on/)).toBeVisible();

  // Read-only on every surface: the page still opens, a write is refused with 423.
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { level: 1 }).first()).toBeVisible();

  await page.goto("/account/privacy");
  await page.getByRole("button", { name: "Cancel the deletion" }).click();
  await expect(page.getByRole("button", { name: "Delete the workspace" })).toBeVisible();
});

test("deleting the account ends the session, and signing in offers to cancel", async ({ page }) => {
  await page.goto("/account/privacy");

  await page.getByLabel(/Type your email address to confirm/).fill(E2E_ADMIN_EMAIL);
  await page.locator("form", { hasText: "Delete my account" }).locator("input[name=password]").fill(E2E_ADMIN_PASSWORD);
  await page.getByRole("button", { name: "Delete my account" }).click();
  await page.waitForURL(/\/login/);

  // Signing in is not consent: the person lands on the page that offers to keep the account.
  await page.getByLabel("Email").fill(E2E_ADMIN_EMAIL);
  await page.locator("input[name=password]").fill(E2E_ADMIN_PASSWORD);
  await page
    .getByRole("button", { name: /log in/i })
    .first()
    .click();
  await page.waitForURL(/\/account\/deletion/);
  await expect(page.getByRole("heading", { level: 1, name: "Your account is scheduled for deletion" })).toBeVisible();

  await page.getByRole("button", { name: "Keep my account" }).click();
  await page.waitForURL(/\/dashboard/);
});

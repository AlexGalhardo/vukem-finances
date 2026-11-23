import { expect, test } from "@playwright/test";

test.use({ actionTimeout: 8000 });

// Journey 3: the owner invites a viewer, the viewer registers from the link and can read but not write, and once the
// owner removes the member the session of that person stops working.
test("an invited viewer reads, cannot write, and loses access when removed", async ({ page, browser }) => {
  const email = `j3-${Date.now()}@example.test`;

  await page.goto("/workspace");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Role", { exact: true }).selectOption("viewer");
  await page.getByRole("button", { name: "Send invitation" }).click();
  const link = await page.getByLabel("Invitation link").inputValue();
  expect(link).toContain("/invites/");

  const guest = await browser.newContext({
    baseURL: test.info().project.use.baseURL,
    storageState: { cookies: [], origins: [] },
  });
  try {
    const viewer = await guest.newPage();
    await viewer.goto(new URL(link).pathname + new URL(link).search);
    await viewer.getByRole("heading", { level: 1 }).waitFor();
    await viewer.getByLabel("Name", { exact: true }).fill("J3 Viewer");
    await viewer.locator("input[name=password]").fill("Example-Local-Passw0rd-xyz");
    await viewer.locator("input[name=password_confirmation]").fill("Example-Local-Passw0rd-xyz");
    await viewer.getByRole("button", { name: "Create account" }).click();
    // Registering sends the person to the sign-in page: the account exists, the session does not yet.
    await viewer.waitForURL(/\/login/);
    await viewer.getByLabel("Email").fill(email);
    await viewer.locator("input[name=password]").fill("Example-Local-Passw0rd-xyz");
    await viewer
      .getByRole("button", { name: /log in/i })
      .first()
      .click();
    await viewer.waitForURL(/\/dashboard/);

    await viewer.goto("/transactions");
    await viewer.getByRole("heading", { level: 1 }).waitFor();
    await expect(viewer.getByRole("button", { name: "Add transaction" })).toHaveCount(0);
    await viewer.goto("/workspace");
    await expect(viewer.getByLabel("Email", { exact: true })).toHaveCount(0);

    await page.goto("/workspace");
    const member = page
      .getByRole("listitem")
      .filter({ hasText: email, has: page.getByRole("button", { name: "Remove" }) });
    await expect(member).toBeVisible();
    await expect(page.getByLabel("Role of J3 Viewer")).toHaveValue("viewer");
    await member.getByRole("button", { name: "Remove" }).click();
    await expect(member).toHaveCount(0);

    // The removed person no longer reaches the workspace: the next page is not the owner's data.
    const answer = await viewer.goto("/transactions");
    expect(answer?.status()).not.toBe(200);
    expect((await viewer.locator("body").textContent()) ?? "").not.toContain("Groceries");
  } finally {
    await guest.close();
  }
});

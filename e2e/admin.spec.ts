import { expect, test } from "@playwright/test";

const card = (page: import("@playwright/test").Page) => page.locator('[data-slot="tutorial-step"]');

test("the admin panel has its sections, a single h1 and a tutorial that runs on the panel screens", async ({
  page,
}) => {
  await page.goto("/admin");
  await expect(page.getByRole("heading", { level: 1, name: "Dashboard" })).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Admin sections" }).getByRole("link", { name: "Users" }),
  ).toBeVisible();

  await page.getByRole("button", { name: "Start tutorial" }).click();
  await expect(card(page)).toBeVisible();
  await expect(card(page).getByText("Step 1 of 2")).toBeVisible();
  await expect(page.locator('[data-tour="admin-overview"][data-tour-active]')).toHaveCount(1);

  await card(page).getByRole("button", { name: "Next", exact: true }).click();
  await expect(card(page).getByText("Step 2 of 2")).toBeVisible();
  await expect(page.locator('[data-tour="admin-nav"][data-tour-active]')).toHaveCount(1);
});

test("a resource list is a real table with sortable headers and opens a record", async ({ page }) => {
  await page.goto("/admin/users");
  await expect(page.getByRole("columnheader", { name: "Email", exact: true })).toBeVisible();
  await page.getByRole("columnheader", { name: "Name", exact: true }).getByRole("button").click();
  await expect(page.getByRole("columnheader", { name: "Name", exact: true })).toHaveAttribute(
    "aria-sort",
    /ascending|descending/,
  );

  await page.getByRole("row").nth(1).getByRole("link").first().click();
  await expect(page).toHaveURL(/\/admin\/users\/\d+/);
  await expect(page.getByText("admin@example.test")).toBeVisible();
});

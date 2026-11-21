import { expect, test } from "@playwright/test";

test("a token is shown once, works on the API with its scopes only, and stops when revoked", async ({
  page,
  request,
}) => {
  await page.goto("/tokens");
  await expect(page.getByRole("heading", { level: 1, name: "API tokens" })).toBeVisible();

  await page.getByLabel("Name").fill("e2e token");
  await page.getByLabel("accounts:read").check();
  await page.getByRole("button", { name: "Create token" }).click();

  const secret = page.getByTestId("token-secret");
  await expect(secret).toBeVisible();
  const token = (await secret.textContent()) ?? "";
  expect(token).toMatch(/^vf_[A-Z2-7]+\.[A-Z2-7]+$/);

  // The scope it holds works; the one it lacks is a 403, and a cookie-less call needs the token.
  const auth = { Authorization: `Bearer ${token}` };
  const me = await request.get("/api/v1/me", { headers: auth });
  expect(me.status()).toBe(200);
  expect((await me.json()).token_name).toBe("e2e token");
  expect((await request.get("/api/v1/accounts", { headers: auth })).status()).toBe(200);
  expect((await request.get("/api/v1/transactions", { headers: auth })).status()).toBe(403);

  // Dismissing hides it, and a reload never brings it back.
  await page.getByRole("button", { name: "I copied it" }).click();
  await expect(page.getByTestId("token-secret")).toHaveCount(0);
  await page.reload();
  expect(await page.content()).not.toContain(token);
  await expect(page.getByRole("cell", { name: "e2e token" })).toBeVisible();

  // Revoking is immediate.
  await page.getByRole("button", { name: "Revoke" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await expect(page.getByRole("cell", { name: "Revoked" })).toBeVisible();
  expect((await request.get("/api/v1/me", { headers: auth })).status()).toBe(401);
});

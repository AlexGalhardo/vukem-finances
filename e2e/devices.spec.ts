import { expect, test } from "@playwright/test";

test("a device registered through the sync API is listed, and revoking it stops it at once", async ({
  page,
  request,
}) => {
  // A token with the sync scope, made on the Tokens page.
  await page.goto("/tokens");
  await page.getByLabel("Name").fill("e2e sync token");
  await page.getByLabel("sync", { exact: true }).check();
  await page.getByRole("button", { name: "Create token" }).click();
  const token = (await page.getByTestId("token-secret").textContent()) ?? "";
  const auth = { Authorization: `Bearer ${token}`, "X-Sync-Protocol": "1" };

  const made = await request.post("/api/v1/sync/devices", {
    headers: auth,
    data: { name: "e2e phone", platform: "android" },
  });
  expect(made.status()).toBe(201);
  const deviceId = (await made.json()).device_id as string;

  const sync = { ...auth, "X-Device-Id": deviceId };
  expect((await request.get("/api/v1/sync/pull?cursor=0&limit=1", { headers: sync })).status()).toBe(200);

  await page.goto("/devices");
  await expect(page.getByRole("heading", { level: 1, name: "Devices" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "e2e phone" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Syncing" })).toBeVisible();
  await expect(page.getByText("keeps its local data")).toBeVisible();

  await page.getByRole("button", { name: "Revoke" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await expect(page.getByRole("cell", { name: "Revoked" })).toBeVisible();

  const refused = await request.get("/api/v1/sync/pull?cursor=0&limit=1", { headers: sync });
  expect(refused.status()).toBe(403);
  expect((await refused.json()).code).toBe("device_revoked");
});

import { createHmac } from "node:crypto";
import { readFileSync } from "node:fs";
import { type APIRequestContext, expect, test } from "@playwright/test";
import { E2E_FAKE_PORT, E2E_SAAS_PORT, PIX_WEBHOOK_SECRET } from "./env";

// The saas server of this spec talks to a fake AbacatePay (e2e/fake-abacatepay.mjs) and to nothing else: no real
// provider is called and every credential here is a made-up placeholder.
const control = `http://127.0.0.1:${E2E_FAKE_PORT}/__control`;
// The public key AbacatePay documents for the body signature: read from the code that checks it, so they cannot drift.
const publicKey = /pixPublicKey = "([^"]+)"/.exec(readFileSync("app/billing/abacatepay.go", "utf8"))?.[1] ?? "";

type Charge = { id: string; amount: number; status: string };

async function lastCharge(): Promise<Charge> {
  const list = (await (await fetch(`${control}/charges`)).json()) as Charge[];
  return list[list.length - 1];
}

async function setStatus(id: string, status: string): Promise<void> {
  await fetch(`${control}/status?id=${id}&status=${status}`, { method: "POST" });
}

/** A signed delivery like AbacatePay's: the shared secret in a header and the HMAC of the raw body. */
async function deliver(request: APIRequestContext, event: string, id: string) {
  const body = JSON.stringify({ id: "log_e2e", event, apiVersion: 2, devMode: true, data: { transparent: { id } } });
  return request.post(`http://localhost:${E2E_SAAS_PORT}/abacatepay/webhook`, {
    headers: {
      "Content-Type": "application/json",
      "X-Webhook-Secret": PIX_WEBHOOK_SECRET,
      "X-Webhook-Signature": createHmac("sha256", publicKey).update(body).digest("base64"),
    },
    data: body,
  });
}

test.describe.configure({ mode: "serial" });

test("a PIX payment turns the workspace into Pro and a refund takes it away", async ({ page, request, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/billing");
  await expect(page.getByRole("heading", { level: 1, name: "Billing" })).toBeVisible();
  await expect(page.getByTestId("plan-state")).toHaveText("This workspace is on the Free plan.");

  // The total comes from the server's price table: 3 months of R$ 19.90.
  await page.getByLabel("Period").selectOption("3");
  await expect(page.getByText(/Total: .*59[.,]70/)).toBeVisible();
  await page.getByRole("button", { name: "Pay with PIX" }).click();

  const dialog = page.getByRole("dialog", { name: "Pay with PIX" });
  await expect(dialog.getByRole("img", { name: "PIX QR code" })).toBeVisible();
  await expect(dialog.getByText(/^00020101021226fake/)).toBeVisible();
  await dialog.getByRole("button", { name: "Copy code" }).click();
  await expect(dialog.getByText("Code copied")).toBeVisible();
  await expect(dialog.getByRole("status").filter({ hasText: "Waiting for the payment" })).toBeVisible();

  const charge = await lastCharge();
  expect(charge.amount).toBe(3 * 1990);

  // The provider says paid: the next poll announces it in the live region.
  await setStatus(charge.id, "PAID");
  await expect(dialog.getByRole("status").filter({ hasText: "Payment received" })).toBeVisible({ timeout: 20_000 });
  await dialog.getByRole("button", { name: "Close" }).click();
  await expect(page.getByTestId("plan-state")).toContainText("Pro is paid with PIX until");

  // A refund event: the server re-reads the charge, revokes it, and the page says why.
  await setStatus(charge.id, "REFUNDED");
  expect((await deliver(request, "transparent.refunded", charge.id)).status()).toBe(200);
  await page.reload();
  await expect(page.getByTestId("plan-state")).toHaveText("Pro access ended: the payment was refunded.");
});

test("a forged paid event for an unpaid charge activates nothing", async ({ page, request }) => {
  await page.goto("/billing");
  await page.getByRole("button", { name: "Pay with PIX" }).click();
  const dialog = page.getByRole("dialog", { name: "Pay with PIX" });
  await expect(dialog.getByRole("img", { name: "PIX QR code" })).toBeVisible();

  const charge = await lastCharge();
  expect(charge.status).toBe("PENDING");
  expect((await deliver(request, "transparent.completed", charge.id)).status()).toBe(200);

  await page.waitForTimeout(4000); // longer than one poll
  await expect(dialog.getByRole("status").filter({ hasText: "Waiting for the payment" })).toBeVisible();
  await dialog.getByRole("button", { name: "Close" }).click();
  await page.reload();
  await expect(page.getByTestId("plan-state")).not.toContainText("Pro is paid with PIX");
});

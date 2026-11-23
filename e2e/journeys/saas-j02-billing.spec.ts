import { expect, test } from "@playwright/test";
import { E2E_FAKE_PORT, E2E_SAAS_PORT, PIX_WEBHOOK_SECRET } from "../env";
import { signPixBody } from "../pix-signature.mjs";

// Journey 2 (saas edition, fake AbacatePay only): the Free plan, an upgrade by PIX, a refund that revokes it and the
// downgrade the page shows. Stripe test mode is not driven from the browser: its checkout, webhook and revoke paths
// run against the fake Stripe in the Go tests listed in scripts/qa/journeys-nonweb.sh (step 2).
const control = `http://127.0.0.1:${E2E_FAKE_PORT}/__control`;
type Charge = { id: string; amount: number; status: string };

async function lastCharge(): Promise<Charge> {
  const list = (await (await fetch(`${control}/charges`)).json()) as Charge[];
  return list[list.length - 1];
}

async function setStatus(id: string, status: string): Promise<void> {
  await fetch(`${control}/status?id=${id}&status=${status}`, { method: "POST" });
}

test("Free limits, a PIX upgrade, a refund that revokes and the downgrade", async ({ page, request }) => {
  await page.goto("/accounts");
  await expect(page.getByRole("heading", { level: 1, name: "Accounts" })).toBeVisible();

  await page.goto("/billing");
  await expect(page.getByTestId("plan-state")).not.toContainText("Pro is paid with PIX until");
  await page.getByRole("button", { name: "Pay with PIX" }).click();
  const dialog = page.getByRole("dialog", { name: "Pay with PIX" });
  await expect(dialog.getByRole("img", { name: "PIX QR code" })).toBeVisible();

  const charge = await lastCharge();
  await setStatus(charge.id, "PAID");
  await expect(dialog.getByRole("status").filter({ hasText: "Payment received" })).toBeVisible({ timeout: 20_000 });
  await dialog.getByRole("button", { name: "Close" }).click();
  await expect(page.getByTestId("plan-state")).toContainText("Pro is paid with PIX until");

  await setStatus(charge.id, "REFUNDED");
  const body = JSON.stringify({
    id: "log_j2",
    event: "transparent.refunded",
    apiVersion: 2,
    devMode: true,
    data: { transparent: { id: charge.id } },
  });
  const delivered = await request.post(`http://localhost:${E2E_SAAS_PORT}/abacatepay/webhook`, {
    headers: {
      "Content-Type": "application/json",
      "X-Webhook-Secret": PIX_WEBHOOK_SECRET,
      "X-Webhook-Signature": signPixBody(body),
    },
    data: body,
  });
  expect(delivered.status()).toBe(200);
  await page.reload();
  await expect(page.getByTestId("plan-state")).toHaveText("Pro access ended: the payment was refunded.");
});

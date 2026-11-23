import { expect, test } from "@playwright/test";
import { E2E_TELEGRAM_PORT } from "../env";

// Journey 7: link a chat, record an expense from it, undo it, unlink. The Bot API is the fake in e2e/fake-telegram.mjs;
// the token is a placeholder and nothing reaches Telegram.
const control = `http://127.0.0.1:${E2E_TELEGRAM_PORT}/__control`;
const CHAT = 515151;

type Sent = { chat: number; text: string; buttons: { text: string; data: string }[] };

async function say(text: string): Promise<void> {
  await fetch(`${control}/push`, { method: "POST", body: JSON.stringify({ chat: CHAT, text }) });
}

async function tap(data: string): Promise<void> {
  await fetch(`${control}/tap`, { method: "POST", body: JSON.stringify({ chat: CHAT, data }) });
}

async function sent(): Promise<Sent[]> {
  const all = (await (await fetch(`${control}/sent`)).json()) as Sent[];
  return all.filter((m) => m.chat === CHAT);
}

test("link, record by chat, undo and unlink", async ({ page }) => {
  const description = `j7 lunch ${Date.now()}`;

  await page.goto("/settings/telegram");
  await page.getByRole("button", { name: "Connect a chat" }).click();
  const code = ((await page.getByTestId("telegram-code").textContent()) ?? "").replace("/start ", "");
  await say(`/start ${code}`);
  await expect.poll(async () => (await sent()).some((m) => m.text.includes("Connected"))).toBe(true);

  const before = (await sent()).length;
  await say(`-12,50 ${description}`);
  await expect.poll(async () => (await sent()).length).toBeGreaterThan(before);
  const draft = (await sent()).at(-1);
  expect(draft?.text).toContain("12.50");
  const none = draft?.buttons.find((b) => b.text === "No category");
  expect(none).toBeDefined();

  // Nothing is saved until the tap.
  await page.goto(`/transactions?q=${encodeURIComponent(description)}`);
  await page.getByRole("heading", { level: 1 }).waitFor();
  await expect(page.getByRole("row").filter({ hasText: description })).toHaveCount(0);

  await tap(none?.data ?? "");
  await expect.poll(async () => (await sent()).some((m) => m.text.includes("Saved"))).toBe(true);
  await page.reload();
  await expect(page.getByRole("row").filter({ hasText: description })).toBeVisible();

  const saved = (await sent()).filter((m) => m.text.includes("Saved")).at(-1);
  const undo = saved?.buttons.find((b) => b.data.startsWith("eu:"));
  expect(undo).toBeDefined();
  await tap(undo?.data ?? "");
  await expect
    .poll(async () => {
      await page.reload();
      return page.getByRole("row").filter({ hasText: description }).count();
    })
    .toBe(0);

  await say("/unlink");
  await expect.poll(async () => (await sent()).some((m) => m.text.startsWith("Disconnected"))).toBe(true);
  await page.goto("/settings/telegram");
  await expect(page.getByText("No chat is linked.")).toBeVisible();
});

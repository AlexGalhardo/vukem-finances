import { expect, type Page, test } from "@playwright/test";
import { E2E_TELEGRAM_PORT } from "./env";

// The personal server runs the Telegram bot in polling mode against e2e/fake-telegram.mjs: no real Bot API is
// called and the token is a made-up placeholder.
const control = `http://127.0.0.1:${E2E_TELEGRAM_PORT}/__control`;
const CHAT = 424242;

async function push(chat: number, text: string): Promise<void> {
  await fetch(`${control}/push`, { method: "POST", body: JSON.stringify({ chat, text }) });
}

async function sentTo(chat: number): Promise<string[]> {
  const all = (await (await fetch(`${control}/sent`)).json()) as { chat: number; text: string }[];
  return all.filter((m) => m.chat === chat).map((m) => m.text);
}

async function connect(page: Page): Promise<string> {
  await page.getByRole("button", { name: "Connect a chat" }).click();
  const code = page.getByTestId("telegram-code");
  await expect(code).toBeVisible();
  return ((await code.textContent()) ?? "").replace("/start ", "");
}

test.describe.configure({ mode: "serial" });

test("the code is shown once, counts down and expires", async ({ page }) => {
  await page.clock.install();
  await page.goto("/settings/telegram");
  await expect(page.getByRole("heading", { level: 1, name: "Telegram" })).toBeVisible();
  await expect(page.getByText("not end-to-end encrypted")).toBeVisible();
  await expect(page.getByText("No chat is linked.")).toBeVisible();

  const code = await connect(page);
  expect(code).toMatch(/^[A-Za-z0-9_-]{22}$/);
  await expect(page.getByTestId("telegram-link")).toHaveAttribute("href", `https://t.me/e2e_finance_bot?start=${code}`);
  await expect(page.getByTestId("telegram-countdown")).toContainText(/Expires in (10:00|9:5\d)/);

  // Time passes in the page: the countdown runs down and the code is declared expired.
  await page.clock.runFor("05:00");
  await expect(page.getByTestId("telegram-countdown")).toContainText(/Expires in [45]:\d\d/);
  await page.clock.runFor("05:05");
  await expect(page.getByTestId("telegram-expired")).toBeVisible();
  await expect(page.getByTestId("telegram-link")).toHaveCount(0);

  // A reload never shows a code again, and the page offers no field for a chat id.
  await page.reload();
  await expect(page.getByTestId("telegram-code")).toHaveCount(0);
  expect(await page.content()).not.toContain(code);
  await expect(page.getByLabel(/chat id/i)).toHaveCount(0);
});

test("a chat links with the code, the alert switch works and unlink cuts the chat off", async ({ page }) => {
  await page.goto("/settings/telegram");
  const code = await connect(page);

  // Telegram delivers /start <code> from a private chat; the app polls the fake and links it.
  await push(CHAT, `/start ${code}`);
  await expect.poll(async () => (await sentTo(CHAT)).some((t) => t.includes("Connected"))).toBe(true);

  await page.reload();
  await expect(page.getByText("Chat ending in 4242")).toBeVisible();
  expect(await page.content()).not.toContain(String(CHAT));

  // The same code does nothing a second time.
  await push(CHAT + 1, `/start ${code}`);
  await expect
    .poll(async () => (await sentTo(CHAT + 1)).some((t) => t.includes("invalid, expired or already used")))
    .toBe(true);

  // The alert switch sticks across a reload.
  const alerts = page.getByLabel("Send my alerts of this workspace to my linked chats");
  await expect(alerts).not.toBeChecked();
  // A reload before the answer would cancel the save.
  const saved = page.waitForResponse(
    (r) => r.request().method() === "PUT" && new URL(r.url()).pathname === "/settings/telegram/alerts",
  );
  await alerts.check();
  await saved;
  await page.reload();
  await expect(alerts).toBeChecked();
  await alerts.uncheck();

  // Unlink from the web: the chat is cut off at once.
  await page.getByRole("button", { name: "Disconnect" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await expect(page.getByText("No chat is linked.")).toBeVisible();

  await push(CHAT, "/balance");
  await expect.poll(async () => (await sentTo(CHAT)).some((t) => t.includes("not connected"))).toBe(true);
});

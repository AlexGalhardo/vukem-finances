import { expect, test } from "@playwright/test";
import { E2E_PORT } from "./env";

// D-027: the personal edition is offline-first. The browser may talk to the server it came from and to nothing else:
// every other origin is blocked here and any attempt fails the test, even when the page copes with the failure.
const ORIGIN = `http://localhost:${E2E_PORT}`;

// Same origin, or a URL that never leaves the browser.
function isLocal(url: string): boolean {
  if (url.startsWith("data:") || url.startsWith("blob:") || url.startsWith("about:")) return true;
  return new URL(url).origin === ORIGIN;
}

const PAGES = [
  "/dashboard",
  "/accounts",
  "/transactions",
  "/categories",
  "/budgets",
  "/goals",
  "/cards",
  "/recurrences",
  "/reports",
  "/import",
  "/notifications",
  "/workspace",
  "/ai",
  "/settings/telegram",
  "/tokens",
  "/account/privacy",
];

test("the main pages load nothing from another origin", async ({ page, context }) => {
  const attempts: string[] = [];

  await context.route("**/*", (route) => {
    const url = route.request().url();
    if (isLocal(url)) return route.continue();
    attempts.push(`${route.request().method()} ${url}`);
    return route.abort("blockedbyclient");
  });
  page.on("request", (request) => {
    if (!isLocal(request.url())) attempts.push(`${request.method()} ${request.url()} (seen)`);
  });
  page.on("websocket", (socket) => {
    if (!isLocal(socket.url().replace(/^ws/, "http"))) attempts.push(`websocket ${socket.url()}`);
  });

  for (const path of PAGES) {
    const response = await page.goto(path);
    expect(response?.status(), path).toBeLessThan(500);
    await page.getByRole("heading", { level: 1 }).first().waitFor();
    // Let late requests (fonts, images, deferred scripts) show up before the next page.
    await page.waitForLoadState("networkidle");
  }

  // The quick add dialog and the command palette are the busiest client code.
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).first().waitFor();
  await page.keyboard.press("Control+k");
  await page.keyboard.press("Escape");
  await page.waitForLoadState("networkidle");

  expect(attempts, "requests to another origin").toEqual([]);
});

test("the sign-in and setup screens load nothing from another origin", async ({ browser }) => {
  // A fresh context has no session, so these are the pages a first visit sees.
  const context = await browser.newContext({ baseURL: ORIGIN, storageState: { cookies: [], origins: [] } });
  const attempts: string[] = [];

  await context.route("**/*", (route) => {
    const url = route.request().url();
    if (isLocal(url)) return route.continue();
    attempts.push(`${route.request().method()} ${url}`);
    return route.abort("blockedbyclient");
  });

  const page = await context.newPage();

  for (const path of ["/login", "/forgot-password"]) {
    await page.goto(path);
    await page.getByRole("heading", { level: 1 }).first().waitFor();
    await page.waitForLoadState("networkidle");
  }

  await context.close();
  expect(attempts, "requests to another origin").toEqual([]);
});

test("the root redirects inside the app and never to another site", async ({ browser }) => {
  // The personal edition has no landing page: a guest goes to /login on the same origin, a signed-in person to the dashboard.
  const guest = await browser.newContext({ baseURL: ORIGIN, storageState: { cookies: [], origins: [] } });
  const attempts: string[] = [];
  await guest.route("**/*", (route) => {
    const url = route.request().url();
    if (isLocal(url)) return route.continue();
    attempts.push(url);
    return route.abort("blockedbyclient");
  });
  const page = await guest.newPage();
  await page.goto("/");
  await page.getByRole("heading", { level: 1 }).first().waitFor();
  expect(new URL(page.url()).origin).toBe(ORIGIN);
  expect(new URL(page.url()).pathname).toBe("/login");
  await guest.close();
  expect(attempts, "requests to another origin").toEqual([]);
});

test("the root sends a signed-in person to the dashboard", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(`${ORIGIN}/dashboard`);
});

test("the guard sees a request to another origin", async ({ page, context }) => {
  const attempts: string[] = [];
  await context.route("**/*", (route) => {
    const url = route.request().url();
    if (isLocal(url)) return route.continue();
    attempts.push(url);
    return route.abort("blockedbyclient");
  });
  // The app's own pages are under a CSP that refuses other origins before the network is asked, so probe from a blank page.
  await page.goto("about:blank");
  await page.evaluate(() => fetch("https://example.invalid/ping").catch(() => undefined));
  expect(attempts).toEqual(["https://example.invalid/ping"]);
});

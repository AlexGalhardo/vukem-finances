import { expect, test } from "@playwright/test";
import { builtPages, ensureBuilt, OUT, serve, servePlanted } from "./site-server.mjs";

// D-019 and D-020: the built site (landing and docs) is plain static files. Every page is opened from a plain static
// file server and every request is recorded: all must be GET, same origin and a static file, none to a path that
// looks like an API or an account flow. The only dynamic-looking traffic allowed is Pagefind reading its own index
// under /pagefind/.
const API_LIKE = /^\/(?:[a-z]{2}(?:-[a-z]{2})?\/)?(?:api|graphql|login|register|checkout)(?:\/|$|\?)/i;

/** The problems of a list of recorded requests. */
function audit(origin: string, requests: { method: string; url: string }[]): string[] {
  const problems: string[] = [];
  for (const r of requests) {
    if (/^(data|blob|about):/.test(r.url)) continue;
    const u = new URL(r.url);
    if (r.method !== "GET") problems.push(`${r.method} ${r.url}`);
    if (u.origin !== origin) problems.push(`other origin: ${r.url}`);
    if (API_LIKE.test(u.pathname + u.search)) problems.push(`API-like path: ${r.url}`);
  }
  return problems;
}

test.beforeAll(() => ensureBuilt());

test("every page of the built site makes only GET requests for same-origin static files", async ({ page }) => {
  test.setTimeout(240_000);
  const site = await serve(OUT);
  const requests: { method: string; url: string }[] = [];
  const sockets: string[] = [];
  page.on("request", (r) => requests.push({ method: r.method(), url: r.url() }));
  page.on("websocket", (s) => sockets.push(s.url()));

  const pages = builtPages();
  expect(pages.length).toBeGreaterThan(70);
  for (const p of pages) {
    const response = await page.goto(site.origin + p);
    expect(response?.status(), p).toBe(200);
    await page.waitForLoadState("networkidle");
  }
  await site.close();

  expect(sockets).toEqual([]);
  expect(audit(site.origin, requests)).toEqual([]);
  expect(requests.some((r) => r.url.includes("/pagefind/"))).toBe(false);
});

test("the docs search reads only /pagefind/ and finds a page", async ({ page }) => {
  const site = await serve(OUT);
  const requests: { method: string; url: string }[] = [];
  page.on("request", (r) => requests.push({ method: r.method(), url: r.url() }));

  await page.goto(`${site.origin}/docs/`);
  const input = page.getByRole("searchbox", { name: "Search the documentation" });
  await expect(input).toBeVisible();
  await input.fill("docker");
  const results = page.locator("[data-search-results] a");
  await expect(results.first()).toBeVisible();
  await expect(page.getByRole("status")).toContainText("results for");
  const first = await results.first().getAttribute("href");
  expect(first).toMatch(/^\/docs\//);
  await site.close();

  const pagefind = requests.filter((r) => new URL(r.url).pathname.startsWith("/pagefind/"));
  expect(pagefind.length).toBeGreaterThan(0);
  expect(audit(site.origin, requests)).toEqual([]);
});

test("the audit sees a planted request to an API path and to another origin", async ({ page }) => {
  const site = await servePlanted(
    `<!doctype html><title>x</title><script>fetch("/api/v1/me").catch(()=>{});fetch("/login",{method:"POST"}).catch(()=>{})</script>`,
  );
  const requests: { method: string; url: string }[] = [];
  page.on("request", (r) => requests.push({ method: r.method(), url: r.url() }));
  await page.goto(`${site.origin}/`);
  await page.waitForLoadState("networkidle");
  await site.close();

  const problems = audit(site.origin, requests).join("\n");
  expect(problems).toContain("API-like path");
  expect(problems).toContain("POST");
});

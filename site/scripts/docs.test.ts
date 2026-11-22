import { beforeAll, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { checkBuilt } from "../lib/built.ts";
import { checkContent } from "../lib/content-check.ts";
import { type ClaimContext, type ClaimPage, checkClaims, shellCommands } from "../lib/doc-claims.ts";
import { buildContext, toClaimPages } from "../lib/doc-context.ts";
import { buildNavigation, type DocPage, loadPages, orderedSlugs, pagePath, twinPath } from "../lib/docs.ts";
import { nextEntries, staleEntries } from "../lib/last-updated.ts";
import { llmsTxt, twinMarkdown } from "../lib/twins.ts";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const origin = "https://finances.vukem.dev";
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "vf-docs-"));
const pages = loadPages();

const page = (body: string, slug = "run/x"): ClaimPage => ({
  slug,
  file: `site/content/docs/${slug}.md`,
  body,
  bodyLine: 5,
});

const ctx: ClaimContext = {
  cliCommands: new Set(["accounts list", "accounts show"]),
  serverCommands: new Set(["mcp:serve"]),
  isEnvVar: (name) => ["APP_KEY", "APP_EDITION"].includes(name),
  pathExists: (rel) => ["compose.yaml", "docs/api.md"].includes(rel),
  restOperations: [
    { method: "GET", path: "/accounts" },
    { method: "PATCH", path: "/accounts/{uid}" },
  ],
  mcpTools: new Set(["accounts_list"]),
  anchors: new Map([
    ["run/x", new Set(["intro"])],
    ["run/y", new Set(["setup"])],
  ]),
};
const claims = (body: string): string[] => checkClaims([page(body), page("## Setup", "run/y")], ctx);

describe("the claims check, with a planted failure for each kind of claim", () => {
  test("passes a page whose claims all exist", () => {
    const good = [
      "## Intro",
      "",
      "```sh",
      "app accounts list --output json",
      "app mcp:serve --stdio",
      "docker compose -f compose.yaml up -d",
      'curl -H "Authorization: Bearer vf_..." "$BASE/api/v1/accounts/<uid>?x=1"',
      "```",
      "",
      "Set `APP_KEY` in `docs/api.md`; call `GET /api/v1/accounts` or `accounts_list`. See [the guide](/docs/run/y/#setup) and [here](#intro).",
    ].join("\n");
    expect(claims(good)).toEqual([]);
  });

  test("fails a CLI command that does not exist, naming the file and line", () => {
    const problems = claims("## Intro\n\n```sh\napp accounts frobnicate\n```");
    expect(problems).toEqual([
      'site/content/docs/run/x.md:8: "app accounts frobnicate" is not a command of the CLI reference',
    ]);
    expect(claims("```sh\napp nothing here\n```").join()).toContain('"app nothing" is not a command');
    expect(claims("```sh\napp auth explode\n```").join()).toContain("app auth explode");
  });

  test("joins a command continued with a backslash before checking it", () => {
    expect(
      shellCommands([
        { text: "app accounts \\", line: 1 },
        { text: "  list", line: 2 },
      ]),
    ).toEqual([{ text: "app accounts list", line: 1 }]);
  });

  test("fails an environment variable that does not exist", () => {
    expect(claims("Use `NOT_A_REAL_SETTING` here.").join()).toContain(
      "NOT_A_REAL_SETTING is not a setting of the product",
    );
    expect(claims("```sh\nFOO_BAR=1 app accounts list\n```").join()).toContain("FOO_BAR");
  });

  test("fails a REST path or route that is not in the OpenAPI document", () => {
    expect(claims("Call `/api/v1/ghosts`.").join()).toContain("/api/v1/ghosts is not in the OpenAPI document");
    expect(claims("```sh\ncurl $BASE/api/v1/accounts/<uid>/vanish\n```").join()).toContain(
      "/api/v1/accounts/<uid>/vanish",
    );
    expect(claims("Call `DELETE /api/v1/accounts/{uid}`.").join()).toContain(
      "DELETE /api/v1/accounts/<uid>".replace("<uid>", "{uid}"),
    );
  });

  test("fails an MCP tool that does not exist", () => {
    expect(claims("The tool `accounts_vanish` removes it.").join()).toContain(
      "the MCP tool accounts_vanish is not in the MCP reference",
    );
    expect(claims("Plain words like `last_3_months` are not tools.")).toEqual([]);
  });

  test("fails a repository path or compose file that does not exist", () => {
    expect(claims("Read `docs/ghost.md`.").join()).toContain("the path docs/ghost.md does not exist");
    expect(claims("```sh\ndocker compose -f compose.ghost.yaml up\n```").join()).toContain(
      "compose.ghost.yaml does not exist",
    );
    expect(claims("See [x](https://github.com/AlexGalhardo/vukem-finances/blob/main/docs/ghost.md).").join()).toContain(
      "docs/ghost.md",
    );
  });

  test("fails a broken link, a broken anchor, an outside link and a bare URL", () => {
    expect(claims("[a](/docs/run/ghost/)").join()).toContain("points at no page");
    expect(claims("[a](/docs/run/y/#nope)").join()).toContain("points at no heading");
    expect(claims("[a](#nope)").join()).toContain("not a heading of this page");
    expect(claims("[a](/docs/run/y)").join()).toContain("must end with a slash");
    expect(claims("[a](https://example.com/x)").join()).toContain("only links to the repository");
    expect(claims("[a](../y.md)").join()).toContain("absolute /docs/ path");
    expect(claims("Visit https://example.com now.").join()).toContain("bare URL");
    expect(claims("Visit `https://example.com` now.")).toEqual([]);
  });
});

describe("the documentation pages", () => {
  test("every claim of every real page is true", async () => {
    expect(checkClaims(toClaimPages(pages), await buildContext(pages))).toEqual([]);
  }, 60000);

  test("every page passes the content check", async () => {
    expect(await checkContent(pages)).toEqual([]);
  }, 60000);

  test("every page is in the navigation once, and the order is stable", () => {
    const order = orderedSlugs(buildNavigation(pages));
    expect([...order].sort()).toEqual(pages.map((p) => p.slug).sort());
    expect(order[0]).toBe("index");
  });
});

describe("the content check, with planted bad pages", () => {
  const good: DocPage = {
    slug: "start/a",
    title: "A page",
    description: "Does a thing.",
    generated: false,
    file: "a.md",
    body: "## Intro\n\nText.\n",
    bodyLine: 5,
  };
  const nav = [{ title: "Start", items: ["start/a"] }];
  const problems = (change: Partial<DocPage>, sections = nav): Promise<string[]> =>
    checkContent([{ ...good, ...change }], sections);

  test("accepts a good page", async () => {
    expect(await problems({})).toEqual([]);
  });

  test("fails a title that makes the document title longer than 60 characters", async () => {
    expect((await problems({ title: "x".repeat(44) })).join()).toContain("title is 44 characters");
  });

  test("fails a missing description, an h1, a skipped level, raw HTML and a fence with no language", async () => {
    expect((await problems({ description: "" })).join()).toContain("needs a description");
    expect((await problems({ body: "# Title\n" })).join()).toContain("start headings at h2");
    expect((await problems({ body: "## A\n\n#### B\n" })).join()).toContain("jumps from h2 to h4");
    expect((await problems({ body: "## A\n\n<div>x</div>\n" })).join()).toContain("raw HTML");
    expect((await problems({ body: "## A\n\n```\ncode\n```\n" })).join()).toContain("needs a language");
  });

  test("fails a page that is not in the navigation and a navigation entry with no page", async () => {
    expect((await problems({}, [{ title: "Start", items: [] }])).join()).toContain("not in content/navigation.ts");
    expect((await problems({}, [{ title: "Start", items: ["start/a", "start/ghost"] }])).join()).toContain(
      "start/ghost",
    );
    expect((await problems({}, [{ title: "Start", items: ["start/a", "start/a"] }])).join()).toContain("listed twice");
  });

  test("fails a generated page without its banner", async () => {
    expect((await problems({ generated: true })).join()).toContain("generated banner");
  });
});

describe("the Markdown twins and llms.txt", () => {
  test("a twin rewrites internal links to the twins and keeps the anchors", () => {
    const twin = twinMarkdown({ ...pages[0], body: "[x](/docs/run/docker/#start) and [y](/docs/)" }, origin);
    expect(twin).toContain("[x](https://finances.vukem.dev/docs/run/docker.md#start)");
    expect(twin).toContain("[y](https://finances.vukem.dev/docs/index.md)");
    expect(twin.startsWith(`# ${pages[0].title}`)).toBe(true);
  });

  test("llms.txt lists every page with its summary", () => {
    const text = llmsTxt(pages, origin, orderedSlugs(buildNavigation(pages)));
    for (const p of pages) expect(text).toContain(`(${origin}${twinPath(p.slug)}): ${p.description}`);
  });
});

describe("the last-updated dates", () => {
  const a: DocPage = { slug: "a", title: "A", description: "d", generated: false, file: "a.md", body: "", bodyLine: 1 };
  const hashes: Record<string, string> = { a: "h1" };
  const hashOf = (p: DocPage): string => hashes[p.slug];

  test("a changed page gets a new date and an unchanged one keeps its own", () => {
    const first = nextEntries([a], {}, hashOf, () => "2026-01-01");
    expect(first).toEqual({ a: { date: "2026-01-01", hash: "h1" } });
    expect(nextEntries([a], first, hashOf, () => "2026-02-02")).toEqual(first);
    hashes.a = "h2";
    expect(nextEntries([a], first, hashOf, () => "2026-02-02")).toEqual({ a: { date: "2026-02-02", hash: "h2" } });
  });

  test("the check fails when a page changed after its date, is missing or is gone", () => {
    hashes.a = "h1";
    const entries = nextEntries([a], {}, hashOf, () => "2026-01-01");
    expect(staleEntries([a], entries, hashOf)).toEqual([]);
    hashes.a = "h3";
    expect(staleEntries([a], entries, hashOf).join()).toContain("changed after its date 2026-01-01");
    expect(staleEntries([a], {}, hashOf).join()).toContain("no date");
    expect(staleEntries([], entries, hashOf).join()).toContain("not a page any more");
  });
});

describe("the built documentation", () => {
  const out = path.join(tmp, "out");
  const docs = (dir: string, slug: string): string => path.join(dir, "docs", slug, "index.html");

  beforeAll(async () => {
    const env = { ...process.env, SITE_OUT: out, SITE_URL: origin } as Record<string, string>;
    for (const cmd of [
      ["bun", "--bun", "astro", "build"],
      ["bun", "scripts/sitemap.ts"],
      ["bun", "scripts/twins.ts"],
      ["bun", "scripts/search.ts"],
    ]) {
      const proc = Bun.spawn(cmd, { cwd: root, env, stdout: "pipe", stderr: "pipe" });
      if ((await proc.exited) !== 0)
        throw new Error(`${cmd.join(" ")} failed: ${await new Response(proc.stderr).text()}`);
    }
  }, 300000);

  const options = { origin, termsApproved: false };
  const planted = (slug: string, change: (html: string) => string): string => {
    const dir = fs.mkdtempSync(path.join(tmp, "planted-"));
    fs.cpSync(out, dir, {
      recursive: true,
      // The checks only look for the Pagefind library; its many index fragments would slow every copy.
      filter: (source) => !/[/\\]pagefind[/\\](fragment|index)([/\\]|$)/.test(source),
    });
    fs.writeFileSync(docs(dir, slug), change(fs.readFileSync(docs(dir, slug), "utf8")));
    return checkBuilt(dir, options).join("\n");
  };

  test("the build is clean", () => {
    expect(checkBuilt(out, options)).toEqual([]);
  });

  test("every page is built, has a Markdown twin at the same path with .md and is in llms.txt", () => {
    const llms = fs.readFileSync(path.join(out, "llms.txt"), "utf8");
    expect(pages.length).toBeGreaterThan(60);
    for (const p of pages) {
      expect(fs.existsSync(path.join(out, pagePath(p.slug), "index.html")), p.slug).toBe(true);
      expect(fs.existsSync(path.join(out, twinPath(p.slug))), p.slug).toBe(true);
      expect(llms).toContain(`${origin}${twinPath(p.slug)})`);
    }
    expect(fs.readFileSync(path.join(out, "sitemap.xml"), "utf8")).toContain(`${origin}/docs/run/docker/`);
  });

  test("a page with no twin, or missing from llms.txt, fails the check", () => {
    const dir = fs.mkdtempSync(path.join(tmp, "twin-"));
    fs.cpSync(out, dir, { recursive: true });
    fs.rmSync(path.join(dir, "docs", "run", "docker.md"));
    expect(checkBuilt(dir, options).join("\n")).toContain("docs/run/docker/index.html: no Markdown twin");
    fs.writeFileSync(path.join(dir, "llms.txt"), "# empty\n");
    expect(checkBuilt(dir, options).join("\n")).toContain("not listed in llms.txt");
  });

  test("every page has one h1, a lang, a short title, a description, a canonical link and one main", () => {
    for (const p of pages) {
      const html = fs.readFileSync(docs(out, p.slug === "index" ? "" : p.slug), "utf8");
      expect(html.match(/<h1[\s>]/g)?.length, p.slug).toBe(1);
      expect(html).toMatch(/<html lang="en"/);
      expect(html.match(/<title>([^<]*)<\/title>/)?.[1].length, p.slug).toBeLessThanOrEqual(60);
      expect(html).toMatch(/<meta name="description" content="[^"]+"/);
      expect(html).toMatch(/<link rel="canonical" href="https:\/\/finances\.vukem\.dev\/docs/);
      expect(html.match(/<main[\s>]/g)?.length, p.slug).toBe(1);
    }
  });

  test("the code blocks and tables are focusable regions with an accessible name", () => {
    const html = fs.readFileSync(docs(out, "integrate/rest-api"), "utf8");
    expect(html).toMatch(/<pre[^>]*tabindex="0"[^>]*role="region"[^>]*aria-label="Code block/);
    expect(html).toMatch(/<div class="table-scroll" tabindex="0" role="region" aria-label="Table: /);
  });

  test("the pages use no form or other origin and work without JavaScript; the search is the only script", () => {
    for (const slug of ["run/docker", "integrate/cli", "operate/environment"]) {
      const html = fs.readFileSync(docs(out, slug), "utf8");
      expect(html).not.toMatch(/<form/i);
      // The one script only shows the search box (it is hidden in the HTML); the content needs no JavaScript.
      expect(html.match(/<script[\s>]/gi)).toHaveLength(1);
      expect(html).toMatch(/<script type="module">/);
      expect(html).toMatch(/data-docs-search[^>]*\shidden/);
    }
  });

  test("a planted page with two h1 elements, a skipped level or an image without alt fails", () => {
    expect(planted("run/docker", (h) => h.replace("</main>", "<h1>Another</h1></main>"))).toContain(
      "needs exactly one h1, found 2",
    );
    expect(planted("run/docker", (h) => h.replace("</main>", "<h4>Deep</h4></main>"))).toContain("heading level jumps");
    expect(
      planted("run/docker", (h) => h.replace("</main>", '<img src="/favicon.svg" width="1" height="1"></main>')),
    ).toContain("image without alt text");
    expect(
      planted("run/docker", (h) => h.replace(/<title>[^<]*<\/title>/, `<title>${"t".repeat(70)}</title>`)),
    ).toContain("title is 70 characters");
    expect(planted("run/docker", (h) => h.replace(/<link rel="canonical"[^>]*>/, ""))).toContain(
      "has no canonical link",
    );
  });

  test("a planted form, external script or external link in a docs page fails the no-backend check", () => {
    expect(planted("run/docker", (h) => h.replace("</main>", "<form></form></main>"))).toContain("contains a <form");
    expect(
      planted("run/docker", (h) => h.replace("</head>", '<script src="https://cdn.example/x.js"></script></head>')),
    ).toContain("another origin");
    expect(planted("run/docker", (h) => h.replace("</main>", '<a href="https://example.com/">x</a></main>'))).toContain(
      "links to https://example.com/",
    );
  });
});

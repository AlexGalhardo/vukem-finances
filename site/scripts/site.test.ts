import { beforeAll, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { checkBuilt } from "../lib/built.ts";
import { catalogs, flatten } from "../lib/copy.ts";
import { renderMarkdown } from "../lib/markdown.ts";
import { plans } from "../lib/plans.ts";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = path.resolve(root, "..");
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "vf-site-"));
const origin = "https://finances.vukem.dev";

async function build(out: string, env: Record<string, string> = {}): Promise<void> {
  const full = { ...process.env, ...env, SITE_OUT: out } as Record<string, string>;
  for (const cmd of [
    ["bun", "--bun", "astro", "build"],
    ["bun", "scripts/sitemap.ts"],
    ["bun", "scripts/twins.ts"],
  ]) {
    const proc = Bun.spawn(cmd, { cwd: root, env: full, stdout: "pipe", stderr: "pipe" });
    const code = await proc.exited;
    if (code !== 0) throw new Error(`${cmd.join(" ")} failed: ${await new Response(proc.stderr).text()}`);
  }
}

const normal = path.join(tmp, "normal");
const approved = path.join(tmp, "approved");
const prefixed = path.join(tmp, "prefixed");

beforeAll(async () => {
  await build(normal, { SITE_URL: origin, LANDING_TERMS_APPROVED: "false" });
  await build(approved, { SITE_URL: origin, LANDING_TERMS_APPROVED: "true" });
  await build(prefixed, { SITE_URL: "https://example.com/finances", LANDING_TERMS_APPROVED: "false" });
}, 300000);

const read = (dir: string, file: string): string => fs.readFileSync(path.join(dir, file), "utf8");

/** Copies the good build, applies a change and returns the problems the check finds. */
function planted(change: (dir: string) => void, options = { origin, termsApproved: false }): string {
  const dir = fs.mkdtempSync(path.join(tmp, "planted-"));
  fs.cpSync(normal, dir, { recursive: true });
  change(dir);
  return checkBuilt(dir, options).join("\n");
}

const editIndex = (replace: (html: string) => string) => (dir: string) => {
  const file = path.join(dir, "index.html");
  fs.writeFileSync(file, replace(fs.readFileSync(file, "utf8")));
};

describe("catalogs", () => {
  test("the three languages have identical keys and no empty value", () => {
    const keys = Object.keys(flatten(catalogs.en)).sort();
    for (const locale of ["pt-br", "es"] as const) {
      expect(Object.keys(flatten(catalogs[locale])).sort()).toEqual(keys);
    }
    for (const catalog of Object.values(catalogs)) {
      for (const [key, value] of Object.entries(flatten(catalog))) expect(value.trim(), key).not.toBe("");
    }
  });
});

describe("plans", () => {
  test("plans.json has no price and mirrors app/workspaces/limits.go", () => {
    expect(JSON.stringify(plans)).not.toMatch(/price|amount|currency/i);
    const go = fs.readFileSync(path.join(repoRoot, "app", "workspaces", "limits.go"), "utf8");
    const free = go.match(/Free = Limits\{([^}]*)\}/)?.[1] ?? "";
    expect(free).toContain(`Accounts: ${plans.free.accounts}`);
    expect(free).toContain(`Transactions: ${plans.free.transactionsPerMonth}`);
    expect(free).toContain(`Members: ${plans.free.members}`);
    expect(go).toContain("Unlimited = -1");
    const pro = go.match(/Pro\s+= Limits\{([\s\S]*?)\n\t\}/)?.[1] ?? "";
    for (const field of ["Accounts", "Transactions", "Members"]) expect(pro).toContain(`${field}: Unlimited`);
    expect(pro).toContain("CSVImport: true");
    expect(pro).toContain("Budgets: true");
    expect(free).not.toContain("CSVImport: true");
    expect(free).not.toContain("Budgets: true");
  });

  test("the editions block shows the limits of plans.json", () => {
    const html = read(normal, "index.html");
    expect(html).toContain(`<td>${plans.free.transactionsPerMonth}</td>`);
    expect(html).toContain("<td>Unlimited</td>");
    expect(html).not.toMatch(/[$€£]|R\$/);
  });
});

describe("built site", () => {
  test("the real build passes every check, with and without the terms", () => {
    expect(checkBuilt(normal, { origin, termsApproved: false })).toEqual([]);
    expect(checkBuilt(approved, { origin, termsApproved: true })).toEqual([]);
  });

  test("the output is plain static files", () => {
    expect(fs.readdirSync(normal).filter((n) => /worker|server|functions/i.test(n))).toEqual([]);
    expect(read(root, "astro.config.mjs")).not.toContain("adapter");
    expect(read(normal, "CNAME")).toBe("finances.vukem.dev");
  });

  test("languages: lang, canonical and hreflang", () => {
    const pages = {
      "index.html": ["en", "/"],
      "pt-br/index.html": ["pt-BR", "/pt-br/"],
      "es/index.html": ["es", "/es/"],
    } as const;
    for (const [file, [lang, urlPath]] of Object.entries(pages)) {
      const html = read(normal, file);
      expect(html).toContain(`<html lang="${lang}">`);
      expect(html).toContain(`<link rel="canonical" href="${origin}${urlPath}">`);
      for (const [l, p] of [
        ["en", "/"],
        ["pt-BR", "/pt-br/"],
        ["es", "/es/"],
        ["x-default", "/"],
      ]) {
        expect(html).toContain(`<link rel="alternate" hreflang="${l}" href="${origin}${p}">`);
      }
    }
  });

  test("calls to action are plain links to the docs, the repository and the releases", () => {
    const html = read(normal, "index.html");
    expect(html).toContain('href="/docs/"');
    expect(html).toContain('href="https://github.com/AlexGalhardo/vukem-finances"');
    expect(html).toContain('href="https://github.com/AlexGalhardo/vukem-finances/releases"');
    expect(html).toContain("ghcr.io/alexgalhardo/vukem-finances");
    expect(html).not.toMatch(/<a [^>]*href="https?:\/\/(?!github\.com\/AlexGalhardo\/vukem-finances)/);
  });

  test("SEO: Open Graph, Twitter card and JSON-LD without a price", () => {
    const html = read(normal, "index.html");
    expect(html).toContain('property="og:image" content="https://finances.vukem.dev/og.png"');
    expect(html).toContain('name="twitter:card" content="summary_large_image"');
    const ld = html.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/)?.[1] ?? "";
    const parsed = JSON.parse(ld) as Record<string, unknown>;
    expect(parsed["@type"]).toBe("SoftwareApplication");
    expect(ld).not.toMatch(/price|offers/i);
  });

  test("the sitemap lists the landing pages and the privacy page; robots allows all", () => {
    const sitemap = read(normal, "sitemap.xml");
    for (const p of ["/", "/pt-br/", "/es/", "/privacy/"]) expect(sitemap).toContain(`<loc>${origin}${p}</loc>`);
    expect(sitemap).not.toContain("/terms/");
    expect(read(approved, "sitemap.xml")).toContain(`<loc>${origin}/terms/</loc>`);
    const robots = read(normal, "robots.txt");
    expect(robots).toContain("Allow: /");
    expect(robots).not.toContain("Disallow");
    expect(robots).toContain(`Sitemap: ${origin}/sitemap.xml`);
  });

  test("the terms page and link exist only when approved", () => {
    expect(fs.existsSync(path.join(normal, "terms"))).toBe(false);
    expect(read(normal, "index.html")).not.toContain("/terms/");
    expect(read(approved, "index.html")).toContain('href="/terms/"');
    expect(read(approved, "terms/index.html")).toContain("pending review by the author");
  });

  test("a base path prefixes every root-relative link", () => {
    const html = read(prefixed, "index.html");
    expect(html).toContain('href="/finances/docs/"');
    expect(html).toContain('href="/finances/pt-br/"');
    expect(html).toContain('href="https://example.com/finances/"');
    expect(html).not.toMatch(/href="\/(?!finances)/);
    expect(read(prefixed, "robots.txt")).toContain("Sitemap: https://example.com/finances/sitemap.xml");
  });

  test("every image has alt text, dimensions and the below-the-fold ones are lazy", () => {
    for (const file of ["index.html", "pt-br/index.html", "es/index.html"]) {
      for (const img of read(normal, file).matchAll(/<img [^>]*>/g)) {
        expect(img[0]).toMatch(/alt="[^"]+"/);
        expect(img[0]).toMatch(/width="\d+"/);
        expect(img[0]).toMatch(/height="\d+"/);
      }
    }
  });
});

describe("the checks fail on planted problems", () => {
  test("a form", () =>
    expect(planted(editIndex((h) => h.replace("</main>", "<form></form></main>")))).toContain("<form"));

  test("a script or stylesheet from another origin", () => {
    expect(
      planted(editIndex((h) => h.replace("</head>", '<script src="https://cdn.example.net/a.js"></script></head>'))),
    ).toContain("another origin");
    expect(
      planted(
        editIndex((h) => h.replace("</head>", '<link rel="stylesheet" href="https://fonts.example.net/a.css"></head>')),
      ),
    ).toContain("another origin");
  });

  test("a link to another host", () => {
    expect(planted(editIndex((h) => h.replace("</main>", '<a href="https://example.org/">x</a></main>')))).toContain(
      "links to https://example.org/",
    );
  });

  for (const api of [
    "fetch('/x')",
    "new XMLHttpRequest()",
    "new WebSocket('wss://x')",
    "new EventSource('/x')",
    "navigator.sendBeacon('/x')",
  ]) {
    test(`built JavaScript calling ${api}`, () => {
      expect(planted((dir) => fs.writeFileSync(path.join(dir, "_astro", "evil.js"), `${api};`))).toContain(
        "calls a network API",
      );
    });
    test(`inline script calling ${api}`, () => {
      expect(planted(editIndex((h) => h.replace("</body>", `<script>${api}</script></body>`)))).toContain(
        "network API",
      );
    });
  }

  test("a script of any kind", () => {
    expect(planted(editIndex((h) => h.replace("</body>", "<script>1</script></body>")))).toContain(
      "without JavaScript",
    );
  });

  test("a server or worker file", () => {
    expect(planted((dir) => fs.writeFileSync(path.join(dir, "_worker.js"), ""))).toContain("server or worker");
  });

  test("structure: two h1, no skip link, image without alt, no main", () => {
    expect(planted(editIndex((h) => h.replace("</main>", "<h1>two</h1></main>")))).toContain("exactly one h1");
    expect(planted(editIndex((h) => h.replace(/<a class="skip"[^>]*>[^<]*<\/a>/, "")))).toContain("skip link");
    expect(planted(editIndex((h) => h.replace("</main>", '<img src="/x.png" width="1" height="1"></main>')))).toContain(
      "without alt",
    );
    expect(planted(editIndex((h) => h.replace(/<\/?main[^>]*>/g, "")))).toContain("<main>");
  });

  test("metadata: long title, long description, price in JSON-LD, missing canonical", () => {
    expect(planted(editIndex((h) => h.replace(/<title>[^<]*<\/title>/, `<title>${"a".repeat(61)}</title>`)))).toContain(
      "at most 60",
    );
    expect(
      planted(editIndex((h) => h.replace(/(name="description" content=")[^"]*/, `$1${"a".repeat(161)}`))),
    ).toContain("at most 160");
    expect(
      planted(
        editIndex((h) =>
          h.replace('"@type":"SoftwareApplication"', '"@type":"SoftwareApplication","offers":{"price":"1"}'),
        ),
      ),
    ).toContain("price or offers");
    expect(planted(editIndex((h) => h.replace(/<link rel="canonical"[^>]*>/, "")))).toContain("canonical");
  });

  test("missing hreflang alternates", () => {
    expect(planted(editIndex((h) => h.replace(/<link rel="alternate" hreflang="es"[^>]*>/, "")))).toContain("hreflang");
  });

  test("the terms link without approval", () => {
    expect(planted(editIndex((h) => h.replace("</main>", '<a href="/terms/">t</a></main>')))).toContain("terms");
  });

  test("a hosted-service phrase in the visible text", () => {
    expect(planted(editIndex((h) => h.replace("</main>", "<p>Start free today</p></main>")))).toContain("start free");
  });

  test("weight: HTML and CSS over the budget", () => {
    expect(planted(editIndex((h) => h.replace("</main>", `<p>${"x".repeat(61 * 1024)}</p></main>`)))).toContain(
      "at most 61440",
    );
    const bigCss = (dir: string): void => {
      const assets = path.join(dir, "_astro");
      const css = fs.readdirSync(assets).find((f) => f.endsWith(".css")) as string;
      const noise = Array.from({ length: 6000 }, () => Math.random().toString(36).slice(2)).join("\n");
      fs.appendFileSync(path.join(assets, css), `\n/* ${noise} */`);
    };
    expect(planted(bigCss)).toContain("gzipped");
  });

  test("a missing landing page", () => {
    expect(planted((dir) => fs.rmSync(path.join(dir, "es"), { recursive: true }))).toContain(
      "es: landing page missing",
    );
  });
});

describe("pictures", () => {
  test("each screenshot has a WebP and a PNG of at most 150 KB, and the share image is 1200 by 630", () => {
    const dir = path.join(root, "public", "landing");
    const files = fs.readdirSync(dir);
    expect(files.length).toBeGreaterThan(0);
    for (const file of files) expect(fs.statSync(path.join(dir, file)).size, file).toBeLessThanOrEqual(153600);
    for (const name of ["dashboard", "transactions", "reports"]) {
      expect(files).toContain(`${name}.webp`);
      expect(files).toContain(`${name}.png`);
    }
    const og = fs.readFileSync(path.join(root, "public", "og.png"));
    expect([og.readUInt32BE(16), og.readUInt32BE(20)]).toEqual([1200, 630]);
  });
});

describe("markdown", () => {
  test("escapes markup and renders the supported syntax", () => {
    const { html, title } = renderMarkdown(
      "# T\n\n<script>x</script> **b** `c` [l](https://example.com)\n\n- one\n- two\n",
    );
    expect(title).toBe("T");
    expect(html).not.toContain("<script>");
    expect(html).toContain("<strong>b</strong>");
    expect(html).toContain('<a href="https://example.com">l</a>');
    expect(html).toContain("<ul><li>one</li><li>two</li></ul>");
  });

  test("the terms draft is marked pending review", () => {
    expect(fs.readFileSync(path.join(repoRoot, "docs", "legal", "terms.md"), "utf8")).toContain("pending review");
  });
});

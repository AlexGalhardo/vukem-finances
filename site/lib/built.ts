import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { findForbidden } from "./claims.ts";
import { catalogs } from "./copy.ts";
import { locales, repoUrl } from "./site.ts";

export interface BuiltOptions {
  /** Public origin the pages were built for; links and metadata to it are not "another origin". */
  origin: string;
  /** Whether the terms link must be present (true) or absent (false) in the footers. */
  termsApproved: boolean;
}

export const budgets = { htmlBytes: 60 * 1024, cssGzipBytes: 40 * 1024, totalGzipBytes: 150 * 1024 };

const SERVER_NAMES = /worker|server|functions|_routes\.json/i;
const NETWORK_API = /\bfetch\s*\(|XMLHttpRequest|WebSocket|EventSource|sendBeacon/;

function walk(dir: string): string[] {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    return entry.isDirectory() ? walk(full) : [full];
  });
}

const gzipSize = (data: Buffer | string): number => zlib.gzipSync(data).length;

function attr(tag: string, name: string): string | undefined {
  const match = tag.match(new RegExp(`\\s${name}\\s*=\\s*("([^"]*)"|'([^']*)'|([^\\s>]+))`, "i"));
  return match ? (match[2] ?? match[3] ?? match[4]) : undefined;
}

function tags(html: string, name: string): string[] {
  return [...html.matchAll(new RegExp(`<${name}(?=[\\s>/])[^>]*>`, "gi"))].map((m) => m[0]);
}

function stripTags(html: string): string {
  return html
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<[^>]+>/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&#39;|&apos;/g, "'")
    .replace(/&quot;/g, '"')
    .replace(/\s+/g, " ");
}

function isOtherOrigin(url: string, origin: string): boolean {
  if (!/^(https?:)?\/\//i.test(url)) return false;
  return new URL(url, origin).origin !== new URL(origin).origin;
}

/** Problems of one built page: the no-backend scan, the structure, the metadata and the weight. */
export function checkPage(dir: string, file: string, options: BuiltOptions): string[] {
  const problems: string[] = [];
  const rel = path.relative(dir, file).split(path.sep).join("/");
  const html = fs.readFileSync(file, "utf8");
  const fail = (message: string): number => problems.push(`${rel}: ${message}`);

  // No backend.
  if (/<form[\s>]/i.test(html)) fail("contains a <form");
  for (const tag of [
    ...tags(html, "script"),
    ...tags(html, "link"),
    ...tags(html, "img"),
    ...tags(html, "iframe"),
    ...tags(html, "source"),
  ]) {
    const rels = (attr(tag, "rel") ?? "").toLowerCase();
    if (/\b(canonical|alternate)\b/.test(rels)) continue;
    const url = attr(tag, "src") ?? attr(tag, "href");
    if (url && isOtherOrigin(url, options.origin)) fail(`loads ${url} from another origin`);
  }
  for (const script of html.matchAll(/<script(?=[\s>])([^>]*)>([\s\S]*?)<\/script>/gi)) {
    const type = attr(`<script${script[1]}>`, "type");
    if (type === "application/ld+json") continue;
    fail("contains a script, but content must work without JavaScript");
  }
  if (/\son[a-z]+\s*=/i.test(html.replace(/<script[\s\S]*?<\/script>/gi, ""))) fail("contains an inline event handler");
  const inlineJs = [...html.matchAll(/<script(?=[\s>])[^>]*>([\s\S]*?)<\/script>/gi)].map((m) => m[1]).join("\n");
  if (NETWORK_API.test(inlineJs)) fail("calls a network API in an inline script");

  // Only plain anchors lead outside, and only to the repository.
  for (const a of tags(html, "a")) {
    const href = attr(a, "href") ?? "";
    if (isOtherOrigin(href, options.origin) && !href.startsWith(repoUrl)) fail(`links to ${href}`);
  }

  // Structure.
  if (tags(html, "h1").length !== 1) fail(`needs exactly one h1, found ${tags(html, "h1").length}`);
  for (const landmark of ["header", "main", "footer", "nav"]) {
    if (tags(html, landmark).length === 0) fail(`has no <${landmark}>`);
  }
  if (tags(html, "main").length !== 1) fail("needs exactly one <main>");
  let level = 0;
  for (const heading of html.matchAll(/<h([1-6])(?=[\s>])/gi)) {
    const next = Number(heading[1]);
    if (level > 0 && next > level + 1) fail(`heading level jumps from h${level} to h${next}`);
    level = next;
  }
  const lang = attr(tags(html, "html")[0] ?? "", "lang");
  if (!lang) fail("<html> has no lang");
  const bodyAnchors = tags(html.slice(html.search(/<body/i)), "a");
  const skip = bodyAnchors[0] ? attr(bodyAnchors[0], "href") : undefined;
  if (!skip?.startsWith("#") || !new RegExp(`\\sid=["']${skip.slice(1)}["']`).test(html)) {
    fail("the first link of the page must be a skip link to an existing id");
  }
  for (const img of tags(html, "img")) {
    if (!(attr(img, "alt") ?? "").trim()) fail(`image without alt text: ${attr(img, "src")}`);
    if (!attr(img, "width") || !attr(img, "height")) fail(`image without dimensions: ${attr(img, "src")}`);
  }

  // Metadata.
  const title = html.match(/<title>([\s\S]*?)<\/title>/i)?.[1]?.trim() ?? "";
  if (!title) fail("has no title");
  if ([...title].length > 60) fail(`title is ${[...title].length} characters, at most 60`);
  const description = attr(tags(html, "meta").find((m) => attr(m, "name") === "description") ?? "", "content") ?? "";
  if (!description) fail("has no meta description");
  if ([...description].length > 160) fail(`description is ${[...description].length} characters, at most 160`);
  if (!tags(html, "link").some((l) => attr(l, "rel") === "canonical")) fail("has no canonical link");
  for (const ld of html.matchAll(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/gi)) {
    try {
      const parsed: unknown = JSON.parse(ld[1]);
      if (/"(price|offers|priceCurrency)"/i.test(JSON.stringify(parsed))) fail("JSON-LD carries a price or offers");
    } catch {
      fail("JSON-LD does not parse");
    }
  }

  // Terms link gating.
  const hasTerms = tags(html, "a").some((a) => /\/terms\/?$/.test(attr(a, "href") ?? ""));
  if (hasTerms && !options.termsApproved) fail("links to the terms before LANDING_TERMS_APPROVED=true");

  // Hosted-service phrases in the visible text.
  const text = stripTags(html);
  // The documentation describes the CLI, which has a login command; the phrases guard the landing copy.
  for (const locale of rel.startsWith("docs/") ? [] : Object.keys(catalogs)) {
    const matches = lang && locale.toLowerCase() === lang.toLowerCase() ? findForbidden(text, locale) : [];
    for (const phrase of matches) fail(`visible text contains the forbidden phrase "${phrase}"`);
  }

  // Weight.
  const bytes = Buffer.byteLength(html);
  if (bytes > budgets.htmlBytes) fail(`HTML is ${bytes} bytes, at most ${budgets.htmlBytes}`);
  let total = gzipSize(html);
  for (const link of tags(html, "link")) {
    const href = attr(link, "href");
    if (attr(link, "rel") === "stylesheet" && href && !isOtherOrigin(href, options.origin)) {
      const css = path.join(dir, new URL(href, options.origin).pathname);
      if (fs.existsSync(css)) {
        const size = gzipSize(fs.readFileSync(css));
        if (size > budgets.cssGzipBytes) fail(`${href} is ${size} bytes gzipped, at most ${budgets.cssGzipBytes}`);
        total += size;
      }
    }
  }
  for (const img of tags(html, "img")) {
    const src = attr(img, "src");
    if (attr(img, "loading") === "lazy" || !src || isOtherOrigin(src, options.origin)) continue;
    const file = path.join(dir, new URL(src, options.origin).pathname);
    if (fs.existsSync(file)) total += fs.statSync(file).size;
  }
  if (total > budgets.totalGzipBytes)
    fail(`page weighs ${total} bytes gzipped without lazy images, at most ${budgets.totalGzipBytes}`);

  return problems;
}

/** Every documentation page has a Markdown twin and a line of llms.txt. */
function checkDocs(dir: string, files: string[]): string[] {
  const problems: string[] = [];
  const llmsFile = path.join(dir, "llms.txt");
  const llms = fs.existsSync(llmsFile) ? fs.readFileSync(llmsFile, "utf8") : "";
  if (files.some((f) => f.endsWith(`${path.sep}docs${path.sep}index.html`)) && !llms)
    problems.push("llms.txt is missing");
  for (const file of files) {
    const rel = path.relative(dir, file).split(path.sep).join("/");
    const match = rel.match(/^docs\/(?:(.*)\/)?index\.html$/);
    if (!match) continue;
    const slug = match[1];
    const twin = slug ? `docs/${slug}.md` : "docs/index.md";
    if (!fs.existsSync(path.join(dir, twin))) problems.push(`${rel}: no Markdown twin at /${twin}`);
    if (!llms.includes(`/${twin})`)) problems.push(`${rel}: not listed in llms.txt`);
  }
  return problems;
}

/** Problems of a whole build output directory. */
export function checkBuilt(dir: string, options: BuiltOptions): string[] {
  if (!fs.existsSync(dir)) return [`${dir} does not exist: run the build first`];
  const problems: string[] = [];
  for (const name of fs.readdirSync(dir)) {
    if (SERVER_NAMES.test(name)) problems.push(`${name}: a server or worker file in the static output`);
  }
  const files = walk(dir);
  for (const file of files) {
    const rel = path.relative(dir, file).split(path.sep).join("/");
    if (/\.(m?js|cjs)$/.test(file) && NETWORK_API.test(fs.readFileSync(file, "utf8"))) {
      problems.push(`${rel}: calls a network API (fetch, XMLHttpRequest, WebSocket, EventSource or sendBeacon)`);
    }
    if (/\.css$/.test(file) && /@import\s+(url\()?["']?https?:/i.test(fs.readFileSync(file, "utf8"))) {
      problems.push(`${rel}: imports a stylesheet from another origin`);
    }
  }
  const pages = files.filter((f) => f.endsWith(".html"));
  for (const file of pages) problems.push(...checkPage(dir, file, options));

  problems.push(...checkDocs(dir, files));

  for (const locale of locales) {
    const page = path.join(dir, locale === "en" ? "index.html" : `${locale}/index.html`);
    if (!fs.existsSync(page)) {
      problems.push(`${locale}: landing page missing`);
      continue;
    }
    const html = fs.readFileSync(page, "utf8");
    const alternates = tags(html, "link").filter((l) => attr(l, "rel") === "alternate" && attr(l, "hreflang"));
    const langs = alternates.map((l) => attr(l, "hreflang")).sort();
    if (langs.join(",") !== ["en", "es", "pt-BR", "x-default"].sort().join(",")) {
      problems.push(`${locale}: hreflang alternates are ${langs.join(",")}`);
    }
    const hasLinks = (needle: string): boolean => tags(html, "a").some((a) => (attr(a, "href") ?? "").includes(needle));
    for (const needle of ["/docs/", repoUrl, `${repoUrl}/releases`]) {
      if (!hasLinks(needle)) problems.push(`${locale}: no link to ${needle}`);
    }
  }
  return problems;
}

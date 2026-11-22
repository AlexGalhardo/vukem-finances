import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { type NavItem, type NavSection, navigation } from "../content/navigation.ts";
import { parseFrontMatter } from "./front-matter.ts";

// Astro bundles this module into .astro/, so its own location is no guide there: the build runs in site/.
const here = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
export const siteRoot = fs.existsSync(path.join(here, "content")) ? here : process.cwd();
export const repoRoot = path.resolve(siteRoot, "..");
export const contentDirs = { written: "content/docs", generated: "content/generated" } as const;

export interface DocPage {
  /** Path under the content directory, without the extension: "run/docker". The home page is "index". */
  slug: string;
  title: string;
  description: string;
  /** Position inside a directory entry of the navigation (lower first); unset sorts by title. */
  order?: number;
  generated: boolean;
  /** Repository-relative path of the source file. */
  file: string;
  /** Markdown without the front matter. */
  body: string;
  /** 1-based line of the first body line in the file. */
  bodyLine: number;
}

export interface NavGroup {
  title: string;
  pages: DocPage[];
}

function walk(dir: string): string[] {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    return entry.isDirectory() ? walk(full) : [full];
  });
}

/** Reads one page file. `root` is the content directory the file lives in. */
export function readPage(root: string, file: string, generated: boolean): DocPage {
  const raw = fs.readFileSync(file, "utf8").replace(/\r\n/g, "\n");
  const parsed = parseFrontMatter(raw, path.relative(repoRoot, file));
  const slug = path.relative(root, file).split(path.sep).join("/").replace(/\.md$/, "");
  const front = parsed.data as Record<string, unknown>;
  const consumed = raw.length - parsed.content.length;
  return {
    slug,
    title: typeof front.title === "string" ? front.title : "",
    description: typeof front.description === "string" ? front.description : "",
    order: typeof front.order === "number" ? front.order : undefined,
    generated,
    file: path.relative(repoRoot, file).split(path.sep).join("/"),
    body: parsed.content,
    bodyLine: raw.slice(0, consumed).split("\n").length,
  };
}

/** Every page of the documentation, written and generated, sorted by slug. */
export function loadPages(): DocPage[] {
  const pages: DocPage[] = [];
  for (const [kind, dir] of Object.entries(contentDirs)) {
    const root = path.join(siteRoot, dir);
    for (const file of walk(root)) {
      if (file.endsWith(".md")) pages.push(readPage(root, file, kind === "generated"));
    }
  }
  return pages.sort((a, b) => a.slug.localeCompare(b.slug));
}

/** The public path of a page, with the trailing slash and without the base path. */
export function pagePath(slug: string): string {
  return slug === "index" ? "/docs/" : `/docs/${slug}/`;
}

/** The path of the Markdown twin of a page. */
export function twinPath(slug: string): string {
  return slug === "index" ? "/docs/index.md" : `/docs/${slug}.md`;
}

function expand(item: NavItem, pages: DocPage[]): DocPage[] {
  if (typeof item === "string") {
    const page = pages.find((p) => p.slug === item);
    if (!page) throw new Error(`navigation names "${item}", which is not a page`);
    return [page];
  }
  const prefix = `${item.dir}/`;
  return pages
    .filter((p) => p.slug.startsWith(prefix))
    .sort((a, b) => (a.order ?? 1e9) - (b.order ?? 1e9) || a.title.localeCompare(b.title));
}

/** The navigation with its directory entries expanded, in display order. */
export function buildNavigation(pages: DocPage[], sections: NavSection[] = navigation): NavGroup[] {
  return sections.map((section) => ({ title: section.title, pages: section.items.flatMap((i) => expand(i, pages)) }));
}

/** Slugs of the pages in the display order, each once. */
export function orderedSlugs(groups: NavGroup[]): string[] {
  return groups.flatMap((g) => g.pages.map((p) => p.slug));
}

export const lastUpdatedFile = path.join(siteRoot, "content", "last-updated.json");
export interface LastUpdatedEntry {
  date: string;
  hash: string;
}

export function readLastUpdated(): Record<string, LastUpdatedEntry> {
  if (!fs.existsSync(lastUpdatedFile)) return {};
  return JSON.parse(fs.readFileSync(lastUpdatedFile, "utf8")) as Record<string, LastUpdatedEntry>;
}

/** A hash of the whole source file with LF line ends, so Windows and Linux agree. */
export function sourceHash(page: DocPage): string {
  const file = path.join(repoRoot, page.file);
  const text = fs.readFileSync(file, "utf8").replace(/\r\n/g, "\n");
  return createHash("sha256").update(text).digest("hex").slice(0, 16);
}

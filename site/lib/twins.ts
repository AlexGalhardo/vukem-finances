import type { DocPage } from "./docs.ts";
import { pagePath, twinPath } from "./docs.ts";

/** The address of a path on the public site, base path included. */
function publicUrl(origin: string, path: string): string {
  const base = new URL(origin);
  return `${base.origin}${base.pathname.replace(/\/$/, "")}${path}`;
}

/** The Markdown twin of a page: the title, the summary and the body, with links to the twins of other pages. */
export function twinMarkdown(page: DocPage, origin: string): string {
  const body = page.body.replace(/\]\((\/docs\/[^)\s#]*)(#[^)\s]*)?\)/g, (_m, path: string, anchor?: string) => {
    const slug = path === "/docs/" ? "index" : path.replace(/^\/docs\//, "").replace(/\/$/, "");
    return `](${publicUrl(origin, twinPath(slug))}${anchor ?? ""})`;
  });
  return `# ${page.title}\n\n${page.description}\n\nCanonical page: ${publicUrl(origin, pagePath(page.slug))}\n\n${body.trim()}\n`;
}

/** llms.txt: every page with its summary, so a tool can read the documentation without scraping HTML. */
export function llmsTxt(pages: DocPage[], origin: string, order: string[]): string {
  const sorted = [...pages].sort((a, b) => order.indexOf(a.slug) - order.indexOf(b.slug));
  const lines = sorted.map((p) => `- [${p.title}](${publicUrl(origin, twinPath(p.slug))}): ${p.description}`);
  return [
    "# vukem-finances documentation",
    "",
    "> Guides and references for running, using and integrating vukem-finances, an open-source personal finance app. Every page is also available as Markdown at the address listed here.",
    "",
    "## Pages",
    "",
    ...lines,
    "",
  ].join("\n");
}

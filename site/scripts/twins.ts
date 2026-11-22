import fs from "node:fs";
import path from "node:path";
import { buildNavigation, loadPages, orderedSlugs, siteRoot, twinPath } from "../lib/docs.ts";
import { llmsTxt, twinMarkdown } from "../lib/twins.ts";

// Writes the Markdown twin of every documentation page and llms.txt next to the built pages.
const out = path.resolve(siteRoot, process.env.SITE_OUT || "out");
const origin = process.env.SITE_URL || "https://finances.vukem.dev";
const pages = loadPages();

for (const page of pages) {
  const file = path.join(out, twinPath(page.slug));
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, twinMarkdown(page, origin));
}
fs.writeFileSync(path.join(out, "llms.txt"), llmsTxt(pages, origin, orderedSlugs(buildNavigation(pages))));
console.log(`twins: ${pages.length} pages and llms.txt`);

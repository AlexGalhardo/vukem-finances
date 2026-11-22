import { checkContent } from "../lib/content-check.ts";
import { loadPages } from "../lib/docs.ts";

const pages = loadPages();
const problems = await checkContent(pages);
if (problems.length > 0) {
  console.error(`check:mdx failed:\n${problems.map((p) => `  ${p}`).join("\n")}`);
  process.exit(1);
}
console.log(`check:mdx ok (${pages.length} pages)`);

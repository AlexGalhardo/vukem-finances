import { checkClaims } from "../lib/doc-claims.ts";
import { buildContext, toClaimPages } from "../lib/doc-context.ts";
import { loadPages } from "../lib/docs.ts";

const pages = loadPages();
const problems = checkClaims(toClaimPages(pages), await buildContext(pages));
if (problems.length > 0) {
  console.error(`check:claims failed (documentation):\n${problems.map((p) => `  ${p}`).join("\n")}`);
  process.exit(1);
}
console.log(`check:claims ok (documentation, ${pages.length} pages)`);

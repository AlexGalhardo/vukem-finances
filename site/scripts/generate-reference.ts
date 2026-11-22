import fs from "node:fs";
import path from "node:path";
import { repoRoot } from "../lib/docs.ts";
import {
  changelogPages,
  cliPages,
  envPage,
  type GeneratedPage,
  type InventoryOperation,
  mcpPages,
  type OpenAPI,
  pageFile,
  restPages,
  shortcutsPage,
} from "../lib/reference.ts";
import { generatedDir, sources } from "../lib/reference-config.ts";

function read(file: string): string {
  if (!fs.existsSync(file)) {
    console.error(`generate-reference: the source ${path.relative(repoRoot, file)} does not exist`);
    process.exit(1);
  }
  return fs.readFileSync(file, "utf8");
}

/** Every generated page, from the real sources of the repository. */
export function generateAll(): GeneratedPage[] {
  const openapi = JSON.parse(read(sources.openapi)) as OpenAPI;
  const inventory = (JSON.parse(read(sources.operations)) as { operations: InventoryOperation[] }).operations;
  const cli = read(sources.cli);
  const mcp = read(sources.mcp);
  return [
    ...restPages(openapi, inventory),
    ...cliPages(cli),
    ...mcpPages(mcp, inventory),
    envPage(read(sources.env)),
    shortcutsPage(read(sources.shortcuts)),
    ...changelogPages(read(sources.changelog)),
  ];
}

if (import.meta.main) {
  fs.rmSync(generatedDir, { recursive: true, force: true });
  const pages = generateAll();
  for (const page of pages) {
    const file = path.join(generatedDir, `${page.slug}.md`);
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, pageFile(page));
  }
  console.log(`reference: ${pages.length} pages in content/generated`);
}

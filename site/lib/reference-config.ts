import path from "node:path";
import { repoRoot, siteRoot } from "./docs.ts";

// Every source the generated pages read. A missing file stops the generator and names the path.
export const sources = {
  openapi: path.join(repoRoot, "docs", "openapi.json"),
  operations: path.join(repoRoot, "resources", "operations.json"),
  cli: path.join(repoRoot, "docs", "reference", "cli.md"),
  mcp: path.join(repoRoot, "docs", "reference", "mcp.md"),
  env: path.join(repoRoot, ".env.example"),
  shortcuts: path.join(repoRoot, "docs", "shortcuts.md"),
  changelog: path.join(repoRoot, "CHANGELOG.md"),
} as const;

export const generatedDir = path.join(siteRoot, "content", "generated");

import fs from "node:fs";
import path from "node:path";
import type { ClaimContext, ClaimPage } from "./doc-claims.ts";
import { type DocPage, repoRoot } from "./docs.ts";
import { cliCommands, mcpResourceNames, mcpTools, type OpenAPI } from "./reference.ts";
import { sources } from "./reference-config.ts";
import { renderMarkdown } from "./render.ts";

// Commands of the application binary that are not resource commands. Each is declared by the framework or by
// bootstrap/; the claims check only needs to know they are real commands, not CLI reference commands.
export const serverCommands = [
  "about",
  "migrate",
  "mcp:serve",
  "db:backup",
  "user:reset-password",
  "telegram:set-webhook",
  "telegram:info",
  "operations:inventory",
  "cli:reference",
  "mcp:reference",
];

const CORPUS_DIRS = ["app", "bootstrap", "config", "routes", "cmd", "database", "docs", "deploy", "scripts"];
const CORPUS_FILES = [".env.example", "compose.yaml", "compose.prod.yaml", "Dockerfile", "AGENTS.md"];

function walk(dir: string): string[] {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return entry.name === "node_modules" || entry.name === "testdata" ? [] : walk(full);
    return /\.(go|md|ya?ml|sh|json|example)$|Caddyfile$/.test(entry.name) ? [full] : [];
  });
}

/** Every SCREAMING_SNAKE word of the sources that define or document the product's settings. */
function envCorpus(): Set<string> {
  const words = new Set<string>();
  const files = [
    ...CORPUS_DIRS.flatMap((d) => walk(path.join(repoRoot, d))),
    ...CORPUS_FILES.map((f) => path.join(repoRoot, f)),
  ];
  for (const file of files) {
    if (!fs.existsSync(file) || file.endsWith("openapi.json")) continue;
    for (const m of fs.readFileSync(file, "utf8").matchAll(/\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b/g)) words.add(m[0]);
  }
  return words;
}

export function restOperations(api: OpenAPI): { method: string; path: string }[] {
  return Object.entries(api.paths).flatMap(([p, item]) =>
    Object.keys(item)
      .filter((m) => ["get", "post", "patch", "put", "delete"].includes(m))
      .map((m) => ({ method: m.toUpperCase(), path: p })),
  );
}

/** The context built from the real sources of the repository and the rendered anchors of the pages. */
export async function buildContext(pages: DocPage[]): Promise<ClaimContext> {
  const read = (file: string): string => fs.readFileSync(file, "utf8");
  const api = JSON.parse(read(sources.openapi)) as OpenAPI;
  const corpus = envCorpus();
  const anchors = new Map<string, Set<string>>();
  for (const page of pages) {
    const { headings } = await renderMarkdown(page.body);
    anchors.set(page.slug, new Set(headings.map((h) => h.id)));
  }
  return {
    cliCommands: cliCommands(read(sources.cli)),
    serverCommands: new Set(serverCommands),
    isEnvVar: (name) => corpus.has(name),
    pathExists: (rel) => fs.existsSync(path.join(repoRoot, rel)),
    restOperations: restOperations(api),
    mcpTools: new Set([...mcpTools(read(sources.mcp)).map((t) => t.tool), ...mcpResourceNames(read(sources.mcp))]),
    anchors,
  };
}

export const toClaimPages = (pages: DocPage[]): ClaimPage[] =>
  pages.map((p) => ({ slug: p.slug, file: p.file, body: p.body, bodyLine: p.bodyLine }));

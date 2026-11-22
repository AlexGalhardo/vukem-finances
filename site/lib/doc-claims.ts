import { repoUrl } from "./site.ts";

// The documentation claims check: everything a page says about the product must exist in the repository.
// The check is pure: scripts/check-doc-claims.ts builds the context from the real sources, and the tests
// build a small one and plant failures.

export interface ClaimPage {
  slug: string;
  /** Repository-relative file, named in every problem. */
  file: string;
  body: string;
  /** 1-based line of the first body line in the file. */
  bodyLine: number;
}

export interface RestOperation {
  method: string;
  /** Path template without the /api/v1 prefix, such as "/accounts/{uid}". */
  path: string;
}

export interface ClaimContext {
  /** "accounts list" for every command of the CLI reference. */
  cliCommands: Set<string>;
  /** Commands of the application binary that are not resource commands (mcp:serve, db:backup, ...). */
  serverCommands: Set<string>;
  /** Whether a name is an environment variable of the product (.env.example, compose files, the code). */
  isEnvVar: (name: string) => boolean;
  /** Whether a repository-relative path exists. */
  pathExists: (path: string) => boolean;
  restOperations: RestOperation[];
  mcpTools: Set<string>;
  /** Heading anchors of every page, by slug. */
  anchors: Map<string, Set<string>>;
}

export const cliName = "app";
export const authVerbs = new Set(["login", "logout", "status"]);
export const completionShells = new Set(["bash", "zsh", "fish", "powershell"]);
const SHELL_LANGS = new Set(["sh", "bash", "shell", "console"]);
const ROOT_DIRS = new Set([
  "app",
  "docs",
  "deploy",
  "resources",
  "scripts",
  "config",
  "routes",
  "bootstrap",
  "cmd",
  "database",
  "clients",
  "mobile",
  "desktop",
  "site",
  "e2e",
  "testdata",
  "tests",
  ".github",
  ".agents",
]);
const ROOT_FILES = new Set([
  "compose.yaml",
  "compose.prod.yaml",
  ".env.example",
  "Dockerfile",
  "AGENTS.md",
  "CLAUDE.md",
  "CHANGELOG.md",
  "README.md",
  "LICENSE",
]);
const RESOURCE_PREFIXES = new Set([
  "accounts",
  "budgets",
  "cards",
  "categories",
  "export",
  "goals",
  "recurrences",
  "reports",
  "statements",
  "summary",
  "tags",
  "transactions",
  "transfers",
]);

interface Segment {
  text: string;
  /** "inline" for a code span, the fence language for fenced code, "prose" for the rest. */
  kind: string;
  line: number;
}

/** Splits a body into prose, inline code and fenced code, keeping the line of each piece. */
export function segments(page: ClaimPage): Segment[] {
  const out: Segment[] = [];
  let fence: string | null = null;
  page.body.split("\n").forEach((raw, index) => {
    const line = page.bodyLine + index;
    const open = raw.match(/^\s*(```+|~~~+)\s*([A-Za-z0-9_-]*)/);
    if (open && fence === null) {
      fence = open[2] || "text";
      return;
    }
    if (fence !== null) {
      if (/^\s*(```+|~~~+)\s*$/.test(raw)) fence = null;
      else out.push({ text: raw, kind: fence, line });
      return;
    }
    let rest = "";
    let last = 0;
    for (const span of raw.matchAll(/`([^`]+)`/g)) {
      rest += raw.slice(last, span.index);
      out.push({ text: span[1], kind: "inline", line });
      last = (span.index ?? 0) + span[0].length;
    }
    rest += raw.slice(last);
    out.push({ text: rest, kind: "prose", line });
  });
  return out;
}

/** The shell commands of a fenced block: backslash continuations joined, comments and blanks dropped. */
export function shellCommands(lines: { text: string; line: number }[]): { text: string; line: number }[] {
  const commands: { text: string; line: number }[] = [];
  let current: { text: string; line: number } | null = null;
  for (const { text, line } of lines) {
    const trimmed = text.trim();
    if (!current && (trimmed === "" || trimmed.startsWith("#"))) continue;
    const continued = trimmed.endsWith("\\");
    const piece = continued ? trimmed.slice(0, -1).trim() : trimmed;
    current = current ? { text: `${current.text} ${piece}`, line: current.line } : { text: piece, line };
    if (!continued) {
      commands.push(current);
      current = null;
    }
  }
  if (current) commands.push(current);
  return commands;
}

function templateMatches(template: string, path: string): boolean {
  const a = template.split("/");
  const b = path.split("/");
  return a.length === b.length && a.every((seg, i) => /^\{.+\}$/.test(seg) || seg === b[i] || b[i] === "");
}

function repoPathOf(token: string): string | null {
  const text = token.trim().replace(/[:#].*$/, "");
  if (/[*<>{}\s]|\.\.\./.test(text)) return null;
  if (ROOT_FILES.has(text)) return text;
  const first = text.split("/")[0];
  if (text.includes("/") && ROOT_DIRS.has(first) && /^[A-Za-z0-9_./-]+$/.test(text)) return text.replace(/\/$/, "");
  return null;
}

/** Problems of the pages, each naming the file and the line. */
export function checkClaims(pages: ClaimPage[], ctx: ClaimContext): string[] {
  const problems: string[] = [];
  const slugs = new Set(pages.map((p) => p.slug));
  for (const page of pages) {
    const fail = (line: number, message: string): number => problems.push(`${page.file}:${line}: ${message}`);
    const parts = segments(page);

    // Shell commands.
    const shell = parts.filter((p) => SHELL_LANGS.has(p.kind));
    for (const command of shellCommands(shell)) {
      const tokens = command.text.replace(/^\$\s+/, "").split(/\s+/);
      if (tokens[0] === cliName) {
        const resource = tokens[1] ?? "";
        const verb = tokens[2] ?? "";
        const known = [...ctx.cliCommands].some((c) => c.startsWith(`${resource} `));
        if (resource === "auth") {
          if (!authVerbs.has(verb)) fail(command.line, `"app auth ${verb}" is not a command of the CLI reference`);
        } else if (resource === "completion") {
          if (!completionShells.has(verb))
            fail(command.line, `"app completion ${verb}" is not a shell of the CLI reference`);
        } else if (known) {
          if (!ctx.cliCommands.has(`${resource} ${verb}`))
            fail(command.line, `"app ${resource} ${verb}" is not a command of the CLI reference`);
        } else if (!ctx.serverCommands.has(resource)) {
          fail(command.line, `"app ${resource}" is not a command of the CLI reference or of the application`);
        }
      }
      if (tokens[0] === "docker" && tokens[1] === "compose") {
        tokens.forEach((token, i) => {
          if ((token === "-f" || token === "--file") && tokens[i + 1] && !ctx.pathExists(tokens[i + 1])) {
            fail(command.line, `the compose file ${tokens[i + 1]} does not exist`);
          }
        });
      }
    }

    for (const part of parts) {
      const isCode = part.kind !== "prose";

      // Environment variables.
      if (isCode) {
        for (const m of part.text.matchAll(/\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b/g)) {
          if (!ctx.isEnvVar(m[0])) fail(part.line, `${m[0]} is not a setting of the product`);
        }
      }

      // Repository paths in inline code.
      if (part.kind === "inline") {
        const rel = repoPathOf(part.text);
        if (rel && !ctx.pathExists(rel)) fail(part.line, `the path ${rel} does not exist in the repository`);
      }

      // REST paths, and the method when an inline span names one.
      if (isCode) {
        for (const m of part.text.matchAll(
          /(?:\b(GET|POST|PATCH|PUT|DELETE)\s+)?(?:\$BASE)?(\/api\/v1(?:\/[A-Za-z0-9_{}$<>.-]+)*)/g,
        )) {
          const segs = m[2]
            .replace(/^\/api\/v1/, "")
            .split("/")
            .filter(Boolean);
          if (segs.includes("...")) continue;
          const path = segs.length ? `/${segs.join("/")}` : "";
          if (path === "" || path === "/openapi.json") continue;
          const matching = ctx.restOperations.filter((o) => templateMatches(o.path, path));
          if (matching.length === 0) fail(part.line, `the REST path /api/v1${path} is not in the OpenAPI document`);
          else if (m[1] && part.kind === "inline" && !matching.some((o) => o.method === m[1])) {
            fail(part.line, `the REST route ${m[1]} /api/v1${path} is not in the OpenAPI document`);
          }
        }
      }

      // MCP tools.
      if (part.kind === "inline" && /^[a-z]+(?:_[a-z]+)+$/.test(part.text)) {
        if (RESOURCE_PREFIXES.has(part.text.split("_")[0]) && !ctx.mcpTools.has(part.text)) {
          fail(part.line, `the MCP tool ${part.text} is not in the MCP reference`);
        }
      }

      if (part.kind !== "prose") continue;

      // Links and bare URLs, in prose only.
      for (const m of part.text.matchAll(/\[[^\]]*\]\(([^)\s]*)\)/g)) {
        const href = m[1];
        if (href.startsWith("#")) {
          if (!ctx.anchors.get(page.slug)?.has(href.slice(1)))
            fail(part.line, `the anchor ${href} is not a heading of this page`);
        } else if (href.startsWith("/docs/") || href === "/docs") {
          const [path, anchor] = href.split("#");
          const slug = path === "/docs/" ? "index" : path.replace(/^\/docs\//, "").replace(/\/$/, "");
          if (!path.endsWith("/")) fail(part.line, `the link ${href} must end with a slash`);
          else if (!slugs.has(slug)) fail(part.line, `the link ${href} points at no page`);
          else if (anchor && !ctx.anchors.get(slug)?.has(anchor))
            fail(part.line, `the link ${href} points at no heading`);
        } else if (href.startsWith(repoUrl)) {
          const match = href.match(/\/(?:blob|tree)\/main\/([^#?]+)/);
          if (match && !ctx.pathExists(match[1].replace(/\/$/, "")))
            fail(part.line, `the repository path ${match[1]} does not exist`);
        } else if (/^[a-z][a-z0-9+.-]*:/i.test(href)) {
          fail(part.line, `the link ${href} leaves the site: only links to the repository are allowed`);
        } else {
          fail(part.line, `the link ${href} must be an absolute /docs/ path`);
        }
      }
      const withoutLinks = part.text.replace(/\[[^\]]*\]\(([^)\s]*)\)/g, " ");
      if (/https?:\/\//.test(withoutLinks)) {
        fail(part.line, "a bare URL becomes a link to another origin: put it in code or link the repository");
      }
    }
  }
  return problems;
}

// Pure generators of the reference pages: each takes the text or JSON of a source and returns pages.
// scripts/generate-reference.ts reads the sources and writes the files.

export interface GeneratedPage {
  /** Path under content/generated without the extension. */
  slug: string;
  title: string;
  description: string;
  order?: number;
  /** Source file the page comes from, shown in the banner. */
  source: string;
  body: string;
}

export function banner(source: string): string {
  return `> Generated from \`${source}\`. Do not edit this page: change the source and run \`bun run reference\` in \`site/\`.\n`;
}

/** Renders a page file: front matter, then the banner and the body. */
export function pageFile(page: GeneratedPage): string {
  const front = [
    "---",
    `title: ${JSON.stringify(page.title)}`,
    `description: ${JSON.stringify(page.description)}`,
    ...(page.order === undefined ? [] : [`order: ${page.order}`]),
    "---",
    "",
  ];
  return `${front.join("\n")}\n${banner(page.source)}\n${page.body.trim()}\n`.replace(/\n{3,}/g, "\n\n");
}

const cell = (text: string): string =>
  text
    .replace(/\|/g, "\\|")
    .replace(/\s*\n\s*/g, " ")
    .trim();
const code = (text: string): string => `\`${text.replace(/`/g, "")}\``;
/** Prose of a table cell: URLs become code, and raw HTML and emphasis characters are escaped. */
const prose = (text: string): string =>
  text
    .split(/(`[^`]*`|https?:\/\/[^\s)]+[^\s).,;])/)
    .map((part, i) => {
      if (i % 2 === 0) return part.replace(/([<>*])/g, "\\$1");
      return part.startsWith("`") ? part : code(part);
    })
    .join("");

function table(head: string[], rows: string[][]): string {
  const line = (cells: string[]): string => `| ${cells.join(" | ")} |`;
  return [line(head), line(head.map(() => "---")), ...rows.map((r) => line(r.map(cell)))].join("\n");
}

// ---------------------------------------------------------------------------------------------- REST

interface Schema {
  $ref?: string;
  type?: string | string[];
  format?: string;
  description?: string;
  enum?: unknown[];
  items?: Schema;
  properties?: Record<string, Schema>;
  required?: string[];
  anyOf?: Schema[];
  examples?: unknown[];
  example?: unknown;
  maxLength?: number;
}
interface Parameter {
  name: string;
  in: string;
  required?: boolean;
  description?: string;
  example?: unknown;
  schema?: Schema;
}
interface ResponseObject {
  $ref?: string;
  description?: string;
  content?: Record<string, { schema: Schema }>;
}
interface Operation {
  operationId: string;
  summary?: string;
  tags?: string[];
  parameters?: Parameter[];
  requestBody?: { content: Record<string, { schema: Schema }> };
  responses: Record<string, ResponseObject>;
  "x-required-scope"?: string;
  deprecated?: boolean;
}
export interface OpenAPI {
  paths: Record<string, Record<string, Operation>>;
  components: { schemas: Record<string, Schema>; responses: Record<string, { description: string }> };
}
export interface InventoryOperation {
  name: string;
  summary: string;
  scope: string;
  kind: "read" | "write" | "destructive";
  idempotent: boolean;
  editions: string[];
  surfaces: { surface: string; offered: boolean; route?: string; reason?: string }[];
  input: { name: string; type: string; required?: boolean }[];
}

const METHODS = ["get", "post", "patch", "put", "delete"];

// The stable error codes of each failure status, from docs/api.md ("Requests and answers").
export const errorCodes: Record<string, string> = {
  "400": "bad_request",
  "401": "unauthorized",
  "402": "plan_limit",
  "403": "missing_scope, forbidden",
  "404": "not_found",
  "409": "conflict, version_conflict",
  "422": "validation_failed",
  "429": "rate_limited",
  "503": "unavailable",
};

/** "Envelope_github.com_x_app_api.AccountOut" becomes "Envelope<AccountOut>". */
export function schemaName(ref: string): string {
  const name = ref.replace(/^#\/components\/schemas\//, "");
  const envelope = name.match(/^Envelope_.*\.([A-Za-z0-9]+)$/);
  return envelope ? `Envelope<${envelope[1]}>` : name.replace(/^.*\./, "");
}

export function typeOf(schema: Schema | undefined): string {
  if (!schema) return "any";
  if (schema.$ref) return schemaName(schema.$ref);
  if (schema.anyOf) {
    const parts = schema.anyOf.map(typeOf).filter((t) => t !== "null");
    return parts.join(" or ") || "null";
  }
  const types = (Array.isArray(schema.type) ? schema.type : [schema.type ?? "object"]).filter((t) => t !== "null");
  const base = types.join(" or ") || "null";
  if (base === "array") return `array of ${typeOf(schema.items)}`;
  if (schema.enum) return `${base}: ${schema.enum.map((v) => JSON.stringify(v)).join(", ")}`;
  return schema.format ? `${base} (${schema.format})` : base;
}

function resolve(api: OpenAPI, schema: Schema | undefined): Schema | undefined {
  if (schema?.$ref) return api.components.schemas[schema.$ref.replace("#/components/schemas/", "")];
  return schema;
}

function propertyRows(api: OpenAPI, schema: Schema | undefined): string[][] {
  const resolved = resolve(api, schema);
  if (!resolved?.properties) return [];
  const required = new Set(resolved.required ?? []);
  return Object.entries(resolved.properties).map(([name, prop]) => [
    code(name),
    code(typeOf(prop)),
    required.has(name) ? "yes" : "",
    [prose(prop.description ?? ""), prop.maxLength ? `At most ${prop.maxLength} characters.` : ""]
      .filter(Boolean)
      .join(" "),
  ]);
}

function example(api: OpenAPI, schema: Schema | undefined, depth = 0): unknown {
  const s = resolve(api, schema);
  if (!s || depth > 3) return null;
  if (s.examples?.length) return s.examples[0];
  if (s.example !== undefined) return s.example;
  if (s.properties) {
    const names = s.required?.length ? s.required : Object.keys(s.properties).slice(0, 3);
    return Object.fromEntries(names.map((name) => [name, example(api, s.properties?.[name], depth + 1)]));
  }
  if (s.anyOf) return example(api, s.anyOf.find((a) => a.type !== "null") ?? s.anyOf[0], depth);
  const type = Array.isArray(s.type) ? s.type.find((t) => t !== "null") : s.type;
  if (type === "array") return [example(api, s.items, depth + 1)];
  if (type === "integer" || type === "number") return 0;
  if (type === "boolean") return false;
  if (type === "string") return s.enum?.[0] ?? "";
  return null;
}

function curl(api: OpenAPI, method: string, path: string, op: Operation): string {
  const url = path.replace(/\{([^}]+)\}/g, (_m, name: string) => `<${name}>`);
  const query = (op.parameters ?? []).filter((p) => p.in === "query" && p.required);
  const qs = query.length ? `?${query.map((p) => `${p.name}=${p.example ?? "..."}`).join("&")}` : "";
  const lines = [`curl${method === "get" ? "" : ` -X ${method.toUpperCase()}`} -H "Authorization: Bearer vf_..." \\`];
  const body = op.requestBody?.content["application/json"]?.schema;
  if (body) {
    lines.push('  -H "Content-Type: application/json" \\');
    lines.push(`  -d '${JSON.stringify(example(api, body))}' \\`);
  }
  lines.push(`  "$BASE/api/v1${url}${qs}"`);
  return lines.join("\n");
}

function operationSection(
  api: OpenAPI,
  inventory: Map<string, InventoryOperation>,
  method: string,
  path: string,
  op: Operation,
): string {
  const out: string[] = [`## ${code(`${method.toUpperCase()} ${path}`)}`, ""];
  out.push(`${op.summary ?? op.operationId}.`, "");
  const inv = inventory.get(op.operationId);
  const facts: string[][] = [
    ["Operation", code(op.operationId)],
    ["Scope", op["x-required-scope"] ? code(op["x-required-scope"]) : "none, any valid token"],
  ];
  if (inv) {
    facts.push(["Kind", inv.kind === "read" ? "read only" : inv.kind]);
    facts.push(["Idempotency-Key", inv.idempotent ? "honored" : "not used"]);
    facts.push(["Editions", inv.editions.join(", ")]);
  } else {
    facts.push(["Kind", op.operationId.startsWith("sync.") ? "REST only, sync protocol" : "REST only"]);
  }
  if (op.deprecated) facts.push(["Deprecated", "yes"]);
  out.push(table(["Field", "Value"], facts), "");

  const params = op.parameters ?? [];
  if (params.length) {
    out.push("### Parameters", "");
    out.push(
      table(
        ["Name", "In", "Type", "Required", "Description"],
        params.map((p) => [
          code(p.name),
          p.in,
          code(typeOf(p.schema)),
          p.required ? "yes" : "",
          prose(p.description ?? ""),
        ]),
      ),
      "",
    );
  }
  const body = op.requestBody?.content["application/json"]?.schema;
  if (body) {
    out.push("### Request body", "");
    const rows = propertyRows(api, body);
    out.push(
      rows.length ? table(["Field", "Type", "Required", "Description"], rows) : `A JSON value of type ${typeOf(body)}.`,
      "",
    );
  }
  out.push("### Responses", "");
  const responseRows = Object.entries(op.responses).map(([status, r]) => {
    const ref = r.$ref?.replace("#/components/responses/", "");
    const meta = ref ? api.components.responses[ref] : r;
    const schema = r.content?.["application/json"]?.schema;
    const payload = schema ? code(typeOf(schema)) : ref ? "problem+json" : "";
    const codes = /^[2]/.test(status) ? "" : errorCodes[status] ? code(errorCodes[status]) : "";
    return [status === "default" ? "other" : status, prose(meta?.description ?? ""), payload, codes];
  });
  out.push(table(["Status", "Meaning", "Body", "Error code"], responseRows), "");

  const success = Object.entries(op.responses).find(([s]) => s.startsWith("2"));
  const successSchema = success?.[1].content?.["application/json"]?.schema;
  const resolved = resolve(api, successSchema);
  const list = resolved?.properties?.data?.items;
  const fields = propertyRows(api, list ?? successSchema);
  if (fields.length) {
    out.push(
      `### Fields of ${code(typeOf(list ?? successSchema))}`,
      "",
      table(["Field", "Type", "Required", "Description"], fields),
      "",
    );
  }
  out.push("### Example", "", "```sh", curl(api, method, path, op), "```", "");
  return out.join("\n");
}

export function tagSlug(tag: string): string {
  return tag.toLowerCase().replace(/[^a-z0-9]+/g, "-");
}

export function restPages(api: OpenAPI, inventory: InventoryOperation[]): GeneratedPage[] {
  const byName = new Map(inventory.map((o) => [o.name, o]));
  const groups = new Map<string, string[]>();
  for (const [path, item] of Object.entries(api.paths).sort(([a], [b]) => a.localeCompare(b))) {
    for (const method of METHODS) {
      const op = item[method];
      if (!op) continue;
      const tag = op.tags?.[0] ?? "Other";
      const list = groups.get(tag) ?? [];
      list.push(operationSection(api, byName, method, path, op));
      groups.set(tag, list);
    }
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([tag, sections]) => ({
      slug: `integrate/rest/${tagSlug(tag)}`,
      title: `REST: ${tag}`,
      description: `The ${tag} endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples.`,
      source: "docs/openapi.json",
      body: `${sections.length} ${sections.length === 1 ? "operation" : "operations"}. Every request needs \`Authorization: Bearer vf_...\`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.\n\n${sections.join("\n")}`,
    }));
}

// ---------------------------------------------------------------------------------------------- CLI

/** Splits docs/reference/cli.md into an overview page and one page per resource. */
export function cliPages(markdown: string): GeneratedPage[] {
  const text = markdown.replace(/\r\n/g, "\n").replace(/<!--[\s\S]*?-->\n?/g, "");
  const parts = text.split(/^## /m);
  const intro = (parts.shift() ?? "").replace(/^# .*\n+/, "").trim();
  const common: string[] = [];
  const resources: { name: string; body: string }[] = [];
  for (const part of parts) {
    const name = part.split("\n", 1)[0].trim();
    if (/^(Modes|Shared flags|Exit codes|Other commands)$/.test(name)) common.push(`## ${part.trimEnd()}`);
    else resources.push({ name, body: part.split("\n").slice(1).join("\n") });
  }
  const pages: GeneratedPage[] = [
    {
      slug: "integrate/cli/overview",
      title: "CLI: modes, flags, exit codes",
      description:
        "How the command line runs locally or against a server, the flags every command shares and its exit codes.",
      order: 0,
      source: "docs/reference/cli.md",
      body: `${intro}\n\n${common.join("\n\n")}`,
    },
  ];
  resources.forEach((r, i) => {
    pages.push({
      slug: `integrate/cli/${r.name}`,
      title: `CLI: ${r.name}`,
      description: `The ${r.name} commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each.`,
      order: i + 1,
      source: "docs/reference/cli.md",
      body: r.body.replace(/^### /gm, "## ").replace(/^#### /gm, "### "),
    });
  });
  return pages;
}

/** Every `<resource> <verb>` command named by the CLI reference. */
export function cliCommands(markdown: string): Set<string> {
  const commands = new Set<string>();
  for (const m of markdown.matchAll(/^### `([a-z]+ [a-z-]+)`/gm)) commands.add(m[1]);
  return commands;
}

// ---------------------------------------------------------------------------------------------- MCP

export interface McpTool {
  tool: string;
  operation: string;
  kind: string;
  scope: string;
  toolsets: string;
}

function sectionOf(markdown: string, heading: string): string {
  const start = markdown.search(new RegExp(`^## ${heading}$`, "m"));
  if (start < 0) return "";
  const rest = markdown.slice(start).split("\n").slice(1).join("\n");
  const next = rest.search(/^## /m);
  return next < 0 ? rest : rest.slice(0, next);
}

const unquote = (s: string): string => s.replace(/^`|`$/g, "");

function tableRows(section: string): string[][] {
  return section
    .split("\n")
    .filter((l) => l.startsWith("|"))
    .slice(2)
    .map((l) =>
      l
        .replace(/^\||\|$/g, "")
        .split(/(?<!\\)\|/)
        .map((c) => c.trim()),
    );
}

export function mcpTools(markdown: string): McpTool[] {
  return tableRows(sectionOf(markdown, "Tools")).map(([tool, operation, kind, scope, toolsets]) => ({
    tool: unquote(tool),
    operation: unquote(operation),
    kind,
    scope: unquote(scope),
    toolsets,
  }));
}

export function mcpPages(markdown: string, inventory: InventoryOperation[]): GeneratedPage[] {
  const text = markdown.replace(/\r\n/g, "\n");
  const byName = new Map(inventory.map((o) => [o.name, o]));
  const tools = mcpTools(text);
  const toolSections = tools.map((t) => {
    const op = byName.get(t.operation);
    const kind = t.kind === "destructive" ? "destructive, asks you to confirm every call" : t.kind;
    const lines = [
      `## ${code(t.tool)}`,
      "",
      `${op?.summary ?? t.operation}.`,
      "",
      `- Operation: ${code(t.operation)}, ${kind}, scope ${code(t.scope)}.`,
      `- Toolsets: ${t.toolsets}.`,
      op?.input.length
        ? `- Input: ${op.input.map((i) => `${code(i.name)} (${i.type}${i.required ? ", required" : ""})`).join(", ")}.`
        : "- Input: none.",
      "",
    ];
    return lines.join("\n");
  });
  const toolsets = sectionOf(text, "Toolsets").trim();
  const afterTable = sectionOf(text, "Tools").split("Operations with no tool")[1] ?? "";
  return [
    {
      slug: "integrate/mcp/tools",
      title: "MCP: tools",
      description: "Every tool of the vukem-finances MCP server with its operation, kind, scope, toolsets and input.",
      order: 0,
      source: "docs/reference/mcp.md",
      body: `${tools.length} tools. A tool name is the operation name with an underscore for each dot. The \`MCP_TOOLSETS\` setting decides which tools a server offers; [MCP over stdio](/docs/integrate/mcp-stdio/) explains how to set it.\n\n## Toolsets\n\n${toolsets}\n\n${toolSections.join("\n")}\n## Operations with no tool\n\n${afterTable.trim() ? `Each one has a written reason.\n${afterTable.replace(/^[^\n]*\n/, "").trimEnd()}` : "None."}\n`,
    },
    {
      slug: "integrate/mcp/resources-prompts",
      title: "MCP: resources and prompts",
      description: "The read-only resources, resource templates and prompts of the vukem-finances MCP server.",
      order: 1,
      source: "docs/reference/mcp.md",
      body: `## Resources\n\n${sectionOf(text, "Resources").trim()}\n\n## Resource templates\n\n${sectionOf(text, "Resource templates").trim()}\n\n## Prompts\n\n${sectionOf(text, "Prompts").trim()}\n`,
    },
  ];
}

export function mcpResourceNames(markdown: string): string[] {
  const text = markdown.replace(/\r\n/g, "\n");
  return ["Resources", "Resource templates", "Prompts"].flatMap((h) =>
    tableRows(sectionOf(text, h)).map((r) => unquote(r[0])),
  );
}

// ---------------------------------------------------------------------------------------------- Environment

export interface EnvVar {
  name: string;
  value: string;
  description: string;
  commented: boolean;
}

// Where a variable matters. The rules mirror the comments of .env.example and docs/guide.md.
const EDITION_RULES: [RegExp, string][] = [
  [/^(STRIPE_|ABACATEPAY_|PIX_|MCP_OAUTH_CLIENTS$)/, "saas"],
  [/^(SETUP_TOKEN|AI_ALLOW_PRIVATE_HOSTS)$/, "personal"],
];

export function editionOf(name: string): string {
  return EDITION_RULES.find(([re]) => re.test(name))?.[1] ?? "both";
}

/** Reads .env.example: a variable takes the comment lines right above it; a commented-out variable is optional. */
export function envVars(text: string): EnvVar[] {
  const vars: EnvVar[] = [];
  let comment: string[] = [];
  for (const raw of text.replace(/\r\n/g, "\n").split("\n")) {
    const line = raw.trimEnd();
    const live = line.match(/^([A-Z][A-Z0-9_]*)=(.*)$/);
    const off = line.match(/^# ([A-Z][A-Z0-9_]*)=(\[.*|\S*)$/);
    const match = live ?? off;
    if (match) {
      vars.push({
        name: match[1],
        value: match[2].replace(/^"(.*)"$/, "$1"),
        description: comment.join(" "),
        commented: !live,
      });
      comment = [];
    } else if (line.startsWith("#")) comment.push(line.replace(/^#\s?/, ""));
    else comment = [];
  }
  return vars;
}

export function envPage(text: string): GeneratedPage {
  const vars = envVars(text);
  const rows = vars.map((v) => [
    code(v.name),
    v.value ? code(v.value) : "empty",
    editionOf(v.name),
    `${v.commented ? "Optional, commented out in the example. " : ""}${prose(v.description)}`.trim() ||
      "No comment in the example file.",
  ]);
  return {
    slug: "operate/environment",
    title: "Environment variables",
    description: "Every setting of .env.example with its default, the edition it belongs to and what it does.",
    source: ".env.example",
    body: `${vars.length} variables. Edition says where a variable matters: \`personal\`, \`saas\` or \`both\`. Copy \`.env.example\` to \`.env\` and set what you need; the example file holds placeholders only.\n\n${table(["Variable", "Example value", "Edition", "What it does"], rows)}\n`,
  };
}

// ---------------------------------------------------------------------------------------------- Shortcuts

export function shortcutsPage(markdown: string): GeneratedPage {
  return {
    slug: "use/shortcuts",
    title: "Keyboard shortcuts",
    description: "Every keyboard shortcut of vukem-finances and the rules that keep them from firing while you type.",
    source: "docs/shortcuts.md",
    body: markdown.replace(/\r\n/g, "\n").replace(/^# .*\n+/, ""),
  };
}

// ---------------------------------------------------------------------------------------------- Changelog

export function changelogPages(markdown: string): GeneratedPage[] {
  const releases = markdown
    .replace(/\r\n/g, "\n")
    .split(/^## /m)
    .slice(1)
    .filter((s) => /^\[\d/.test(s));
  const pages: GeneratedPage[] = [];
  const index: string[] = [];
  releases.forEach((section, i) => {
    const head = section.match(/^\[(\d+\.\d+\.\d+)\]\s*-\s*(\S+)/);
    if (!head) return;
    const [, version, date] = head;
    index.push(`- [${version}](/docs/changelog/v${version}/), ${date}`);
    pages.push({
      slug: `changelog/v${version}`,
      title: `Release ${version}`,
      description: `What changed in vukem-finances ${version}, released ${date}.`,
      order: i,
      source: "CHANGELOG.md",
      body: `Released ${date}.\n\n${section.split("\n").slice(1).join("\n").replace(/^### /gm, "## ")}`,
    });
  });
  pages.unshift({
    slug: "changelog",
    title: "Changelog",
    description: "The releases of vukem-finances and what each one changed, generated from the project changelog.",
    source: "CHANGELOG.md",
    body: `Each release has its own page. The format follows Keep a Changelog and the versions follow Semantic Versioning; the notes are written from the commit history when a release is cut.\n\n## Releases\n\n${index.join("\n")}\n`,
  });
  return pages;
}

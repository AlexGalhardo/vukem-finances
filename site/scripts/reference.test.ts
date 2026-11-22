import { describe, expect, test } from "bun:test";
import fs from "node:fs";
import path from "node:path";
import {
  changelogPages,
  cliCommands,
  cliPages,
  editionOf,
  envPage,
  envVars,
  type InventoryOperation,
  mcpPages,
  mcpResourceNames,
  mcpTools,
  type OpenAPI,
  pageFile,
  restPages,
  schemaName,
  typeOf,
} from "../lib/reference.ts";
import { generatedDir, sources } from "../lib/reference-config.ts";
import { generateAll } from "./generate-reference.ts";

const read = (file: string): string => fs.readFileSync(file, "utf8");
const openapi = JSON.parse(read(sources.openapi)) as OpenAPI;
const inventory = (JSON.parse(read(sources.operations)) as { operations: InventoryOperation[] }).operations;
const generated = generateAll();
const bySlugPrefix = (prefix: string): string =>
  generated
    .filter((p) => p.slug.startsWith(prefix))
    .map((p) => p.body)
    .join("\n");

describe("the real sources", () => {
  test("the REST reference has one section per operation of docs/openapi.json", () => {
    const operations = Object.values(openapi.paths).flatMap((item) =>
      Object.entries(item).filter(([m]) => ["get", "post", "patch", "put", "delete"].includes(m)),
    );
    const sections = bySlugPrefix("integrate/rest/").match(/^## `(GET|POST|PATCH|PUT|DELETE) \//gm) ?? [];
    expect(operations.length).toBeGreaterThan(50);
    expect(sections.length).toBe(operations.length);
    for (const [p, item] of Object.entries(openapi.paths)) {
      for (const method of Object.keys(item)) {
        expect(bySlugPrefix("integrate/rest/")).toContain(`## \`${method.toUpperCase()} ${p}\``);
      }
    }
  });

  test("every operation names its scope and carries a curl example with a bearer token", () => {
    const body = bySlugPrefix("integrate/rest/");
    expect((body.match(/^### Example$/gm) ?? []).length).toBe((body.match(/^## `/gm) ?? []).length);
    expect(body).toContain('-H "Authorization: Bearer vf_..."');
    expect(body).toContain("`accounts:write`");
  });

  test("the CLI reference has a section for every command", () => {
    const commands = cliCommands(read(sources.cli));
    expect(commands.size).toBeGreaterThan(40);
    const body = bySlugPrefix("integrate/cli/");
    for (const command of commands) expect(body).toContain(`## \`${command}\``);
  });

  test("the MCP reference has a section for every tool, resource, template and prompt", () => {
    const text = read(sources.mcp);
    const tools = mcpTools(text);
    expect(tools.length).toBeGreaterThan(40);
    const body = bySlugPrefix("integrate/mcp/");
    for (const tool of tools) expect(body).toContain(`## \`${tool.tool}\``);
    for (const name of mcpResourceNames(text)) expect(body).toContain(`\`${name}\``);
    // The tool operations are the inventory's, so a renamed operation fails here.
    const names = new Set(inventory.map((o) => o.name));
    for (const tool of tools) expect(names.has(tool.operation), tool.operation).toBe(true);
  });

  test("the environment page lists every variable of .env.example with its edition", () => {
    const vars = envVars(read(sources.env));
    const body = generated.find((p) => p.slug === "operate/environment")?.body ?? "";
    expect(vars.length).toBeGreaterThan(30);
    for (const v of vars) expect(body).toContain(`\`${v.name}\``);
    expect(editionOf("STRIPE_KEY")).toBe("saas");
    expect(editionOf("SETUP_TOKEN")).toBe("personal");
    expect(editionOf("APP_NAME")).toBe("both");
  });

  test("the shortcuts and changelog pages come from their sources", () => {
    expect(generated.find((p) => p.slug === "use/shortcuts")?.body).toContain("`mod+k`");
    const releases = generated.filter((p) => p.slug.startsWith("changelog/v"));
    expect(releases.length).toBeGreaterThan(0);
    expect(generated.find((p) => p.slug === "changelog")?.body).toContain("/docs/changelog/v0.5.0/");
  });

  test("regenerating leaves no difference with content/generated", () => {
    const expected = new Map(generated.map((p) => [`${p.slug}.md`, pageFile(p)]));
    for (const [file, content] of expected) {
      const full = path.join(generatedDir, file);
      expect(fs.existsSync(full), file).toBe(true);
      expect(read(full).replace(/\r\n/g, "\n"), file).toBe(content);
    }
  });

  test("every generated page starts with the do-not-edit banner", () => {
    for (const page of generated) expect(pageFile(page)).toContain("> Generated from `");
  });
});

describe("the generators on small fixtures", () => {
  const api: OpenAPI = {
    paths: {
      "/things/{uid}": {
        patch: {
          operationId: "things.update",
          summary: "Change a thing",
          tags: ["Things"],
          parameters: [{ name: "uid", in: "path", required: true, schema: { type: "string" } }],
          requestBody: {
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["name"],
                  properties: { name: { type: "string", examples: ["Rent"] } },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "OK",
              content: { "application/json": { schema: { $ref: "#/components/schemas/ThingOut" } } },
            },
            "404": { $ref: "#/components/responses/Problem" },
          },
          "x-required-scope": "things:write",
        },
      },
    },
    components: {
      schemas: {
        ThingOut: {
          type: "object",
          required: ["uid"],
          properties: { uid: { type: "string" }, n: { type: ["integer", "null"] } },
        },
      },
      responses: { Problem: { description: "Error (RFC 9457 problem details)" } },
    },
  };
  const inv: InventoryOperation[] = [
    {
      name: "things.update",
      summary: "Change a thing",
      scope: "things:write",
      kind: "write",
      idempotent: true,
      editions: ["personal"],
      surfaces: [],
      input: [],
    },
  ];

  test("REST: one page per tag with method, path, scope, flags, schemas and a curl example", () => {
    const [page] = restPages(api, inv);
    expect(page.slug).toBe("integrate/rest/things");
    expect(page.body).toContain("## `PATCH /things/{uid}`");
    expect(page.body).toContain("`things:write`");
    expect(page.body).toContain("honored");
    expect(page.body).toContain("| `name` | `string` | yes |");
    expect(page.body).toContain("| 404 | Error (RFC 9457 problem details) | problem+json | `not_found` |");
    expect(page.body).toContain("### Fields of `ThingOut`");
    expect(page.body).toContain(`curl -X PATCH -H "Authorization: Bearer vf_..." \\`);
    expect(page.body).toContain(`-d '{"name":"Rent"}'`);
  });

  test("schema names and types read well", () => {
    expect(schemaName("#/components/schemas/Envelope_github.com_x_app_api.AccountOut")).toBe("Envelope<AccountOut>");
    expect(typeOf({ type: ["integer", "null"], format: "int64" })).toBe("integer (int64)");
    expect(typeOf({ type: "array", items: { $ref: "#/components/schemas/ThingOut" } })).toBe("array of ThingOut");
  });

  test("CLI: an overview and one page per resource", () => {
    const md =
      "# CLI reference\n\n<!-- x -->\n\nIntro.\n\n## Modes\n\nm\n\n## things\n\n### `things list`\n\nList.\n\n```sh\nthings list [flags]\n```\n";
    const pages = cliPages(md);
    expect(pages.map((p) => p.slug)).toEqual(["integrate/cli/overview", "integrate/cli/things"]);
    expect(pages[1].body).toContain("## `things list`");
    expect([...cliCommands(md)]).toEqual(["things list"]);
  });

  test("MCP: a section per tool with its input", () => {
    const md =
      "## Tools\n\n| Tool | Operation | Kind | Scope | Toolsets |\n| --- | --- | --- | --- | --- |\n| `things_update` | `things.update` | write | `things:write` | all |\n";
    const [tools] = mcpPages(md, [{ ...inv[0], input: [{ name: "uid", type: "string", required: true }] }]);
    expect(tools.body).toContain("## `things_update`");
    expect(tools.body).toContain("`uid` (string, required)");
  });

  test("environment: comments above a variable, optional variables and editions", () => {
    const text = "# The port.\nPORT=3000\n\n# Optional.\n# MCP_TOOLSETS=read\nSTRIPE_KEY=\n";
    expect(envVars(text)).toEqual([
      { name: "PORT", value: "3000", description: "The port.", commented: false },
      { name: "MCP_TOOLSETS", value: "read", description: "Optional.", commented: true },
      { name: "STRIPE_KEY", value: "", description: "", commented: false },
    ]);
    expect(envPage(text).body).toContain("| `STRIPE_KEY` | empty | saas |");
  });

  test("changelog: one page per release", () => {
    const pages = changelogPages(
      "# Changelog\n\n## [Unreleased]\n\n## [1.2.3] - 2026-01-02\n\n### Added\n\n- A thing\n",
    );
    expect(pages.map((p) => p.slug)).toEqual(["changelog", "changelog/v1.2.3"]);
    expect(pages[1].body).toContain("## Added");
  });
});

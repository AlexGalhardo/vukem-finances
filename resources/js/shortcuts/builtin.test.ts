import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { navItemsFor } from "../nav";
import { builtinActionSpecs, navActionSpecs } from "./builtin";
import { normalizeBinding } from "./keys";
import { createRegistry } from "./registry";

interface Row {
  keys: string;
  id: string;
  owner: string;
}

function docsRows(): Row[] {
  const doc = readFileSync(join(import.meta.dir, "../../../docs/shortcuts.md"), "utf8");
  const rows: Row[] = [];
  for (const line of doc.split("\n")) {
    const match = line.match(/^\| `([^`]+)` \| `([^`]+)` \| ([^|]+) \|$/);
    if (match) rows.push({ keys: match[1] as string, id: match[2] as string, owner: (match[3] as string).trim() });
  }
  return rows;
}

describe("docs/shortcuts.md", () => {
  const rows = docsRows();

  test("lists every built-in action with the same keys", () => {
    expect(rows.length).toBeGreaterThan(0);
    for (const spec of builtinActionSpecs()) {
      if (!spec.keys) continue;
      const row = rows.find((r) => r.id === spec.id);
      expect(row, `${spec.id} is missing from docs/shortcuts.md`).toBeDefined();
      expect(normalizeBinding(row?.keys ?? "x x x")).toBe(normalizeBinding(spec.keys));
      expect(row?.owner).toBe("shell");
    }
  });

  test("lists no shell action that the code does not register", () => {
    const ids = new Set(builtinActionSpecs().map((s) => s.id));
    for (const row of rows.filter((r) => r.owner === "shell")) expect(ids.has(row.id), row.id).toBe(true);
  });

  test("gives every key to one action only", () => {
    const keys = rows.map((r) => normalizeBinding(r.keys));
    expect(new Set(keys).size).toBe(keys.length);
    const ids = rows.map((r) => r.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  test("the whole table is accepted by the registry (no collision, nothing reserved)", () => {
    const registry = createRegistry({ strict: true });
    for (const row of rows)
      registry.register({ id: row.id, labelKey: "dashboard", group: "actions", keys: row.keys, run() {} });
  });
});

describe("navigation by edition", () => {
  test("the personal edition has no billing; saas has it", () => {
    expect(navItemsFor("personal").map((i) => i.id)).not.toContain("billing");
    expect(navItemsFor("saas").map((i) => i.id)).toContain("billing");
    expect(navItemsFor(undefined).map((i) => i.id)).toContain("billing");
  });

  test("a navigation action exists per item, so the palette cannot offer billing in personal", () => {
    const personal = navActionSpecs(navItemsFor("personal")).map((s) => s.id);
    expect(personal).not.toContain("go-billing");
    expect(personal).toContain("go-transactions");
  });
});

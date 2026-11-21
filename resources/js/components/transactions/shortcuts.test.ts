import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// The keys QuickAdd.tsx registers through useAction; the allocation table of docs/shortcuts.md must list the same.
const registered = [
  { id: "new-expense", keys: "n e" },
  { id: "new-income", keys: "n i" },
  { id: "new-transfer", keys: "n t" },
];

describe("quick add shortcuts", () => {
  const doc = readFileSync(join(import.meta.dir, "../../../../docs/shortcuts.md"), "utf8");
  const source = readFileSync(join(import.meta.dir, "QuickAdd.tsx"), "utf8");

  for (const { id, keys } of registered) {
    test(`${id} is registered with ${keys} and allocated in docs/shortcuts.md`, () => {
      expect(source).toContain(`id: "${id}"`);
      expect(source).toContain(`keys: "${keys}"`);
      expect(doc).toContain(`| \`${keys}\` | \`${id}\` | epic 07 |`);
    });
  }
});

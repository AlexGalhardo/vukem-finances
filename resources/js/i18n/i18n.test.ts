import { describe, expect, test } from "bun:test";
import { readdirSync } from "node:fs";
import { join } from "node:path";
import { buildDictionaries, type DictionaryModules, findMissingKeys, locales } from "./load";

async function loadAll(): Promise<DictionaryModules> {
  const modules: DictionaryModules = {};
  for (const locale of locales) {
    for (const file of readdirSync(join(import.meta.dir, locale))) {
      if (file.endsWith(".ts")) modules[`./${locale}/${file}`] = await import(`./${locale}/${file}`);
    }
  }
  return modules;
}

describe("i18n parity", () => {
  test("every locale defines every key of en, domain by domain", async () => {
    const modules = await loadAll();
    const domains = (locale: string) =>
      Object.keys(modules)
        .filter((p) => p.startsWith(`./${locale}/`))
        .map((p) => p.split("/")[2]);
    for (const locale of locales) expect(domains(locale).sort()).toEqual(domains("en").sort());
    expect(findMissingKeys(buildDictionaries(modules))).toEqual({});
  });

  test("a locale that misses a key is reported", () => {
    const dictionaries = buildDictionaries({
      "./en/common.ts": { default: { a: "A", b: "B" } },
      "./pt-BR/common.ts": { default: { a: "A" } },
      "./es/common.ts": { default: { a: "A", b: "B" } },
    });
    expect(findMissingKeys(dictionaries)).toEqual({ "pt-BR": ["b"] });
  });

  test("a duplicate key across domain files is refused", () => {
    expect(() =>
      buildDictionaries({
        "./en/common.ts": { default: { a: "A" } },
        "./en/shell.ts": { default: { a: "A" } },
      }),
    ).toThrow("duplicate");
  });

  test("no translation is left empty", async () => {
    const dictionaries = buildDictionaries(await loadAll());
    for (const locale of locales)
      for (const [key, value] of Object.entries(dictionaries[locale])) expect(value, `${locale}.${key}`).not.toBe("");
  });
});

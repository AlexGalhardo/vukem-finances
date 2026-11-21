export const locales = ["en", "pt-BR", "es"] as const;
export type Locale = (typeof locales)[number];
export type Dictionary = Record<string, string>;
export type DictionaryModules = Record<string, { default: Dictionary }>;

/** Merges `./<locale>/<domain>.ts` modules (the shape of `import.meta.glob`) into one dictionary per locale. */
export function buildDictionaries(modules: DictionaryModules): Record<Locale, Dictionary> {
  const result: Record<Locale, Dictionary> = { en: {}, "pt-BR": {}, es: {} };
  for (const [path, module] of Object.entries(modules)) {
    const locale = path.split("/").slice(-2, -1)[0];
    if (!isLocale(locale)) throw new Error(`i18n: ${path} is not inside a locale directory (${locales.join(", ")})`);
    for (const [key, value] of Object.entries(module.default)) {
      if (key in result[locale]) throw new Error(`i18n: duplicate key "${key}" in ${path}`);
      result[locale][key] = value;
    }
  }
  return result;
}

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && (locales as readonly string[]).includes(value);
}

/** Keys that `en` defines and another locale lacks, per locale. */
export function findMissingKeys(dictionaries: Record<Locale, Dictionary>): Partial<Record<Locale, string[]>> {
  const missing: Partial<Record<Locale, string[]>> = {};
  for (const locale of locales) {
    if (locale === "en") continue;
    const absent = Object.keys(dictionaries.en).filter((key) => !(key in dictionaries[locale]));
    if (absent.length > 0) missing[locale] = absent;
  }
  return missing;
}

import en from "../src/landing/copy/en.json";
import es from "../src/landing/copy/es.json";
import ptBr from "../src/landing/copy/pt-br.json";
import type { Locale } from "./site.ts";

export type Copy = typeof en;

export const catalogs: Record<Locale, Copy> = { en, "pt-br": ptBr, es };

/** Flattens a catalog to `a.b.0.c` keys, so catalogs can be compared and scanned. */
export function flatten(value: unknown, prefix = ""): Record<string, string> {
  if (typeof value === "string") return { [prefix]: value };
  if (value === null || typeof value !== "object") return { [prefix]: String(value) };
  const out: Record<string, string> = {};
  for (const [key, child] of Object.entries(value)) {
    Object.assign(out, flatten(child, prefix ? `${prefix}.${key}` : key));
  }
  return out;
}

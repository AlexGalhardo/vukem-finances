import { translate, type Locale } from "./index";

/** [singular, plural] of the main and the minor unit of a currency, per language. */
type Units = { major: readonly [string, string]; minor: readonly [string, string] };

const UNITS: Record<Locale, Record<string, Units>> = {
  en: {
    USD: { major: ["dollar", "dollars"], minor: ["cent", "cents"] },
    EUR: { major: ["euro", "euros"], minor: ["cent", "cents"] },
    GBP: { major: ["pound", "pounds"], minor: ["penny", "pence"] },
    BRL: { major: ["real", "reais"], minor: ["centavo", "centavos"] },
  },
  "pt-BR": {
    USD: { major: ["dólar", "dólares"], minor: ["centavo", "centavos"] },
    EUR: { major: ["euro", "euros"], minor: ["centavo", "centavos"] },
    GBP: { major: ["libra", "libras"], minor: ["pêni", "pence"] },
    BRL: { major: ["real", "reais"], minor: ["centavo", "centavos"] },
  },
  es: {
    USD: { major: ["dólar", "dólares"], minor: ["centavo", "centavos"] },
    EUR: { major: ["euro", "euros"], minor: ["céntimo", "céntimos"] },
    GBP: { major: ["libra", "libras"], minor: ["penique", "peniques"] },
    BRL: { major: ["real", "reales"], minor: ["centavo", "centavos"] },
  },
};

const GENERIC: Record<Locale, Units> = {
  en: { major: ["unit", "units"], minor: ["cent", "cents"] },
  "pt-BR": { major: ["unidade", "unidades"], minor: ["centavo", "centavos"] },
  es: { major: ["unidad", "unidades"], minor: ["centavo", "centavos"] },
};

function unit(pair: readonly [string, string], n: number): string {
  return n === 1 ? pair[0] : pair[1];
}

/**
 * Integer cents as words for a screen reader: "minus 12 dollars 50 cents".
 * A zero part is left out ("12 dollars", "50 cents"); zero is "0 dollars".
 * Never touches a float: the parts come from integer division.
 */
export function spokenMoney(cents: number, currency: string, locale: Locale): string {
  if (!Number.isSafeInteger(cents)) {
    throw new RangeError("money must be a safe integer of cents");
  }
  const units = UNITS[locale][currency] ?? { ...GENERIC[locale], major: [currency, currency] as const };
  const mag = Math.abs(cents);
  const major = Math.floor(mag / 100);
  const minor = mag % 100;
  const sign = cents < 0 ? translate(locale, "spokenMinus") : "";

  const parts: string[] = [];
  if (major > 0 || minor === 0) {
    parts.push(`${major} ${unit(units.major, major)}`);
  }
  if (minor > 0) {
    parts.push(`${minor} ${unit(units.minor, minor)}`);
  }
  return `${sign}${parts.join(" ")}`;
}

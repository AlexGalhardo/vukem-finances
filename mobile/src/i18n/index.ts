import en, { type MessageKey } from "./en";
import es from "./es";
import ptBR from "./pt-BR";

export type { MessageKey };
export type Locale = "en" | "pt-BR" | "es";

export const LOCALES: readonly Locale[] = ["en", "pt-BR", "es"];

/** The name of each language in itself, for the language picker. */
export const LOCALE_NAMES: Record<Locale, string> = { en: "English", "pt-BR": "Português (Brasil)", es: "Español" };

export const catalogs: Record<Locale, Record<MessageKey, string>> = { en, "pt-BR": ptBR, es };

/** Picks the supported language for a BCP 47 tag from the device ("pt-PT" gives pt-BR, "es-MX" gives es), English otherwise. */
export function resolveLocale(tag: string | null | undefined): Locale {
	const language = (tag ?? "").toLowerCase().split(/[-_]/u)[0];
	if (language === "pt") {
		return "pt-BR";
	}
	if (language === "es") {
		return "es";
	}
	return "en";
}

export type Variables = Record<string, string | number>;

/** Looks up a message and fills its {placeholders}; an unknown placeholder is left as written. */
export function translate(locale: Locale, key: MessageKey, variables: Variables = {}): string {
	const template = catalogs[locale][key];
	return template.replace(/\{(\w+)\}/gu, (whole, name: string) => {
		const value = variables[name];
		return value === undefined ? whole : String(value);
	});
}

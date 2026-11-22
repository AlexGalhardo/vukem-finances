import { catalogs, LOCALES, type MessageKey, resolveLocale, translate } from "../../src/i18n";

const keysOf = (locale: keyof typeof catalogs): string[] => Object.keys(catalogs[locale]).sort();
const placeholders = (text: string): string[] => [...text.matchAll(/\{(\w+)\}/gu)].map((m) => m[1] ?? "").sort();

describe("catalog parity", () => {
	it.each(LOCALES.filter((l) => l !== "en"))("%s has exactly the keys of en", (locale) => {
		expect(keysOf(locale)).toEqual(keysOf("en"));
	});

	it.each(LOCALES.filter((l) => l !== "en"))("%s keeps the placeholders of every en message", (locale) => {
		for (const key of Object.keys(catalogs.en) as MessageKey[]) {
			expect([key, placeholders(catalogs[locale][key])]).toEqual([key, placeholders(catalogs.en[key])]);
		}
	});

	it.each(LOCALES)("%s has no empty message", (locale) => {
		for (const [key, text] of Object.entries(catalogs[locale])) {
			expect([key, text.trim().length > 0]).toEqual([key, true]);
		}
	});

	it("translates other languages for real (a message is not left in English by accident)", () => {
		const same = (Object.keys(catalogs.en) as MessageKey[]).filter(
			(k) => catalogs["pt-BR"][k] === catalogs.en[k] && catalogs.es[k] === catalogs.en[k],
		);
		// Names that are the same word in the three languages are fine; a long sentence is not.
		expect(same.filter((k) => catalogs.en[k].replace(/\{\w+\}/gu, "").trim().length > 20)).toEqual([]);
	});
});

describe("resolveLocale", () => {
	it("maps device tags to the supported languages", () => {
		expect(resolveLocale("pt-BR")).toBe("pt-BR");
		expect(resolveLocale("pt_PT")).toBe("pt-BR");
		expect(resolveLocale("es-MX")).toBe("es");
		expect(resolveLocale("en-GB")).toBe("en");
		expect(resolveLocale("ja-JP")).toBe("en");
		expect(resolveLocale(null)).toBe("en");
	});
});

describe("translate", () => {
	it("fills placeholders in every language", () => {
		expect(translate("en", "settingsVersion", { version: "0.1.0" })).toBe("Version 0.1.0");
		expect(translate("pt-BR", "cardStatementDue", { date: "10/12" })).toBe("Vence em 10/12");
		expect(translate("es", "budgetPercent", { percent: 88 })).toBe("88 por ciento usado");
	});

	it("leaves an unknown placeholder visible", () => {
		expect(translate("en", "budgetSpent", { spent: "$1" })).toBe("Spent $1 of {limit}");
	});
});

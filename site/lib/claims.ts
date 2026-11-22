import fs from "node:fs";
import path from "node:path";
import { catalogs, flatten } from "./copy.ts";

// Phrases that would imply a hosted service, a sign-up, a purchase or a demo. Matched on word boundaries, so
// "buyer" does not trip "buy".
export const forbiddenPhrases: Record<string, string[]> = {
	en: ["sign up", "signup", "start free", "log in", "login", "subscribe now", "buy", "get started free", "hosted for you", "try the demo", "contact us"],
	"pt-br": ["cadastre-se", "comece grátis", "faça login", "entrar na sua conta", "assine agora", "comprar", "compre agora", "hospedado para você", "experimente a demo", "fale conosco"],
	es: ["regístrate", "empieza gratis", "inicia sesión", "suscríbete ahora", "comprar", "compra ahora", "alojado para ti", "prueba la demo", "contáctanos"],
};

const letters = String.raw`[\p{L}\p{N}]`;

export function findForbidden(text: string, locale?: string): string[] {
	const found: string[] = [];
	for (const phrases of locale ? [forbiddenPhrases[locale] ?? []] : Object.values(forbiddenPhrases)) {
		for (const phrase of phrases) {
			const escaped = phrase.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`).replace(/ /g, String.raw`\s+`);
			if (new RegExp(`(?<!${letters})${escaped}(?!${letters})`, "iu").test(text)) found.push(phrase);
		}
	}
	return found;
}

export interface Claim {
	claim: string;
	where: string;
	evidence: string[];
	how: string;
}

/** Reads the table of the "Claims register" section of the brief. */
export function parseRegister(markdown: string): Claim[] {
	const section = markdown.split(/^## Claims register$/m)[1];
	if (!section) throw new Error("docs/landing-brief.md has no '## Claims register' section");
	const rows = section
		.split("\n")
		.filter((line) => line.trim().startsWith("|"))
		.slice(2);
	return rows.map((line) => {
		const cells = line.trim().replace(/^\||\|$/g, "").split("|").map((c) => c.trim());
		if (cells.length !== 4) throw new Error(`register row must have 4 cells: ${line}`);
		const [claim, where, evidence, how] = cells;
		return {
			claim,
			where: where.replace(/`/g, ""),
			evidence: [...evidence.matchAll(/`([^`]+)`/g)].map((m) => m[1]),
			how,
		};
	});
}

/** Problems of the register: a row without evidence, a missing evidence path, or a `where` key not in the catalog. */
export function checkRegister(markdown: string, repoRoot: string): string[] {
	const problems: string[] = [];
	const keys = Object.keys(flatten(catalogs.en));
	for (const row of parseRegister(markdown)) {
		if (row.evidence.length === 0) problems.push(`no evidence path: ${row.claim}`);
		for (const file of row.evidence) {
			const clean = file.split("#")[0];
			if (!fs.existsSync(path.join(repoRoot, clean))) problems.push(`missing evidence ${file}: ${row.claim}`);
		}
		for (const key of row.where.split(",").map((k) => k.trim())) {
			if (!keys.some((k) => k === key || k.startsWith(`${key}.`))) problems.push(`unknown catalog key ${key}: ${row.claim}`);
		}
	}
	return problems;
}

/** Forbidden phrases in the catalogs, as `locale key: phrase`. */
export function checkCatalogs(): string[] {
	const problems: string[] = [];
	for (const [locale, catalog] of Object.entries(catalogs)) {
		for (const [key, value] of Object.entries(flatten(catalog))) {
			for (const phrase of findForbidden(value, locale)) problems.push(`${locale} ${key}: "${phrase}"`);
		}
	}
	return problems;
}

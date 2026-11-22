import type { Database } from "../db/database";
import { LOCALES, type Locale } from "../i18n";

export type ThemeSetting = "system" | "light" | "dark";

/** The preferences kept in the `meta` table. The app lock setting is not here: it lives in secure storage. */
export interface Settings {
	locale: Locale;
	currency: string;
	timezone: string;
	theme: ThemeSetting;
	onboarded: boolean;
	/** Keep the queue of changes for a future sync (default on). */
	keepSyncQueue: boolean;
}

export const CURRENCIES = ["USD", "EUR", "GBP", "BRL"] as const;

export function defaultSettings(deviceLocale: Locale, deviceTimezone: string): Settings {
	return {
		locale: deviceLocale,
		currency: deviceLocale === "pt-BR" ? "BRL" : "USD",
		timezone: deviceTimezone,
		theme: "system",
		onboarded: false,
		keepSyncQueue: true,
	};
}

export async function loadSettings(db: Database, defaults: Settings): Promise<Settings> {
	const rows = await db.all<{ key: string; value: string }>(
		"SELECT key, value FROM meta WHERE key IN ('locale', 'currency', 'timezone', 'theme', 'onboarded', 'keep_sync_queue')",
	);
	const stored = new Map(rows.map((r) => [r.key, r.value]));
	const locale = stored.get("locale");
	const theme = stored.get("theme");
	return {
		locale: LOCALES.find((l) => l === locale) ?? defaults.locale,
		currency: stored.get("currency") ?? defaults.currency,
		timezone: stored.get("timezone") ?? defaults.timezone,
		theme: theme === "light" || theme === "dark" || theme === "system" ? theme : defaults.theme,
		onboarded: stored.get("onboarded") === "1",
		keepSyncQueue: stored.get("keep_sync_queue") !== "0",
	};
}

export async function saveSettings(db: Database, patch: Partial<Settings>): Promise<void> {
	const entries: [string, string][] = [];
	if (patch.locale !== undefined) entries.push(["locale", patch.locale]);
	if (patch.currency !== undefined) entries.push(["currency", patch.currency]);
	if (patch.timezone !== undefined) entries.push(["timezone", patch.timezone]);
	if (patch.theme !== undefined) entries.push(["theme", patch.theme]);
	if (patch.onboarded !== undefined) entries.push(["onboarded", patch.onboarded ? "1" : "0"]);
	if (patch.keepSyncQueue !== undefined) entries.push(["keep_sync_queue", patch.keepSyncQueue ? "1" : "0"]);
	await db.transaction(async () => {
		for (const [key, value] of entries) {
			await db.run(
				"INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
				[key, value],
			);
		}
	});
}

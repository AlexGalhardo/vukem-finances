import { render } from "@testing-library/react-native";
import type { ReactElement } from "react";
import { defaultSettings, type Settings } from "../../src/core/settings";
import { AppProvider } from "../../src/core/state";
import { type Env, openEnv } from "./fixtures";

export const NOW = new Date(Date.UTC(2027, 2, 15, 15, 0, 0));

/** Renders a screen over an in-memory database, with the clock fixed at 2027-03-15 and UTC as the zone. */
export async function renderApp(
	screen: ReactElement,
	options: { settings?: Partial<Settings>; env?: Env } = {},
): Promise<Env> {
	const env = options.env ?? (await openEnv());
	const settings = { ...defaultSettings("en", "UTC"), onboarded: true, ...options.settings };
	await render(
		<AppProvider db={env.db} repo={env.repo} ledger={env.ledger} initialSettings={settings} now={() => NOW}>
			{screen}
		</AppProvider>,
	);
	return env;
}

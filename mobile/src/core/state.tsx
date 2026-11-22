import { createContext, type ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { type Scheme, UiProvider } from "../components/ui";
import type { Database } from "../db/database";
import type { Repository } from "../db/repository";
import { todayIn } from "../domain/time";
import type { Ledger } from "../domain/ledger";
import { type Settings, saveSettings } from "./settings";

export interface App {
	db: Database;
	repo: Repository;
	ledger: Ledger;
	settings: Settings;
	updateSettings: (patch: Partial<Settings>) => Promise<void>;
	/** Today in the workspace time zone. */
	today: () => string;
	/** Bumped after every write: screens list it as a dependency to reload. */
	version: number;
	/** Call after a write so the other screens reload. */
	changed: () => void;
}

const AppContext = createContext<App | null>(null);

export function useApp(): App {
	const app = useContext(AppContext);
	if (app === null) {
		throw new Error("useApp needs an AppProvider");
	}
	return app;
}

export function AppProvider(props: {
	db: Database;
	repo: Repository;
	ledger: Ledger;
	initialSettings: Settings;
	/** The scheme the system asks for; the theme setting overrides it. */
	systemScheme?: Scheme;
	now?: () => Date;
	children: ReactNode;
}): ReactNode {
	const { db, repo, ledger, initialSettings, systemScheme = "light", now, children } = props;
	const [settings, setSettings] = useState(initialSettings);
	const [version, setVersion] = useState(0);

	const updateSettings = useCallback(
		async (patch: Partial<Settings>): Promise<void> => {
			await saveSettings(db, patch);
			if (patch.keepSyncQueue !== undefined) repo.setKeepOutbox(patch.keepSyncQueue);
			setSettings((s) => ({ ...s, ...patch }));
		},
		[db, repo],
	);
	const changed = useCallback(() => setVersion((v) => v + 1), []);
	const timezone = settings.timezone;
	const today = useCallback(() => todayIn(timezone, now?.()), [timezone, now]);

	const value = useMemo<App>(
		() => ({ db, repo, ledger, settings, updateSettings, today, version, changed }),
		[db, repo, ledger, settings, updateSettings, today, version, changed],
	);
	const scheme: Scheme = settings.theme === "system" ? systemScheme : settings.theme;

	return (
		<AppContext.Provider value={value}>
			<UiProvider locale={settings.locale} scheme={scheme} currency={settings.currency}>
				{children}
			</UiProvider>
		</AppContext.Provider>
	);
}

/** Runs a loader now and again whenever the data changes; `undefined` until the first result. */
export function useLoad<T>(load: (app: App) => Promise<T>, deps: readonly unknown[] = []): T | undefined {
	const app = useApp();
	const [result, setResult] = useState<T | undefined>(undefined);
	const [failure, setFailure] = useState<unknown>(null);
	if (failure !== null) {
		// Surface a failed read to the nearest error boundary instead of leaving the screen empty.
		throw failure;
	}
	// biome-ignore lint/correctness/useExhaustiveDependencies: the caller lists the inputs of `load`
	useEffect(() => {
		let live = true;
		load(app).then(
			(r) => {
				if (live) setResult(r);
			},
			(error: unknown) => {
				if (live) setFailure(error);
			},
		);
		return () => {
			live = false;
		};
	}, [app.version, app.settings.timezone, ...deps]);
	return result;
}

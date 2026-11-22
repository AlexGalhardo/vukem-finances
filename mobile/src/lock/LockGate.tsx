import { type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { AppState, type AppStateStatus, Text, View } from "react-native";
import { Button } from "../components/Button";
import { useUi } from "../components/ui";
import { type LockPlatform, type LockStore, shouldRelock } from "./lock";

export interface AppStateSource {
	addEventListener(type: "change", listener: (state: AppStateStatus) => void): { remove: () => void };
}

/**
 * Hides the app behind the device lock when the setting is on: at open, and
 * again after 60 seconds in the background. While locked the children are not
 * mounted, so no data is drawn. It is an access gate, not disk encryption.
 */
export function LockGate(props: {
	store: LockStore;
	platform: LockPlatform;
	children: ReactNode;
	now?: () => number;
	appState?: AppStateSource;
}): ReactNode {
	const { store, platform, children, now = Date.now, appState = AppState } = props;
	const ui = useUi();
	// null while the setting is being read, so nothing flashes before the lock.
	const [locked, setLocked] = useState<boolean | null>(null);
	const [failed, setFailed] = useState(false);
	const backgroundedAt = useRef<number | null>(null);

	useEffect(() => {
		let live = true;
		store.isEnabled().then((enabled) => {
			if (live) setLocked(enabled);
		});
		return () => {
			live = false;
		};
	}, [store]);

	useEffect(() => {
		const sub = appState.addEventListener("change", (state) => {
			if (state === "background") {
				backgroundedAt.current = now();
				return;
			}
			if (state === "active") {
				const since = backgroundedAt.current;
				backgroundedAt.current = null;
				if (shouldRelock(since, now())) {
					store.isEnabled().then((enabled) => {
						if (enabled) setLocked(true);
					});
				}
			}
		});
		return () => sub.remove();
	}, [appState, now, store]);

	const unlock = useCallback(async (): Promise<void> => {
		setFailed(false);
		const ok = await platform.authenticate(ui.t("lockPrompt"));
		if (ok) {
			setLocked(false);
		} else {
			setFailed(true);
		}
	}, [platform, ui]);

	// Ask as soon as the lock appears, so the person does not have to find the button.
	useEffect(() => {
		if (locked === true) {
			void unlock();
		}
	}, [locked, unlock]);

	if (locked === null) {
		return null;
	}
	if (!locked) {
		return children;
	}
	return (
		<View style={{ flex: 1, justifyContent: "center", padding: ui.spacing[6], gap: ui.spacing[4], backgroundColor: ui.colors.bg }}>
			<Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.xl, textAlign: "center" }}>
				{ui.t("appName")}
			</Text>
			{failed ? (
				<Text accessibilityRole="alert" style={{ color: ui.colors.expense, textAlign: "center" }}>
					{ui.t("errorGeneric")}
				</Text>
			) : null}
			<Button label={ui.t("lockButton")} onPress={() => void unlock()} />
		</View>
	);
}

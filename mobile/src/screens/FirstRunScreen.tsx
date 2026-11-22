import { type ReactNode, useState } from "react";
import { ScrollView, Text, View } from "react-native";
import { CURRENCIES } from "../core/settings";
import { useApp } from "../core/state";
import { AmountField } from "../components/AmountField";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { ScreenHeader } from "../components/ScreenHeader";
import { TextField } from "../components/TextField";
import { useUi } from "../components/ui";
import { DEFAULT_CATEGORIES } from "../domain/defaults";
import { LOCALE_NAMES, LOCALES, type Locale, type MessageKey, translate } from "../i18n";
import { parseAmount } from "../domain/money";
import { isValidTimezone } from "../domain/time";

const ACCOUNT_KINDS: { value: string; label: MessageKey }[] = [
	{ value: "checking", label: "accountKindChecking" },
	{ value: "savings", label: "accountKindSavings" },
	{ value: "cash", label: "accountKindCash" },
	{ value: "wallet", label: "accountKindWallet" },
];

/**
 * First run: language, currency, time zone, a first account and, if wanted,
 * the default categories. Nothing leaves the phone and no account is needed.
 */
export function FirstRunScreen(props: { onFinished?: () => void }): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [locale, setLocale] = useState<Locale>(app.settings.locale);
	const [currency, setCurrency] = useState(app.settings.currency);
	const [timezone, setTimezone] = useState(app.settings.timezone);
	const [name, setName] = useState("");
	const [kind, setKind] = useState("checking");
	const [opening, setOpening] = useState("");
	const [withCategories, setWithCategories] = useState(true);
	const [error, setError] = useState<string | null>(null);

	// The text follows the language picked here, before it is saved.
	const t = (key: MessageKey): string => translate(locale, key);

	async function start(): Promise<void> {
		const cents = opening.trim() === "" ? 0 : parseAmount(opening);
		if (name.trim() === "") {
			setError(t("required"));
			return;
		}
		if (cents === null) {
			setError(t("invalidAmount"));
			return;
		}
		if (!isValidTimezone(timezone)) {
			setError(t("errorGeneric"));
			return;
		}
		setError(null);

		await app.db.transaction(async () => {
			await app.repo.create("account", { name: name.trim(), kind, opening_cents: cents, archived_at: null, position: 0 });
			if (withCategories) {
				for (const c of DEFAULT_CATEGORIES) {
					await app.ledger.createCategory({ name: c.names[locale], kind: c.kind, parent_uid: null, color: c.color, icon: "" });
				}
			}
		});
		await app.updateSettings({ locale, currency, timezone, onboarded: true });
		app.changed();
		props.onFinished?.();
	}

	return (
		<ScrollView contentContainerStyle={{ padding: ui.spacing[4], gap: ui.spacing[4] }} keyboardShouldPersistTaps="handled">
			<ScreenHeader title={t("firstRunWelcome")} />
			<Text style={{ color: ui.colors.text, fontSize: ui.fontSize.base }}>{t("firstRunIntro")}</Text>

			<Text style={{ color: ui.colors.textMuted }}>{t("firstRunLanguage")}</Text>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				{LOCALES.map((l) => (
					<Chip key={l} role="radio" label={LOCALE_NAMES[l]} selected={locale === l} onPress={() => setLocale(l)} />
				))}
			</View>

			<Text style={{ color: ui.colors.textMuted }}>{t("firstRunCurrency")}</Text>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				{CURRENCIES.map((c) => (
					<Chip key={c} role="radio" label={c} selected={currency === c} onPress={() => setCurrency(c)} />
				))}
			</View>

			<TextField label={t("firstRunTimezone")} value={timezone} onChangeText={setTimezone} autoCapitalize="none" />
			<TextField label={t("firstRunAccountName")} value={name} onChangeText={setName} />

			<Text style={{ color: ui.colors.textMuted }}>{t("firstRunAccountKind")}</Text>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				{ACCOUNT_KINDS.map((k) => (
					<Chip key={k.value} role="radio" label={t(k.label)} selected={kind === k.value} onPress={() => setKind(k.value)} />
				))}
			</View>

			<AmountField label={t("firstRunOpening")} value={opening} onChangeText={setOpening} />
			<Chip role="checkbox" label={t("firstRunCategories")} selected={withCategories} onPress={() => setWithCategories((v) => !v)} />

			{error === null ? null : (
				<Text accessibilityRole="alert" accessibilityLiveRegion="polite" style={{ color: ui.colors.expense }}>
					{error}
				</Text>
			)}
			<Button label={t("firstRunStart")} onPress={() => void start()} />
		</ScrollView>
	);
}

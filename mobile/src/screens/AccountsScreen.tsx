import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { describeError } from "../app/errors";
import { type App, useApp, useLoad } from "../app/state";
import { AmountField } from "../components/AmountField";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { DeleteButton } from "../components/DeleteButton";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { MoneyText } from "../components/MoneyText";
import { Page } from "../components/Page";
import { TextField } from "../components/TextField";
import { useUi } from "../components/ui";
import { accountBalances, type AccountBalance } from "../domain/ledger";
import { formatDecimalCents, parseAmount } from "../domain/money";
import type { MessageKey } from "../i18n";
import { AddScreen } from "./AddScreen";

const KINDS: { value: string; label: MessageKey }[] = [
	{ value: "checking", label: "accountKindChecking" },
	{ value: "savings", label: "accountKindSavings" },
	{ value: "cash", label: "accountKindCash" },
	{ value: "investment", label: "accountKindInvestment" },
	{ value: "wallet", label: "accountKindWallet" },
];

/** Accounts with their balances, create and edit, archive, delete (with its transactions) and the transfer form. */
export function AccountsScreen(): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [editing, setEditing] = useState<string | "new" | null>(null);
	const [transferring, setTransferring] = useState(false);
	const [name, setName] = useState("");
	const [kind, setKind] = useState("checking");
	const [opening, setOpening] = useState("");
	const [error, setError] = useState<string | null>(null);

	const today = app.today();
	const data = useLoad(
		async (a: App) => {
			const accounts = (await a.repo.list("account")).filter((x) => x.kind !== "credit_card");
			const balances = new Map<string, AccountBalance>((await accountBalances(a.db, a.today())).map((b) => [b.account_uid, b]));
			return { accounts, balances };
		},
		[today],
	);

	function open(uid: string | "new"): void {
		const account = data?.accounts.find((a) => a.uid === uid);
		setName(account?.name ?? "");
		setKind(account?.kind ?? "checking");
		setOpening(account === undefined ? "" : formatDecimalCents(account.opening_cents));
		setError(null);
		setEditing(uid);
	}

	async function save(): Promise<void> {
		const cents = opening.trim() === "" ? 0 : parseAmount(opening);
		if (name.trim() === "") {
			setError(ui.t("required"));
			return;
		}
		if (cents === null) {
			setError(ui.t("invalidAmount"));
			return;
		}
		try {
			if (editing === "new") {
				await app.repo.create("account", { name: name.trim(), kind, opening_cents: cents, archived_at: null, position: data?.accounts.length ?? 0 });
			} else if (editing !== null) {
				await app.repo.update("account", editing, { name: name.trim(), kind, opening_cents: cents });
			}
		} catch (e) {
			setError(describeError(ui, e));
			return;
		}
		setEditing(null);
		app.changed();
	}

	async function toggleArchive(uid: string, archived: boolean): Promise<void> {
		await app.repo.update("account", uid, { archived_at: archived ? null : new Date().toISOString() });
		setEditing(null);
		app.changed();
	}

	async function remove(uid: string): Promise<void> {
		await app.ledger.deleteAccount(uid);
		setEditing(null);
		app.changed();
	}

	if (transferring) {
		return <AddScreen initialKind="transfer" onDone={() => setTransferring(false)} />;
	}

	if (editing !== null) {
		const archived = data?.accounts.find((a) => a.uid === editing)?.archived_at != null;
		return (
			<Page title={editing === "new" ? ui.t("accountNew") : ui.t("accountEdit")} onBack={() => setEditing(null)}>
				<TextField label={ui.t("name")} value={name} onChangeText={setName} />
				<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
					{KINDS.map((k) => (
						<Chip key={k.value} role="radio" label={ui.t(k.label)} selected={kind === k.value} onPress={() => setKind(k.value)} />
					))}
				</View>
				<AmountField label={ui.t("accountOpening")} value={opening} onChangeText={setOpening} />
				{error === null ? null : (
					<Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
						{error}
					</Text>
				)}
				<Button label={ui.t("save")} onPress={() => void save()} />
				{editing === "new" ? null : (
					<>
						<Button variant="secondary" label={archived ? ui.t("unarchive") : ui.t("archive")} onPress={() => void toggleArchive(editing, archived)} />
						<DeleteButton onConfirm={() => void remove(editing)} />
					</>
				)}
			</Page>
		);
	}

	const accounts = data?.accounts ?? [];
	return (
		<Page title={ui.t("tabAccounts")} action={{ label: ui.t("add"), onPress: () => open("new") }}>
			{data === undefined ? null : accounts.length === 0 ? (
				<EmptyState message={ui.t("accountsEmpty")} action={{ label: ui.t("accountNew"), onPress: () => open("new") }} />
			) : (
				accounts.map((a) => {
					const balance = data.balances.get(a.uid)?.balance_cents ?? 0;
					return (
						<ListRow
							key={a.uid}
							title={a.archived_at === null ? a.name : `${a.name} (${ui.t("accountArchived")})`}
							subtitle={ui.t(KINDS.find((k) => k.value === a.kind)?.label ?? "accountKindChecking")}
							trailing={<MoneyText cents={balance} />}
							onPress={() => open(a.uid)}
						/>
					);
				})
			)}
			{accounts.length >= 2 ? <Button variant="secondary" label={ui.t("accountTransfer")} onPress={() => setTransferring(true)} /> : null}
		</Page>
	);
}

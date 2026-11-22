import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { describeError } from "../core/errors";
import { type App, useApp, useLoad } from "../core/state";
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
import { isIsoDate } from "../domain/dates";
import { formatDecimalCents, parseAmount } from "../domain/money";
import { createInstallments, type Frequency, generateDue, upcoming, validateRule } from "../domain/recurrence";
import type { MessageKey } from "../i18n";

const FREQUENCIES: { value: Frequency; label: MessageKey }[] = [
	{ value: "daily", label: "frequencyDaily" },
	{ value: "weekly", label: "frequencyWeekly" },
	{ value: "monthly", label: "frequencyMonthly" },
	{ value: "yearly", label: "frequencyYearly" },
];

type Mode = "list" | "rule" | "installment";

/**
 * Recurring transactions and what is upcoming (30 days), plus the installment
 * purchase form. Rules generate their transactions at app start and on
 * foreground; saving one also runs a generation so a rule that starts today shows up.
 */
export function RecurrencesScreen(props: { onBack?: () => void }): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [mode, setMode] = useState<Mode>("list");
	const [editing, setEditing] = useState<string | null>(null);
	const [kind, setKind] = useState("expense");
	const [accountUid, setAccountUid] = useState("");
	const [amount, setAmount] = useState("");
	const [description, setDescription] = useState("");
	const [frequency, setFrequency] = useState<Frequency>("monthly");
	const [every, setEvery] = useState("1");
	const [start, setStart] = useState(app.today());
	const [end, setEnd] = useState("");
	const [count, setCount] = useState("");
	const [parts, setParts] = useState("3");
	const [error, setError] = useState<string | null>(null);

	const today = app.today();
	const data = useLoad(
		async (a: App) => ({
			rules: await a.repo.list("recurrence"),
			accounts: (await a.repo.list("account")).filter((x) => x.archived_at === null),
			due: await upcoming(a.db, a.repo, a.today(), 30),
		}),
		[today],
	);
	const account = accountUid === "" ? (data?.accounts[0]?.uid ?? "") : accountUid;

	function reset(next: Mode): void {
		setError(null);
		setEditing(null);
		setAmount("");
		setDescription("");
		setEvery("1");
		setEnd("");
		setCount("");
		setParts("3");
		setStart(app.today());
		setMode(next);
	}

	function edit(uid: string): void {
		const r = data?.rules.find((x) => x.uid === uid);
		if (r === undefined) return;
		reset("rule");
		setEditing(uid);
		setKind(r.kind);
		setAccountUid(r.account_uid);
		setAmount(formatDecimalCents(r.amount_cents));
		setDescription(r.description);
		setFrequency(r.frequency as Frequency);
		setEvery(String(r.interval));
		setStart(r.start_date);
		setEnd(r.end_date ?? "");
		setCount(r.max_count === null ? "" : String(r.max_count));
	}

	async function saveRule(): Promise<void> {
		const cents = parseAmount(amount);
		if (cents === null || cents <= 0) {
			setError(ui.t("invalidAmount"));
			return;
		}
		if (!isIsoDate(start) || (end !== "" && !isIsoDate(end))) {
			setError(ui.t("invalidDate"));
			return;
		}
		const rule = {
			frequency,
			interval: Number(every),
			start_date: start,
			end_date: end === "" ? null : end,
			max_count: count === "" ? null : Number(count),
		};
		try {
			validateRule(rule);
			if (editing === null) {
				await app.repo.create("recurrence", {
					account_uid: account,
					category_uid: null,
					kind,
					amount_cents: cents,
					description: description.trim(),
					...rule,
					generated_through: null,
					active: true,
				});
			} else {
				await app.repo.update("recurrence", editing, {
					account_uid: account,
					kind,
					amount_cents: cents,
					description: description.trim(),
					...rule,
				});
			}
			await generateDue(app.db, app.repo, app.today());
		} catch (e) {
			setError(describeError(ui, e));
			return;
		}
		setMode("list");
		app.changed();
	}

	async function saveInstallments(): Promise<void> {
		const cents = parseAmount(amount);
		if (cents === null || cents <= 0) {
			setError(ui.t("invalidAmount"));
			return;
		}
		if (!isIsoDate(start)) {
			setError(ui.t("invalidDate"));
			return;
		}
		try {
			await createInstallments(app.db, app.repo, {
				account_uid: account,
				category_uid: null,
				total_cents: cents,
				parts: Number(parts),
				first_date: start,
				description: description.trim(),
			});
		} catch (e) {
			setError(describeError(ui, e));
			return;
		}
		setMode("list");
		app.changed();
	}

	async function toggleActive(uid: string, active: boolean): Promise<void> {
		await app.repo.update("recurrence", uid, { active: !active });
		setMode("list");
		app.changed();
	}

	async function remove(uid: string): Promise<void> {
		await app.repo.remove("recurrence", uid);
		setMode("list");
		app.changed();
	}

	const accountChips = (
		<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
			{(data?.accounts ?? []).map((a) => (
				<Chip key={a.uid} role="radio" label={a.name} selected={account === a.uid} onPress={() => setAccountUid(a.uid)} />
			))}
		</View>
	);
	const errorText =
		error === null ? null : (
			<Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
				{error}
			</Text>
		);

	if (mode === "rule") {
		const rule = editing === null ? undefined : data?.rules.find((r) => r.uid === editing);
		return (
			<Page title={ui.t("recurrenceNew")} onBack={() => setMode("list")}>
				<AmountField value={amount} onChangeText={setAmount} />
				<View style={{ flexDirection: "row", gap: ui.spacing[2] }}>
					<Chip role="radio" label={ui.t("expense")} selected={kind === "expense"} onPress={() => setKind("expense")} />
					<Chip role="radio" label={ui.t("income")} selected={kind === "income"} onPress={() => setKind("income")} />
				</View>
				{accountChips}
				<TextField label={ui.t("addDescription")} value={description} onChangeText={setDescription} />
				<Text style={{ color: ui.colors.textMuted }}>{ui.t("recurrenceFrequency")}</Text>
				<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
					{FREQUENCIES.map((f) => (
						<Chip key={f.value} role="radio" label={ui.t(f.label)} selected={frequency === f.value} onPress={() => setFrequency(f.value)} />
					))}
				</View>
				<TextField label={ui.t("recurrenceEvery")} value={every} onChangeText={setEvery} keyboardType="number-pad" />
				<TextField label={ui.t("recurrenceStart")} value={start} onChangeText={setStart} keyboardType="numbers-and-punctuation" />
				<TextField label={ui.t("recurrenceEnd")} value={end} onChangeText={setEnd} keyboardType="numbers-and-punctuation" />
				<TextField label={ui.t("recurrenceCount")} value={count} onChangeText={setCount} keyboardType="number-pad" />
				{errorText}
				<Button label={ui.t("save")} onPress={() => void saveRule()} />
				{rule === undefined ? null : (
					<>
						<Button
							variant="secondary"
							label={rule.active ? ui.t("archive") : ui.t("unarchive")}
							onPress={() => void toggleActive(rule.uid, rule.active)}
						/>
						<DeleteButton onConfirm={() => void remove(rule.uid)} />
					</>
				)}
			</Page>
		);
	}

	if (mode === "installment") {
		return (
			<Page title={ui.t("installmentNew")} onBack={() => setMode("list")}>
				<AmountField label={ui.t("installmentTotal")} value={amount} onChangeText={setAmount} />
				<TextField label={ui.t("installmentParts")} value={parts} onChangeText={setParts} keyboardType="number-pad" />
				{accountChips}
				<TextField label={ui.t("addDescription")} value={description} onChangeText={setDescription} />
				<TextField label={ui.t("date")} value={start} onChangeText={setStart} keyboardType="numbers-and-punctuation" />
				{errorText}
				<Button label={ui.t("save")} onPress={() => void saveInstallments()} />
			</Page>
		);
	}

	const rules = data?.rules ?? [];
	const due = data?.due ?? [];
	return (
		<Page title={ui.t("titleRecurrences")} onBack={props.onBack} action={{ label: ui.t("add"), onPress: () => reset("rule") }}>
			{data === undefined ? null : rules.length === 0 ? (
				<EmptyState message={ui.t("recurrencesEmpty")} action={{ label: ui.t("recurrenceNew"), onPress: () => reset("rule") }} />
			) : (
				rules.map((r) => (
					<ListRow
						key={r.uid}
						title={r.description !== "" ? r.description : ui.t(r.kind === "income" ? "income" : "expense")}
						subtitle={`${ui.t(FREQUENCIES.find((f) => f.value === r.frequency)?.label ?? "frequencyMonthly")} · ${r.start_date}${r.active ? "" : ` · ${ui.t("accountArchived")}`}`}
						trailing={<MoneyText cents={r.amount_cents} kind={r.kind === "income" ? "income" : "expense"} />}
						onPress={() => edit(r.uid)}
					/>
				))
			)}
			<Button variant="secondary" label={ui.t("installmentNew")} onPress={() => reset("installment")} />

			<Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
				{ui.t("titleUpcoming")}
			</Text>
			{due.length === 0 ? (
				<Text style={{ color: ui.colors.textMuted }}>{ui.t("upcomingEmpty")}</Text>
			) : (
				due.map((i) => (
					<ListRow
						key={`${i.date}-${i.transaction_uid ?? i.recurrence_uid ?? ""}`}
						title={i.description !== "" ? i.description : ui.t(i.kind === "income" ? "income" : "expense")}
						subtitle={i.installment_no === null ? i.date : `${i.date} · ${i.installment_no}/${i.installment_total}`}
						trailing={<MoneyText cents={i.amount_cents} kind={i.kind === "income" ? "income" : "expense"} />}
					/>
				))
			)}
		</Page>
	);
}

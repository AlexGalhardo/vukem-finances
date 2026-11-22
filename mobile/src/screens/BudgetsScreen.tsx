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
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import { type BudgetStatus, budgetStatuses, saveBudget } from "../domain/budgets";
import { formatDecimalCents, parseAmount } from "../domain/money";

/** The monthly budgets with what this month has spent; the bar always comes with the numbers and the percent in words. */
export function BudgetsScreen(props: { onBack?: () => void }): ReactNode {
	const app = useApp();
	const ui = useUi();
	const month = app.today().slice(0, 7);
	const [editing, setEditing] = useState<BudgetStatus | "new" | null>(null);
	const [category, setCategory] = useState<string | null>(null);
	const [limit, setLimit] = useState("");
	const [rollover, setRollover] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const data = useLoad(
		async (a: App) => ({
			statuses: await budgetStatuses(a.db, a.today().slice(0, 7)),
			categories: (await a.repo.list("category")).filter((c) => c.kind === "expense" && c.archived_at === null),
		}),
		[month],
	);

	function open(target: BudgetStatus | "new"): void {
		setCategory(target === "new" ? null : target.category_uid);
		setLimit(target === "new" ? "" : formatDecimalCents(target.limit_cents));
		setRollover(target === "new" ? false : target.rollover);
		setError(null);
		setEditing(target);
	}

	async function save(): Promise<void> {
		const cents = parseAmount(limit);
		if (cents === null || cents <= 0) {
			setError(ui.t("invalidAmount"));
			return;
		}
		try {
			await saveBudget(app.db, app.repo, { category_uid: category, limit_cents: cents, rollover });
		} catch (e) {
			setError(describeError(ui, e));
			return;
		}
		setEditing(null);
		app.changed();
	}

	async function remove(uid: string): Promise<void> {
		await app.repo.remove("budget", uid);
		setEditing(null);
		app.changed();
	}

	if (editing !== null) {
		const taken = new Set((data?.statuses ?? []).filter((s) => editing === "new" || s.uid !== editing.uid).map((s) => s.category_uid));
		return (
			<Page title={ui.t("budgetNew")} onBack={() => setEditing(null)}>
				<Text style={{ color: ui.colors.textMuted }}>{ui.t("category")}</Text>
				<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
					{editing === "new" && !taken.has(null) || (editing !== "new" && editing.overall) ? (
						<Chip role="radio" label={ui.t("budgetOverall")} selected={category === null} onPress={() => setCategory(null)} />
					) : null}
					{(data?.categories ?? [])
						.filter((c) => !taken.has(c.uid))
						.map((c) => (
							<Chip key={c.uid} role="radio" label={c.name} selected={category === c.uid} onPress={() => setCategory(c.uid)} />
						))}
				</View>
				<AmountField label={ui.t("budgetLimit")} value={limit} onChangeText={setLimit} />
				<Chip role="checkbox" label={ui.t("budgetRollover")} selected={rollover} onPress={() => setRollover((v) => !v)} />
				{error === null ? null : (
					<Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
						{error}
					</Text>
				)}
				<Button label={ui.t("save")} onPress={() => void save()} />
				{editing === "new" ? null : <DeleteButton onConfirm={() => void remove(editing.uid)} />}
			</Page>
		);
	}

	const statuses = data?.statuses ?? [];
	return (
		<Page title={ui.t("titleBudgets")} onBack={props.onBack} action={{ label: ui.t("add"), onPress: () => open("new") }}>
			{data === undefined ? null : statuses.length === 0 ? (
				<EmptyState message={ui.t("budgetsEmpty")} action={{ label: ui.t("budgetNew"), onPress: () => open("new") }} />
			) : (
				statuses.map((s) => {
					const title = s.overall ? ui.t("budgetOverall") : s.category;
					const summary = ui.t("budgetSpent", { spent: ui.money(s.spent_cents), limit: ui.money(s.available_cents) });
					const over = s.percent > 100;
					return (
						<View key={s.uid} style={{ gap: ui.spacing[1] }}>
							<ListRow title={title} subtitle={`${summary} · ${ui.t("budgetPercent", { percent: s.percent })}`} onPress={() => open(s)} />
							{s.carry_cents > 0 ? <Text style={{ color: ui.colors.textMuted }}>{ui.t("budgetCarry", { amount: ui.money(s.carry_cents) })}</Text> : null}
							<View style={{ height: 8, borderRadius: 4, backgroundColor: ui.colors.surfaceMuted }}>
								<View
									style={{
										width: `${Math.min(100, s.percent)}%`,
										height: 8,
										borderRadius: 4,
										backgroundColor: over ? ui.colors.expense : ui.colors.brand,
									}}
								/>
							</View>
						</View>
					);
				})
			)}
		</Page>
	);
}

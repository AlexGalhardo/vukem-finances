import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { type App, useApp, useLoad } from "../core/state";
import { Chip } from "../components/Chip";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { MoneyText } from "../components/MoneyText";
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import { byCategory, cashFlow, type CategoryRow, type PeriodPreset, resolvePeriod } from "../domain/reports";
import type { MessageKey } from "../i18n";

const PERIODS: { value: Exclude<PeriodPreset, "custom">; label: MessageKey }[] = [
	{ value: "this_month", label: "periodThisMonth" },
	{ value: "last_month", label: "periodLastMonth" },
	{ value: "last_3_months", label: "periodLast3Months" },
	{ value: "last_12_months", label: "periodLast12Months" },
	{ value: "this_year", label: "periodThisYear" },
];

/**
 * Cash flow and spending by category for a period. Each report has bars and,
 * one press away, a table with the same numbers; the bars are never the only
 * way to read a value (each carries its amount as text).
 */
export function ReportsScreen(props: { onBack?: () => void }): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [period, setPeriod] = useState<Exclude<PeriodPreset, "custom">>("this_month");
	const [report, setReport] = useState<"cash_flow" | "by_category">("cash_flow");
	const [table, setTable] = useState(false);
	const [kind, setKind] = useState<"expense" | "income">("expense");

	const today = app.today();
	const data = useLoad(
		async (a: App) => {
			const p = resolvePeriod(period, "", "", a.today());
			return { flow: await cashFlow(a.db, p), categories: await byCategory(a.db, p, kind) };
		},
		[period, kind, today],
	);

	/** A bar whose length is a share of the largest value, in a flexible row beside its numbers. */
	const bar = (cents: number, max: number, color: string): ReactNode => (
		<View style={{ flexDirection: "row", height: 8 }} importantForAccessibility="no-hide-descendants">
			<View style={{ flex: max === 0 ? 0 : Math.min(cents, max), backgroundColor: color, borderRadius: 4 }} />
			<View style={{ flex: Math.max(max - cents, 0) }} />
		</View>
	);

	const flowRows = data?.flow.months ?? [];
	const maxFlow = Math.max(0, ...flowRows.flatMap((m) => [m.income_cents, m.expense_cents]));
	const rows = data?.categories.rows ?? [];

	const categoryLine = (row: CategoryRow, depth: number): ReactNode => (
		<View key={`${row.uid}-${depth}`}>
			<ListRow
				title={`${depth > 0 ? "↳ " : ""}${row.uncategorized ? ui.t("uncategorized") : row.name}`}
				subtitle={`${ui.t("reportsShare")} ${(row.share_bps / 100).toFixed(2)}%`.replace(/\.00%$/u, "%")}
				trailing={<MoneyText cents={row.cents} kind={kind} />}
			/>
			{table ? null : bar(row.cents, data?.categories.total_cents ?? 0, kind === "income" ? ui.colors.income : ui.colors.expense)}
			{row.children.map((c) => categoryLine(c, depth + 1))}
		</View>
	);

	return (
		<Page title={ui.t("titleReports")} onBack={props.onBack}>
			<Text style={{ color: ui.colors.textMuted }}>{ui.t("reportsPeriod")}</Text>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				{PERIODS.map((p) => (
					<Chip key={p.value} role="radio" label={ui.t(p.label)} selected={period === p.value} onPress={() => setPeriod(p.value)} />
				))}
			</View>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				<Chip role="radio" label={ui.t("reportsCashFlow")} selected={report === "cash_flow"} onPress={() => setReport("cash_flow")} />
				<Chip role="radio" label={ui.t("reportsByCategory")} selected={report === "by_category"} onPress={() => setReport("by_category")} />
			</View>
			<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
				<Chip role="radio" label={ui.t("reportsChartView")} selected={!table} onPress={() => setTable(false)} />
				<Chip role="radio" label={ui.t("reportsTableView")} selected={table} onPress={() => setTable(true)} />
			</View>

			{data === undefined ? null : report === "cash_flow" ? (
				flowRows.length === 0 || (data.flow.totals.income_cents === 0 && data.flow.totals.expense_cents === 0) ? (
					<EmptyState message={ui.t("reportsEmpty")} />
				) : (
					<View accessibilityLabel={ui.t("reportsCashFlow")}>
						{flowRows.map((m) => (
							<View key={m.month}>
								<Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.base }}>
									{m.month}
								</Text>
								<ListRow title={ui.t("income")} trailing={<MoneyText cents={m.income_cents} kind="income" />} />
								{table ? null : bar(m.income_cents, maxFlow, ui.colors.income)}
								<ListRow title={ui.t("expense")} trailing={<MoneyText cents={m.expense_cents} kind="expense" />} />
								{table ? null : bar(m.expense_cents, maxFlow, ui.colors.expense)}
								<ListRow title={ui.t("net")} trailing={<MoneyText cents={m.net_cents} />} />
							</View>
						))}
						<ListRow title={ui.t("total")} trailing={<MoneyText cents={data.flow.totals.net_cents} />} />
					</View>
				)
			) : (
				<View accessibilityLabel={ui.t("reportsByCategory")}>
					<View style={{ flexDirection: "row", gap: ui.spacing[2] }}>
						<Chip role="radio" label={ui.t("expense")} selected={kind === "expense"} onPress={() => setKind("expense")} />
						<Chip role="radio" label={ui.t("income")} selected={kind === "income"} onPress={() => setKind("income")} />
					</View>
					{rows.length === 0 ? <EmptyState message={ui.t("reportsEmpty")} /> : rows.map((r) => categoryLine(r, 0))}
					{rows.length === 0 ? null : <ListRow title={ui.t("total")} trailing={<MoneyText cents={data.categories.total_cents} kind={kind} />} />}
				</View>
			)}
		</Page>
	);
}

import { type ReactNode, useState } from "react";
import { FlatList, ScrollView, Text, View } from "react-native";
import { type App, useApp, useLoad } from "../app/state";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { MoneyText } from "../components/MoneyText";
import { ScreenHeader } from "../components/ScreenHeader";
import { TextField } from "../components/TextField";
import { Toast } from "../components/Toast";
import { useUi } from "../components/ui";
import { listTransactions, type TransactionFilter, type TransactionListItem } from "../db/queries";
import { resolvePeriod } from "../domain/reports";
import { AddScreen } from "./AddScreen";

const PAGE = 30;

type PeriodChoice = "all" | "this_month" | "last_month";

/**
 * The list of transactions with search and filters by account, category, tag
 * and period. It loads a page at a time (more on scroll, and a visible "Load
 * more" button), and every row opens its actions: edit and delete, with undo.
 */
export function TransactionsScreen(): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [search, setSearch] = useState("");
	const [accountUid, setAccountUid] = useState<string | null>(null);
	const [categoryUid, setCategoryUid] = useState<string | null>(null);
	const [tagUid, setTagUid] = useState<string | null>(null);
	const [period, setPeriod] = useState<PeriodChoice>("all");
	const [limit, setLimit] = useState(PAGE);
	const [selected, setSelected] = useState<TransactionListItem | null>(null);
	const [editing, setEditing] = useState<string | null>(null);
	const [undo, setUndo] = useState<{ uid: string } | null>(null);

	const today = app.today();
	const range =
		period === "all" ? null : resolvePeriod(period, "", "", today);
	const filter: TransactionFilter = {
		search,
		...(accountUid === null ? {} : { account_uid: accountUid }),
		...(categoryUid === null ? {} : { category_uid: categoryUid }),
		...(tagUid === null ? {} : { tag_uid: tagUid }),
		...(range === null ? {} : { from: range.from, to: range.to }),
	};

	const lookups = useLoad(async (a: App) => ({
		accounts: await a.repo.list("account"),
		categories: await a.repo.list("category"),
		tags: await a.repo.list("tag"),
	}));
	const rows = useLoad(
		(a: App) => listTransactions(a.db, filter, limit, 0),
		[search, accountUid, categoryUid, tagUid, period, limit, today],
	);

	function resetPaging<T>(set: (value: T) => void): (value: T) => void {
		return (value) => {
			set(value);
			setLimit(PAGE);
		};
	}

	async function remove(item: TransactionListItem): Promise<void> {
		await app.ledger.deleteTransaction(item.uid);
		setSelected(null);
		setUndo({ uid: item.uid });
		app.changed();
	}

	async function restore(uid: string): Promise<void> {
		await app.ledger.restoreTransaction(uid);
		setUndo(null);
		app.changed();
	}

	if (editing !== null) {
		return (
			<AddScreen
				editUid={editing}
				onDone={() => {
					setEditing(null);
					setSelected(null);
				}}
			/>
		);
	}

	const filterRow = (
		label: string,
		items: { uid: string; name: string }[],
		value: string | null,
		set: (v: string | null) => void,
	): ReactNode => (
		<View>
			<Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{label}</Text>
			<ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ gap: ui.spacing[2] }}>
				<Chip role="radio" label={`${label}: ${ui.t("all")}`} selected={value === null} onPress={() => set(null)} />
				{items.map((i) => (
					<Chip key={i.uid} role="radio" label={i.name} selected={value === i.uid} onPress={() => set(i.uid)} />
				))}
			</ScrollView>
		</View>
	);

	const header = (
		<View style={{ gap: ui.spacing[3], padding: ui.spacing[4] }}>
			<ScreenHeader title={ui.t("tabTransactions")} />
			<TextField label={ui.t("transactionsSearch")} value={search} onChangeText={resetPaging(setSearch)} />
			{filterRow(ui.t("filterAccount"), lookups?.accounts ?? [], accountUid, resetPaging(setAccountUid))}
			{filterRow(ui.t("filterCategory"), lookups?.categories ?? [], categoryUid, resetPaging(setCategoryUid))}
			{(lookups?.tags.length ?? 0) > 0 ? filterRow(ui.t("filterTag"), lookups?.tags ?? [], tagUid, resetPaging(setTagUid)) : null}
			<View>
				<Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("filterPeriod")}</Text>
				<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
					<Chip role="radio" label={ui.t("all")} selected={period === "all"} onPress={() => resetPaging(setPeriod)("all")} />
					<Chip
						role="radio"
						label={ui.t("periodThisMonth")}
						selected={period === "this_month"}
						onPress={() => resetPaging(setPeriod)("this_month")}
					/>
					<Chip
						role="radio"
						label={ui.t("periodLastMonth")}
						selected={period === "last_month"}
						onPress={() => resetPaging(setPeriod)("last_month")}
					/>
				</View>
			</View>
			{undo === null ? null : (
				<Toast message={ui.t("deleted")} action={{ label: ui.t("undo"), onPress: () => void restore(undo.uid) }} onDismiss={() => setUndo(null)} />
			)}
		</View>
	);

	const footer =
		rows !== undefined && rows.length >= limit ? (
			<View style={{ padding: ui.spacing[4] }}>
				<Button variant="secondary" label={ui.t("transactionsLoadMore")} onPress={() => setLimit((l) => l + PAGE)} />
			</View>
		) : null;

	return (
		<FlatList
			data={rows ?? []}
			keyExtractor={(item) => item.uid}
			ListHeaderComponent={header}
			ListFooterComponent={footer ?? undefined}
			initialNumToRender={PAGE}
			maxToRenderPerBatch={PAGE}
			ListEmptyComponent={rows === undefined ? undefined : <EmptyState message={ui.t("transactionsEmpty")} />}
			onEndReached={() => {
				if (rows !== undefined && rows.length >= limit) setLimit((l) => l + PAGE);
			}}
			onEndReachedThreshold={0.5}
			renderItem={({ item }) => (
				<View>
					<ListRow
						title={item.description !== "" ? item.description : (item.category_name ?? ui.t("uncategorized"))}
						subtitle={[item.occurred_on, item.account_name, item.transfer_uid !== null ? ui.t("transferLeg") : item.category_name]
							.filter((s): s is string => s !== null && s !== "")
							.join(" · ")}
						trailing={<MoneyText cents={item.amount_cents} kind={item.kind === "income" ? "income" : "expense"} />}
						onPress={() => setSelected(selected?.uid === item.uid ? null : item)}
					/>
					{selected?.uid === item.uid ? (
						<View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2], paddingHorizontal: ui.spacing[4] }}>
							{item.transfer_uid === null ? <Button variant="secondary" label={ui.t("edit")} onPress={() => setEditing(item.uid)} /> : null}
							<Button variant="destructive" label={ui.t("delete")} onPress={() => void remove(item)} />
						</View>
					) : null}
				</View>
			)}
		/>
	);
}

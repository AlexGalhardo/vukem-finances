import type { ReactNode } from "react";
import { ScrollView, Text, View } from "react-native";
import { type App, useApp, useLoad } from "../core/state";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { MoneyText } from "../components/MoneyText";
import { ScreenHeader } from "../components/ScreenHeader";
import { useUi } from "../components/ui";
import { accountBalances, netWorthOf } from "../domain/ledger";
import { upcoming } from "../domain/recurrence";
import { cashFlow, resolvePeriod } from "../domain/reports";

/** Net worth, this month's income and expense, and what is due in the next 7 days. */
export function HomeScreen(props: { onAdd?: () => void }): ReactNode {
  const app = useApp();
  const ui = useUi();
  const today = app.today();

  const data = useLoad(
    async (a: App) => {
      const t = a.today();
      const balances = await accountBalances(a.db, t);
      const flow = await cashFlow(a.db, resolvePeriod("this_month", "", "", t));
      const due = await upcoming(a.db, a.repo, t, 7);
      const count = await a.db.all<{ n: number }>("SELECT COUNT(*) AS n FROM transactions WHERE deleted_at IS NULL");
      return { net: netWorthOf(balances), flow: flow.totals, due, empty: (count[0]?.n ?? 0) === 0 };
    },
    [today],
  );

  return (
    <ScrollView contentContainerStyle={{ padding: ui.spacing[4], gap: ui.spacing[4] }}>
      <ScreenHeader title={ui.t("tabHome")} />
      {data === undefined ? null : (
        <>
          <View accessible accessibilityLabel={`${ui.t("homeNetWorth")}, ${ui.spoken(data.net)}`}>
            <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("homeNetWorth")}</Text>
            <MoneyText cents={data.net} style={{ fontSize: ui.fontSize.xl }} />
          </View>

          <View>
            <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
              {ui.t("homeThisMonth")}
            </Text>
            <ListRow title={ui.t("income")} trailing={<MoneyText cents={data.flow.income_cents} kind="income" />} />
            <ListRow title={ui.t("expense")} trailing={<MoneyText cents={data.flow.expense_cents} kind="expense" />} />
            <ListRow title={ui.t("net")} trailing={<MoneyText cents={data.flow.net_cents} />} />
          </View>

          {data.empty ? (
            <EmptyState
              message={ui.t("homeAddFirst")}
              action={props.onAdd === undefined ? undefined : { label: ui.t("add"), onPress: props.onAdd }}
            />
          ) : null}

          <View>
            <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
              {ui.t("homeUpcoming")}
            </Text>
            {data.due.length === 0 ? (
              <Text style={{ color: ui.colors.textMuted }}>{ui.t("homeNothingUpcoming")}</Text>
            ) : (
              data.due.map((item) => (
                <ListRow
                  key={`${item.date}-${item.transaction_uid ?? item.recurrence_uid ?? ""}`}
                  title={
                    item.description !== "" ? item.description : ui.t(item.kind === "income" ? "income" : "expense")
                  }
                  subtitle={item.date}
                  trailing={
                    <MoneyText cents={item.amount_cents} kind={item.kind === "income" ? "income" : "expense"} />
                  }
                />
              ))
            )}
          </View>
        </>
      )}
    </ScrollView>
  );
}

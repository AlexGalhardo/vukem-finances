import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
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
import { describeError } from "../core/errors";
import { type App, useApp, useLoad } from "../core/state";
import { type CardOverview, cardOverviews, createCard, payStatement } from "../domain/cards";
import { parseAmount } from "../domain/money";
import { cardStatements, type StatementStatus, type StatementSummary } from "../domain/statements";
import type { MessageKey } from "../i18n";

const STATUS_LABEL: Record<StatementStatus, MessageKey> = {
  open: "cardStatementOpen",
  closed: "cardStatementClosed",
  paid: "cardStatementPaid",
  overdue: "cardStatementOverdue",
};

type View_ =
  | { kind: "list" }
  | { kind: "new" }
  | { kind: "detail"; uid: string }
  | { kind: "pay"; uid: string; statement: StatementSummary };

/**
 * Credit cards: limit used and available, the statements of each (the status
 * is written out, never only a color), paying a statement as a transfer.
 */
export function CardsScreen(props: { onBack?: () => void }): ReactNode {
  const app = useApp();
  const ui = useUi();
  const [view, setView] = useState<View_>({ kind: "list" });
  const [name, setName] = useState("");
  const [limit, setLimit] = useState("");
  const [closing, setClosing] = useState("");
  const [due, setDue] = useState("");
  const [payFrom, setPayFrom] = useState<string | null>(null);
  const [payAmount, setPayAmount] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [paid, setPaid] = useState(false);

  const today = app.today();
  const data = useLoad(
    async (a: App) => {
      const overviews = await cardOverviews(a.db, a.repo, a.today());
      const statements = new Map<string, StatementSummary[]>();
      for (const o of overviews) {
        statements.set(o.card.uid, await cardStatements(a.db, o.card, a.today()));
      }
      const banks = (await a.repo.list("account")).filter((x) => x.kind !== "credit_card" && x.archived_at === null);
      return { overviews, statements, banks };
    },
    [today],
  );

  async function create(): Promise<void> {
    const cents = parseAmount(limit);
    if (cents === null || cents <= 0) {
      setError(ui.t("invalidAmount"));
      return;
    }
    try {
      await createCard(app.db, app.repo, {
        name,
        limit_cents: cents,
        closing_day: Number(closing),
        due_day: Number(due),
        payment_account_uid: data?.banks[0]?.uid ?? null,
      });
    } catch (e) {
      setError(describeError(ui, e));
      return;
    }
    setView({ kind: "list" });
    app.changed();
  }

  async function pay(uid: string, statement: StatementSummary): Promise<void> {
    const from =
      payFrom ?? data?.overviews.find((o) => o.card.uid === uid)?.card.payment_account_uid ?? data?.banks[0]?.uid;
    const cents = payAmount.trim() === "" ? null : parseAmount(payAmount);
    if (from === undefined || (payAmount.trim() !== "" && (cents === null || cents <= 0))) {
      setError(ui.t("invalidAmount"));
      return;
    }
    try {
      await payStatement(
        app.db,
        app.repo,
        app.ledger,
        uid,
        statement.ref,
        { from_account_uid: from, amount_cents: cents, date: today },
        today,
      );
    } catch (e) {
      setError(describeError(ui, e));
      return;
    }
    setPaid(true);
    setView({ kind: "detail", uid });
    app.changed();
  }

  async function remove(o: CardOverview): Promise<void> {
    await app.repo.remove("credit_card", o.card.uid);
    await app.ledger.deleteAccount(o.card.account_uid);
    setView({ kind: "list" });
    app.changed();
  }

  const errorText =
    error === null ? null : (
      <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
        {error}
      </Text>
    );

  if (view.kind === "new") {
    return (
      <Page title={ui.t("cardNew")} onBack={() => setView({ kind: "list" })}>
        <TextField label={ui.t("name")} value={name} onChangeText={setName} />
        <AmountField label={ui.t("cardLimit")} value={limit} onChangeText={setLimit} />
        <TextField label={ui.t("cardClosingDay")} value={closing} onChangeText={setClosing} keyboardType="number-pad" />
        <TextField label={ui.t("cardDueDay")} value={due} onChangeText={setDue} keyboardType="number-pad" />
        {errorText}
        <Button label={ui.t("save")} onPress={() => void create()} />
      </Page>
    );
  }

  if (view.kind === "pay") {
    const { uid, statement } = view;
    return (
      <Page title={ui.t("cardPay")} onBack={() => setView({ kind: "detail", uid })}>
        <View style={{ flexDirection: "row", gap: ui.spacing[2], alignItems: "center" }}>
          <Text style={{ color: ui.colors.text }}>{statement.ref}</Text>
          <MoneyText cents={statement.remaining_cents} />
        </View>
        <Text style={{ color: ui.colors.textMuted }}>{ui.t("cardPayFrom")}</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
          {(data?.banks ?? []).map((b) => (
            <Chip
              key={b.uid}
              role="radio"
              label={b.name}
              selected={
                (payFrom ??
                  data?.overviews.find((o) => o.card.uid === uid)?.card.payment_account_uid ??
                  data?.banks[0]?.uid) === b.uid
              }
              onPress={() => setPayFrom(b.uid)}
            />
          ))}
        </View>
        <AmountField label={ui.t("amount")} value={payAmount} onChangeText={setPayAmount} />
        {errorText}
        <Button label={ui.t("cardPay")} onPress={() => void pay(uid, statement)} />
      </Page>
    );
  }

  if (view.kind === "detail") {
    const overview = data?.overviews.find((o) => o.card.uid === view.uid);
    const statements = data?.statements.get(view.uid) ?? [];
    if (overview === undefined) {
      return (
        <Page title={ui.t("titleCards")} onBack={() => setView({ kind: "list" })}>
          {null}
        </Page>
      );
    }
    return (
      <Page title={overview.name} onBack={() => setView({ kind: "list" })}>
        {paid ? (
          <Text accessibilityRole="alert" style={{ color: ui.colors.text }}>
            {ui.t("cardPaid")}
          </Text>
        ) : null}
        <ListRow
          title={ui.t("cardLimitUsed")}
          subtitle={`${ui.t("cardLimit")}: ${ui.money(overview.card.limit_cents)}`}
          trailing={<MoneyText cents={overview.used_cents} />}
        />
        <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
          {ui.t("cardStatements")}
        </Text>
        {statements.map((s) => (
          <View key={s.ref}>
            <ListRow
              title={`${ui.t(STATUS_LABEL[s.status])} · ${s.closing_on}`}
              subtitle={ui.t("cardStatementDue", { date: s.due_on })}
              trailing={<MoneyText cents={s.total_cents} />}
            />
            {s.remaining_cents > 0 && s.status !== "open" ? (
              <View style={{ paddingHorizontal: ui.spacing[4] }}>
                <Button
                  variant="secondary"
                  label={`${ui.t("cardPay")} ${s.closing_on}`}
                  onPress={() => {
                    setError(null);
                    setPayAmount("");
                    setPaid(false);
                    setView({ kind: "pay", uid: overview.card.uid, statement: s });
                  }}
                />
              </View>
            ) : null}
          </View>
        ))}
        <DeleteButton onConfirm={() => void remove(overview)} />
      </Page>
    );
  }

  const overviews = data?.overviews ?? [];
  return (
    <Page
      title={ui.t("titleCards")}
      onBack={props.onBack}
      action={{
        label: ui.t("add"),
        onPress: () => {
          setError(null);
          setView({ kind: "new" });
        },
      }}
    >
      {data === undefined ? null : overviews.length === 0 ? (
        <EmptyState
          message={ui.t("cardsEmpty")}
          action={{ label: ui.t("cardNew"), onPress: () => setView({ kind: "new" }) }}
        />
      ) : (
        overviews.map((o) => (
          <ListRow
            key={o.card.uid}
            title={o.name}
            subtitle={`${ui.t("cardLimitUsed")}: ${ui.money(o.used_cents)} / ${ui.money(o.card.limit_cents)}`}
            trailing={<MoneyText cents={o.available_cents} />}
            onPress={() => setView({ kind: "detail", uid: o.card.uid })}
          />
        ))
      )}
    </Page>
  );
}

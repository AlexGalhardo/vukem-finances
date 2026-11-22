import { type ReactNode, useEffect, useState } from "react";
import { ScrollView, Text, View } from "react-native";
import { type App, useApp, useLoad } from "../core/state";
import { AmountField } from "../components/AmountField";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { ScreenHeader } from "../components/ScreenHeader";
import { TextField } from "../components/TextField";
import { Toast } from "../components/Toast";
import { useUi } from "../components/ui";
import { recentCategories } from "../db/queries";
import { addDays, isIsoDate } from "../domain/dates";
import { formatDecimalCents, parseAmount } from "../domain/money";
import { statementRefIn } from "../domain/statements";

export type Kind = "expense" | "income" | "transfer";

interface Props {
  /** Edit this transaction instead of making a new one. */
  editUid?: string;
  initialKind?: Kind;
  onDone?: () => void;
}

/**
 * The fast form: amount first on a numeric keypad, the kind, the account, the
 * recent categories of that kind as chips, the date and a note. The transfer
 * variant asks for two accounts instead of a category.
 */
export function AddScreen({ editUid, initialKind = "expense", onDone }: Props): ReactNode {
  const app = useApp();
  const ui = useUi();
  const [kind, setKind] = useState<Kind>(initialKind);
  const [amount, setAmount] = useState("");
  const [accountUid, setAccountUid] = useState("");
  const [toAccountUid, setToAccountUid] = useState("");
  const [categoryUid, setCategoryUid] = useState<string | null>(null);
  const [date, setDate] = useState(app.today());
  const [note, setNote] = useState("");
  const [tagUids, setTagUids] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);

  const data = useLoad(
    async (a: App) => {
      const accounts = (await a.repo.list("account")).filter((x) => x.archived_at === null);
      const tags = await a.repo.list("tag");
      const categories = await recentCategories(a.db, kind === "income" ? "income" : "expense", 12);
      return { accounts, categories, tags };
    },
    [kind],
  );

  // Edit mode: fill the form from the stored row once.
  useEffect(() => {
    if (editUid === undefined) {
      return;
    }
    app.repo.get("transaction", editUid).then((row) => {
      if (row === null) {
        return;
      }
      setKind(row.kind === "income" ? "income" : "expense");
      setAmount(formatDecimalCents(row.amount_cents));
      setAccountUid(row.account_uid);
      setCategoryUid(row.category_uid);
      setDate(row.occurred_on);
      setNote(row.description);
      setTagUids(row.tag_uids);
    });
  }, [app.repo, editUid]);

  // The first account is the default until the person picks one.
  const firstAccount = data?.accounts[0]?.uid ?? "";
  const account = accountUid === "" ? firstAccount : accountUid;
  const toAccount = toAccountUid === "" ? (data?.accounts.find((a) => a.uid !== account)?.uid ?? "") : toAccountUid;

  async function save(): Promise<void> {
    const cents = parseAmount(amount);
    if (cents === null || cents <= 0) {
      setError(ui.t("invalidAmount"));
      return;
    }
    if (!isIsoDate(date)) {
      setError(ui.t("invalidDate"));
      return;
    }
    if (account === "" || (kind === "transfer" && toAccount === "")) {
      setError(ui.t("required"));
      return;
    }
    setError(null);

    try {
      if (kind === "transfer") {
        await app.ledger.createTransfer({
          from_account_uid: account,
          to_account_uid: toAccount,
          amount_cents: cents,
          date,
          description: note,
        });
        setSaved(ui.t("addTransferSaved"));
      } else if (editUid !== undefined) {
        await app.repo.update("transaction", editUid, {
          kind,
          amount_cents: cents,
          account_uid: account,
          category_uid: categoryUid,
          occurred_on: date,
          description: note.trim(),
          tag_uids: tagUids,
          statement_ref: await statementRefIn(app.db, account, date),
        });
        setSaved(ui.t("saved"));
      } else {
        await app.repo.create("transaction", {
          account_uid: account,
          category_uid: categoryUid,
          kind,
          amount_cents: cents,
          description: note.trim(),
          occurred_on: date,
          transfer_uid: null,
          tag_uids: tagUids,
          statement_ref: await statementRefIn(app.db, account, date),
          recurrence_uid: null,
          occurrence_on: null,
          installment_group_uid: null,
          installment_no: null,
          installment_total: null,
        });
        setSaved(ui.t("addSaved"));
      }
    } catch {
      setError(ui.t("errorGeneric"));
      return;
    }

    app.changed();
    if (editUid === undefined) {
      setAmount("");
      setNote("");
      setTagUids([]);
    }
    onDone?.();
  }

  const kinds: { value: Kind; label: string }[] = [
    { value: "expense", label: ui.t("addKindExpense") },
    { value: "income", label: ui.t("addKindIncome") },
    ...(editUid === undefined ? [{ value: "transfer" as const, label: ui.t("addKindTransfer") }] : []),
  ];

  return (
    <ScrollView
      contentContainerStyle={{ padding: ui.spacing[4], gap: ui.spacing[4] }}
      keyboardShouldPersistTaps="handled"
    >
      <ScreenHeader
        title={
          editUid !== undefined
            ? ui.t("editTransaction")
            : kind === "transfer"
              ? ui.t("addTransferTitle")
              : ui.t("addTitle")
        }
        onBack={editUid === undefined ? undefined : onDone}
      />

      <AmountField value={amount} onChangeText={setAmount} autoFocus />

      <View accessibilityRole="radiogroup" style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
        {kinds.map((k) => (
          <Chip
            key={k.value}
            role="radio"
            label={k.label}
            selected={kind === k.value}
            onPress={() => setKind(k.value)}
          />
        ))}
      </View>

      <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>
        {kind === "transfer" ? ui.t("addFromAccount") : ui.t("account")}
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
        {(data?.accounts ?? []).map((a) => (
          <Chip
            key={a.uid}
            role="radio"
            label={a.name}
            selected={account === a.uid}
            onPress={() => setAccountUid(a.uid)}
          />
        ))}
      </View>

      {kind === "transfer" ? (
        <>
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("addToAccount")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
            {(data?.accounts ?? [])
              .filter((a) => a.uid !== account)
              .map((a) => (
                <Chip
                  key={a.uid}
                  role="radio"
                  label={a.name}
                  selected={toAccount === a.uid}
                  onPress={() => setToAccountUid(a.uid)}
                />
              ))}
          </View>
        </>
      ) : (
        <>
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("addRecent")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
            <Chip
              role="radio"
              label={ui.t("uncategorized")}
              selected={categoryUid === null}
              onPress={() => setCategoryUid(null)}
            />
            {(data?.categories ?? []).map((c) => (
              <Chip
                key={c.uid}
                role="radio"
                label={c.name}
                selected={categoryUid === c.uid}
                onPress={() => setCategoryUid(c.uid)}
              />
            ))}
          </View>
        </>
      )}

      {kind !== "transfer" && (data?.tags.length ?? 0) > 0 ? (
        <>
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("tags")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
            {(data?.tags ?? []).map((t) => (
              <Chip
                key={t.uid}
                role="checkbox"
                label={t.name}
                selected={tagUids.includes(t.uid)}
                onPress={() =>
                  setTagUids((cur) => (cur.includes(t.uid) ? cur.filter((x) => x !== t.uid) : [...cur, t.uid]))
                }
              />
            ))}
          </View>
        </>
      ) : null}

      <TextField label={ui.t("date")} value={date} onChangeText={setDate} keyboardType="numbers-and-punctuation" />
      <View style={{ flexDirection: "row", gap: ui.spacing[2] }}>
        <Chip label={ui.t("today")} onPress={() => setDate(app.today())} />
        <Chip label={ui.t("yesterday")} onPress={() => setDate(addDays(app.today(), -1))} />
      </View>

      <TextField label={ui.t("note")} value={note} onChangeText={setNote} />

      {error === null ? null : (
        <Text accessibilityRole="alert" accessibilityLiveRegion="polite" style={{ color: ui.colors.expense }}>
          {error}
        </Text>
      )}
      <Button label={ui.t("save")} onPress={() => void save()} />
      {saved === null ? null : <Toast message={saved} onDismiss={() => setSaved(null)} />}
    </ScrollView>
  );
}

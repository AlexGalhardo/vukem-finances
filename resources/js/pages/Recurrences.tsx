import { HelpHint } from "../help/HelpHint";
import { Head, useForm, usePage } from "@inertiajs/react";
import { Pencil, Plus, Repeat } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import AmountInput from "../components/AmountInput";
import ConfirmDelete from "../components/ConfirmDelete";
import DataTable, { type DataTableColumn } from "../components/DataTable";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import UpcomingList from "../components/recurrences/UpcomingList";
import { centsToInput } from "../components/transactions/entryForm";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "../components/ui/empty";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { type Key, useFinanceI18n } from "../lib/i18n";
import { type Frequency, occurrences, type Rule } from "../lib/recurrence";
import type { Page, Recurrence, SharedProps } from "../page-props";

interface RuleForm {
  accountId: string;
  categoryId: string;
  kind: string;
  amount: string;
  description: string;
  frequency: string;
  interval: string;
  startDate: string;
  endDate: string;
  maxCount: string;
  active: boolean;
  backfill: boolean;
}

const frequencies: { value: Frequency; label: Key }[] = [
  { value: "daily", label: "recurrenceDaily" },
  { value: "weekly", label: "recurrenceWeekly" },
  { value: "monthly", label: "recurrenceMonthly" },
  { value: "yearly", label: "recurrenceYearly" },
];

const everyKey: Record<string, Key> = {
  daily: "recurrenceEveryDaily",
  weekly: "recurrenceEveryWeekly",
  monthly: "recurrenceEveryMonthly",
  yearly: "recurrenceEveryYearly",
};

function emptyForm(accountId: string, today: string): RuleForm {
  return {
    accountId,
    categoryId: "",
    kind: "expense",
    amount: "",
    description: "",
    frequency: "monthly",
    interval: "1",
    startDate: today,
    endDate: "",
    maxCount: "",
    active: true,
    backfill: true,
  };
}

function formOf(rule: Recurrence): RuleForm {
  return {
    accountId: String(rule.account_id),
    categoryId: rule.category_id.Valid ? String(rule.category_id.V) : "",
    kind: rule.kind,
    amount: centsToInput(rule.amount_cents),
    description: rule.description,
    frequency: rule.frequency,
    interval: String(rule.interval),
    startDate: rule.start_date,
    endDate: rule.end_date.Valid ? rule.end_date.V : "",
    maxCount: rule.max_count.Valid ? String(rule.max_count.V) : "",
    active: rule.active,
    backfill: false,
  };
}

/** The rule the form describes, for the preview; null while a field cannot be read. */
function previewRule(data: RuleForm): Rule | null {
  const interval = Number(data.interval);
  if (!Number.isInteger(interval) || interval < 1 || !/^\d{4}-\d{2}-\d{2}$/.test(data.startDate)) return null;
  const maxCount = data.maxCount === "" ? null : Number(data.maxCount);
  if (maxCount !== null && (!Number.isInteger(maxCount) || maxCount < 1)) return null;
  return {
    frequency: data.frequency as Frequency,
    interval,
    start_date: data.startDate,
    end_date: data.endDate === "" ? null : data.endDate,
    max_count: maxCount,
  };
}

function nextDates(rule: Rule | null, today: string, count: number): string[] {
  if (rule === null) return [];
  const from = rule.start_date > today ? rule.start_date : today;
  return occurrences(rule, from, "9999-12-31", count);
}

export default function Recurrences({ recurrences, upcoming, accounts, categories, today }: Page<"Recurrences">) {
  const shared = usePage<SharedProps>().props;
  const currency = shared.workspace?.current.currency ?? "USD";
  const canManage = shared.workspace?.role !== "viewer";
  const { t, locale, date } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const firstAccount = accounts.find((a) => !a.archived_at && a.kind !== "credit_card") ?? accounts[0];
  const form = useForm<RuleForm>(emptyForm(firstAccount ? String(firstAccount.id) : "", today));
  const [editing, setEditing] = useState<Recurrence | null>(null);
  const fieldError = (name: keyof RuleForm): string | undefined => form.errors[name];

  const preview = nextDates(previewRule(form.data), today, 5);
  const pastStart = editing === null && form.data.startDate !== "" && form.data.startDate < today;
  const kindCategories = categories.filter((c) => c.kind === form.data.kind && !c.archived_at);

  function reset(): void {
    setEditing(null);
    form.setData(emptyForm(firstAccount ? String(firstAccount.id) : "", today));
    form.clearErrors();
  }

  function submit(event: FormEvent): void {
    event.preventDefault();
    if (editing) form.patch(`/recurrences/${editing.id}`, { preserveScroll: true, onSuccess: reset });
    else form.post("/recurrences", { preserveScroll: true, onSuccess: reset });
  }

  function edit(rule: Recurrence): void {
    setEditing(rule);
    form.setData(formOf(rule));
    form.clearErrors();
    requestAnimationFrame(() => document.getElementById("rule-description")?.focus());
  }

  const columns: DataTableColumn<Recurrence>[] = [
    { id: "description", header: t("description"), cell: (r) => r.description || "—" },
    {
      id: "amount",
      header: t("amount"),
      align: "end",
      cell: (r) => (
        <MoneyText
          cents={r.kind === "expense" ? -r.amount_cents : r.amount_cents}
          currency={currency}
          locale={locale}
          labels={labels}
          tone
        />
      ),
    },
    {
      id: "frequency",
      header: t("recurrenceFrequency"),
      cell: (r) =>
        everyKey[r.frequency] ? t(everyKey[r.frequency] as Key).replace("{n}", String(r.interval)) : r.frequency,
    },
    {
      id: "next",
      header: t("recurrenceNextDates"),
      cell: (r) => {
        const next = r.active
          ? nextDates(
              {
                frequency: r.frequency as Frequency,
                interval: r.interval,
                start_date: r.start_date,
                end_date: r.end_date.Valid ? r.end_date.V : null,
                max_count: r.max_count.Valid ? r.max_count.V : null,
              },
              today,
              1,
            )[0]
          : undefined;
        return next ? date(next) : t("recurrenceNoMoreDates");
      },
    },
    {
      id: "status",
      header: t("recurrenceStatus"),
      cell: (r) => (
        <div className="flex flex-col gap-1">
          <Badge variant={r.active ? "default" : "secondary"} className="w-fit">
            {r.active ? t("recurrenceActive") : t("recurrencePaused")}
          </Badge>
          {r.last_error.Valid && (
            <span role="status" className="text-xs text-destructive">
              {t("recurrenceLastError")}: {r.last_error.V}. {t("recurrenceRetry")}
            </span>
          )}
        </div>
      ),
    },
  ];
  if (canManage) {
    columns.push({
      id: "actions",
      header: t("actions"),
      align: "end",
      cell: (r) => (
        <div className="flex flex-wrap justify-end gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => edit(r)}>
            <Pencil data-icon="inline-start" />
            {t("edit")}
          </Button>
          <ConfirmDelete
            href={`/recurrences/${r.id}`}
            name={r.description || t("recurrences")}
            trigger={
              <Button type="button" variant="outline" size="sm">
                {t("recurrenceDelete")}
              </Button>
            }
          />
          <ConfirmDelete
            href={`/recurrences/${r.id}?future=1`}
            name={r.description || t("recurrences")}
            trigger={
              <Button type="button" variant="outline" size="sm">
                {t("recurrenceDeleteFuture")}
              </Button>
            }
          />
        </div>
      ),
    });
  }

  return (
    <>
      <Head title={t("recurrences")} />
      <PageHeader title={t("recurrences")} description={t("recurrencesDescription")} />
      <p className="max-w-[65ch] text-body text-muted-foreground">{t("recurrencesHelp")}</p>
      {recurrences.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Repeat />
            </EmptyMedia>
            <EmptyTitle>{t("recurrenceEmpty")}</EmptyTitle>
            <EmptyDescription>{t("recurrenceEmptyDescription")}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div data-tour="recurrences-table">
          <DataTable columns={columns} rows={recurrences} rowKey={(r) => r.id} caption={t("recurrencesTable")} />
        </div>
      )}
      {recurrences.length > 0 && canManage && (
        <p className="text-caption text-muted-foreground">{t("recurrenceDeleteHelp")}</p>
      )}
      <Card data-tour="recurrences-upcoming">
        <CardHeader>
          <CardTitle>{t("upcoming")}</CardTitle>
          <CardDescription>{t("upcomingDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <UpcomingList items={upcoming} currency={currency} />
        </CardContent>
      </Card>
      {canManage && (
        <Card data-tour="recurrences-new">
          <CardHeader>
            <CardTitle>{editing ? t("recurrenceEdit") : t("recurrenceNew")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="flex flex-col gap-4" noValidate>
              <FieldGroup className="grid gap-3 sm:grid-cols-2">
                <Field data-invalid={!!fieldError("kind")}>
                  <FieldLabel htmlFor="rule-kind">{t("kind")}</FieldLabel>
                  <NativeSelect
                    id="rule-kind"
                    value={form.data.kind}
                    onChange={(e) => {
                      form.setData("kind", e.target.value);
                      form.setData("categoryId", "");
                    }}
                  >
                    <NativeSelectOption value="expense">{t("expense")}</NativeSelectOption>
                    <NativeSelectOption value="income">{t("income")}</NativeSelectOption>
                  </NativeSelect>
                  <FieldError>{fieldError("kind")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("amount")}>
                  <FieldLabel htmlFor="rule-amount">{t("amount")}</FieldLabel>
                  <AmountInput
                    id="rule-amount"
                    required
                    placeholder="800.00"
                    value={form.data.amount}
                    aria-invalid={!!fieldError("amount")}
                    onChange={(e) => form.setData("amount", e.target.value)}
                  />
                  <FieldError>{fieldError("amount")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("description")}>
                  <FieldLabel htmlFor="rule-description">{t("description")}</FieldLabel>
                  <Input
                    id="rule-description"
                    maxLength={200}
                    value={form.data.description}
                    onChange={(e) => form.setData("description", e.target.value)}
                  />
                  <FieldError>{fieldError("description")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("accountId")}>
                  <FieldLabel htmlFor="rule-account">{t("account")}</FieldLabel>
                  <NativeSelect
                    id="rule-account"
                    value={form.data.accountId}
                    onChange={(e) => form.setData("accountId", e.target.value)}
                  >
                    {accounts.map((a) => (
                      <NativeSelectOption key={a.id} value={String(a.id)}>
                        {a.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                  <FieldError>{fieldError("accountId")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("categoryId")}>
                  <FieldLabel htmlFor="rule-category">{t("category")}</FieldLabel>
                  <NativeSelect
                    id="rule-category"
                    value={form.data.categoryId}
                    onChange={(e) => form.setData("categoryId", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("none")}</NativeSelectOption>
                    {kindCategories.map((c) => (
                      <NativeSelectOption key={c.id} value={String(c.id)}>
                        {c.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                  <FieldError>{fieldError("categoryId")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("frequency")}>
                  <span className="flex items-center gap-1">
                    <FieldLabel htmlFor="rule-frequency">{t("recurrenceFrequency")}</FieldLabel>
                    <HelpHint id="hintRecFrequency" />
                  </span>
                  <NativeSelect
                    id="rule-frequency"
                    value={form.data.frequency}
                    onChange={(e) => form.setData("frequency", e.target.value)}
                  >
                    {frequencies.map((f) => (
                      <NativeSelectOption key={f.value} value={f.value}>
                        {t(f.label)}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                  <FieldError>{fieldError("frequency")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("interval")}>
                  <FieldLabel htmlFor="rule-interval">{t("recurrenceInterval")}</FieldLabel>
                  <Input
                    id="rule-interval"
                    inputMode="numeric"
                    maxLength={3}
                    value={form.data.interval}
                    aria-invalid={!!fieldError("interval")}
                    onChange={(e) => form.setData("interval", e.target.value)}
                  />
                  <FieldDescription>{t("recurrenceIntervalHelp")}</FieldDescription>
                  <FieldError>{fieldError("interval")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("startDate")}>
                  <FieldLabel htmlFor="rule-start">{t("recurrenceStart")}</FieldLabel>
                  <Input
                    id="rule-start"
                    type="date"
                    required
                    value={form.data.startDate}
                    aria-invalid={!!fieldError("startDate")}
                    onChange={(e) => form.setData("startDate", e.target.value)}
                  />
                  <FieldError>{fieldError("startDate")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("endDate")}>
                  <FieldLabel htmlFor="rule-end">{t("recurrenceEnd")}</FieldLabel>
                  <Input
                    id="rule-end"
                    type="date"
                    value={form.data.endDate}
                    aria-invalid={!!fieldError("endDate")}
                    onChange={(e) => form.setData("endDate", e.target.value)}
                  />
                  <FieldError>{fieldError("endDate")}</FieldError>
                </Field>
                <Field data-invalid={!!fieldError("maxCount")}>
                  <FieldLabel htmlFor="rule-count">{t("recurrenceMaxCount")}</FieldLabel>
                  <Input
                    id="rule-count"
                    inputMode="numeric"
                    maxLength={4}
                    value={form.data.maxCount}
                    aria-invalid={!!fieldError("maxCount")}
                    onChange={(e) => form.setData("maxCount", e.target.value)}
                  />
                  <FieldError>{fieldError("maxCount")}</FieldError>
                </Field>
              </FieldGroup>
              {pastStart && (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={form.data.backfill}
                    onChange={(e) => form.setData("backfill", e.target.checked)}
                  />
                  {t("recurrenceBackfill")}
                </label>
              )}
              {editing && (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={form.data.active}
                    onChange={(e) => form.setData("active", e.target.checked)}
                  />
                  {t("recurrenceActive")}
                </label>
              )}
              <p className="text-sm" aria-live="polite">
                <strong>{t("recurrenceNextDates")}:</strong>{" "}
                {preview.length > 0 ? preview.map((d) => date(d)).join(" · ") : t("recurrenceNoMoreDates")}
              </p>
              <div className="flex gap-2">
                <Button type="submit" disabled={form.processing}>
                  <Plus data-icon="inline-start" />
                  {editing ? t("recurrenceSave") : t("recurrenceAdd")}
                </Button>
                {editing && (
                  <Button type="button" variant="outline" onClick={reset}>
                    {t("cancel")}
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      )}
    </>
  );
}

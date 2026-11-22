import { Head, router, useForm, usePage } from "@inertiajs/react";
import { CheckCircle2 } from "lucide-react";
import { type FormEvent, useState } from "react";

import ConfirmDelete from "../components/ConfirmDelete";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import { Card, danger, primary, secondary } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import { fill } from "../lib/notifications";
import type { GoalStatus, Page, SharedProps } from "../page-props";

function ContributionForm({ goal, today }: { goal: GoalStatus; today: string }) {
  const { t } = useFinanceI18n();
  const form = useForm({ amount: "", date: today, note: "" });
  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post(`/goals/${goal.id}/contributions`, {
      preserveScroll: true,
      onSuccess: () => form.reset("amount", "note"),
    });
  };
  return (
    <form onSubmit={submit} className="grid gap-3 sm:grid-cols-4">
      <Field
        label={t("goalAmount")}
        inputMode="decimal"
        required
        value={form.data.amount}
        onChange={(e) => form.setData("amount", e.target.value)}
        error={form.errors.amount}
        aria-describedby={`amount-help-${goal.id}`}
      />
      <Field
        label={t("goalDate")}
        type="date"
        value={form.data.date}
        onChange={(e) => form.setData("date", e.target.value)}
        error={form.errors.date}
      />
      <Field
        label={t("goalNote")}
        value={form.data.note}
        maxLength={200}
        onChange={(e) => form.setData("note", e.target.value)}
        error={form.errors.note}
      />
      <div className="self-end">
        <button type="submit" disabled={form.processing} className={primary}>
          {t("goalContributionAdd")}
        </button>
      </div>
      <p id={`amount-help-${goal.id}`} className="text-xs text-muted-foreground sm:col-span-4">
        {t("goalAmountHelp")}
      </p>
    </form>
  );
}

export default function Goals({ goals, contributions, today }: Page<"Goals">) {
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const { t, money, date } = useFinanceI18n();
  const form = useForm({ name: "", target: "", targetDate: "" });
  const [adding, setAdding] = useState<number | null>(null);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post("/goals", { onSuccess: () => form.reset() });
  };

  return (
    <>
      <Head title={t("goals")} />
      <PageHeader title={t("goals")} description={t("goalsDescription")} />
      {goals.length === 0 ? (
        <Card>
          <p className="font-medium">{t("goalsEmpty")}</p>
          <p className="text-muted-foreground">{t("goalsEmptyDescription")}</p>
        </Card>
      ) : (
        <ul className="flex flex-col gap-4">
          {goals.map((g) => {
            const parts = contributions.filter((c) => c.goal_id === g.id);
            return (
              <li key={g.id}>
                <Card>
                  <div className="flex flex-col gap-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <h2 className="text-lg font-semibold">
                        {g.name} {g.archived && <span className="text-sm font-normal">({t("goalArchived")})</span>}
                      </h2>
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          className={secondary}
                          onClick={() =>
                            router.post(
                              `/goals/${g.id}/${g.archived ? "unarchive" : "archive"}`,
                              {},
                              { preserveScroll: true },
                            )
                          }
                        >
                          {g.archived ? t("goalUnarchive") : t("goalArchive")}
                        </button>
                        <ConfirmDelete
                          href={`/goals/${g.id}`}
                          name={g.name}
                          trigger={
                            <button type="button" className={danger}>
                              {t("delete")}
                            </button>
                          }
                        />
                      </div>
                    </div>
                    <div
                      role="progressbar"
                      aria-label={fill(t("goalProgress"), { name: g.name })}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-valuenow={Math.min(g.percent, 100)}
                      aria-valuetext={`${g.percent}%`}
                      className="h-2 w-full overflow-hidden rounded-full bg-muted"
                    >
                      <div className="h-full bg-primary" style={{ width: `${Math.min(g.percent, 100)}%` }} />
                    </div>
                    <p className="flex flex-wrap items-center gap-x-3 text-sm">
                      <span className="font-medium">{g.percent}%</span>
                      <span>
                        {fill(t("goalSaved"), {
                          saved: money(g.savedCents, currency),
                          target: money(g.targetCents, currency),
                        })}
                      </span>
                      {g.reached ? (
                        <span className="inline-flex items-center gap-1 font-medium text-emerald-700 dark:text-emerald-400">
                          <CheckCircle2 aria-hidden="true" className="size-4" />
                          {t("goalReached")}
                        </span>
                      ) : (
                        <span>{fill(t("goalRemaining"), { amount: money(g.remainingCents, currency) })}</span>
                      )}
                    </p>
                    {!g.reached && g.targetDate && !g.overdue && (
                      <p className="text-sm">
                        {fill(t("goalRequired"), { amount: money(g.requiredMonthlyCents, currency), n: g.monthsLeft })}
                        {" · "}
                        {date(g.targetDate)}
                      </p>
                    )}
                    {g.overdue && (
                      <p className="text-sm font-medium text-red-700 dark:text-red-400">{t("goalOverdue")}</p>
                    )}
                    {g.projectedDate && (
                      <p className="text-sm text-muted-foreground">
                        {fill(t("goalProjected"), { date: date(g.projectedDate) })}
                      </p>
                    )}
                    {!g.archived && (
                      <div>
                        <button
                          type="button"
                          className={secondary}
                          aria-expanded={adding === g.id}
                          onClick={() => setAdding(adding === g.id ? null : g.id)}
                        >
                          {t("goalContribute")}
                        </button>
                      </div>
                    )}
                    {adding === g.id && <ContributionForm goal={g} today={today} />}
                    <div>
                      <h3 className="text-sm font-medium">{t("goalContributions")}</h3>
                      {parts.length === 0 ? (
                        <p className="text-sm text-muted-foreground">{t("goalNoContributions")}</p>
                      ) : (
                        <ul className="divide-y divide-border text-sm">
                          {parts.map((c) => (
                            <li key={c.id} className="flex items-center justify-between gap-2 py-1">
                              <span>
                                {date(c.on_date)} · {money(c.amount_cents, currency)}
                                {c.note ? ` · ${c.note}` : ""}
                              </span>
                              <ConfirmDelete
                                href={`/goals/${g.id}/contributions/${c.id}`}
                                name={money(c.amount_cents, currency)}
                                trigger={
                                  <button type="button" className={danger}>
                                    {t("delete")}
                                  </button>
                                }
                              />
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  </div>
                </Card>
              </li>
            );
          })}
        </ul>
      )}
      <Card title={t("goalNew")}>
        <form onSubmit={submit} className="grid gap-3 sm:grid-cols-4">
          <Field
            label={t("goalName")}
            required
            maxLength={120}
            value={form.data.name}
            onChange={(e) => form.setData("name", e.target.value)}
            error={form.errors.name}
          />
          <Field
            label={t("goalTarget")}
            inputMode="decimal"
            placeholder="1200.00"
            required
            value={form.data.target}
            onChange={(e) => form.setData("target", e.target.value)}
            error={form.errors.target}
          />
          <Field
            label={t("goalTargetDate")}
            type="date"
            value={form.data.targetDate}
            onChange={(e) => form.setData("targetDate", e.target.value)}
            error={form.errors.targetDate}
          />
          <div className="self-end">
            <button type="submit" disabled={form.processing} className={primary}>
              {t("goalSave")}
            </button>
          </div>
        </form>
      </Card>
    </>
  );
}

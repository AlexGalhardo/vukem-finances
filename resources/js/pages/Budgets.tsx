import { Head, Link, useForm, usePage } from "@inertiajs/react";
import { AlertTriangle, CheckCircle2, OctagonAlert } from "lucide-react";
import type { FormEvent } from "react";

import ConfirmDelete from "../components/ConfirmDelete";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import { Card, danger, primary, Select } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import { fill } from "../lib/notifications";
import type { Page, SharedProps } from "../page-props";

/** The month before or after a "YYYY-MM", by plain arithmetic on the numbers. */
export function shiftMonth(month: string, by: number): string {
  const [y, m] = month.split("-").map(Number) as [number, number];
  const index = y * 12 + (m - 1) + by;
  return `${Math.floor(index / 12)}-${String((index % 12) + 1).padStart(2, "0")}`;
}

/** Status is text and an icon, never only a color. */
function statusOf(percent: number): {
  key: "budgetsStatusOk" | "budgetsStatusNear" | "budgetsStatusOver";
  tone: string;
} {
  if (percent >= 100) return { key: "budgetsStatusOver", tone: "text-red-700 dark:text-red-400" };
  if (percent >= 80) return { key: "budgetsStatusNear", tone: "text-amber-700 dark:text-amber-400" };
  return { key: "budgetsStatusOk", tone: "text-emerald-700 dark:text-emerald-400" };
}

export default function Budgets({ budgets, categories, month, enabled }: Page<"Budgets">) {
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const expense = categories.filter((c) => c.kind === "expense");
  const { t, money, month: monthLabel } = useFinanceI18n();
  const form = useForm({ categoryId: "", limit: "", rollover: false });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post("/budgets", { onSuccess: () => form.reset("limit") });
  };

  return (
    <>
      <Head title={t("budgets")} />
      <PageHeader
        title={`${t("budgets")} · ${monthLabel(month)}`}
        actions={
          <nav aria-label={t("budgets")} className="flex gap-2 text-sm">
            <Link href={`/budgets?month=${shiftMonth(month, -1)}`} className="underline">
              {t("budgetsPreviousMonth")}
            </Link>
            <Link href={`/budgets?month=${shiftMonth(month, 1)}`} className="underline">
              {t("budgetsNextMonth")}
            </Link>
          </nav>
        }
      />
      {!enabled && (
        <p className="rounded-md bg-amber-50 px-4 py-3 text-amber-900 dark:bg-amber-950 dark:text-amber-100">
          {t("budgetsProOnly")}{" "}
          <Link href="/billing" className="font-medium underline">
            {t("budgetsProTrial")}
          </Link>
          .
        </p>
      )}
      <Card>
        {budgets.length === 0 ? (
          <p className="text-muted-foreground">{t("budgetsEmpty")}</p>
        ) : (
          <ul className="divide-y divide-border">
            {budgets.map((b) => {
              const name = b.overall ? t("budgetsOverall") : b.category;
              const status = statusOf(b.percent);
              const Icon = b.percent >= 100 ? OctagonAlert : b.percent >= 80 ? AlertTriangle : CheckCircle2;
              return (
                <li key={b.id} className="flex flex-col gap-2 py-3">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{name}</span>
                    <ConfirmDelete
                      href={`/budgets/${b.id}`}
                      name={name}
                      trigger={
                        <button type="button" className={danger}>
                          {t("delete")}
                        </button>
                      }
                    />
                  </div>
                  <div
                    role="progressbar"
                    aria-label={fill(t("budgetsProgress"), { name })}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={Math.min(b.percent, 100)}
                    aria-valuetext={`${b.percent}%`}
                    className="h-2 w-full overflow-hidden rounded-full bg-muted"
                  >
                    <div className="h-full bg-primary" style={{ width: `${Math.min(b.percent, 100)}%` }} />
                  </div>
                  <p className="flex flex-wrap items-center gap-x-3 text-sm">
                    <span className={`inline-flex items-center gap-1 font-medium ${status.tone}`}>
                      <Icon aria-hidden="true" className="size-4" />
                      {b.percent}% · {t(status.key)}
                    </span>
                    <span>
                      {fill(t("budgetsSpent"), {
                        spent: money(b.spentCents, currency),
                        limit: money(b.availableCents, currency),
                      })}
                    </span>
                    {b.carryCents > 0 && (
                      <span className="text-muted-foreground">
                        {fill(t("budgetsCarry"), { amount: money(b.carryCents, currency) })}
                      </span>
                    )}
                  </p>
                  {b.overall && <p className="text-xs text-muted-foreground">{t("budgetsOverallHelp")}</p>}
                </li>
              );
            })}
          </ul>
        )}
      </Card>
      <Card title={t("setBudget")}>
        <form onSubmit={submit} className="grid gap-3 sm:grid-cols-3">
          <Select
            label={t("category")}
            value={form.data.categoryId}
            onChange={(e) => form.setData("categoryId", e.target.value)}
            error={form.errors.categoryId}
          >
            <option value="">{t("budgetsAllSpending")}</option>
            {expense.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
          <Field
            label={t("monthlyLimit")}
            inputMode="decimal"
            placeholder="300.00"
            required
            value={form.data.limit}
            onChange={(e) => form.setData("limit", e.target.value)}
            error={form.errors.limit}
          />
          <div className="self-end">
            <button type="submit" disabled={form.processing} className={primary}>
              {t("budgetsSave")}
            </button>
          </div>
          <div className="sm:col-span-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.data.rollover}
                onChange={(e) => form.setData("rollover", e.target.checked)}
                aria-describedby="rollover-help"
              />
              {t("budgetsRollover")}
            </label>
            <p id="rollover-help" className="mt-1 text-xs text-muted-foreground">
              {t("budgetsRolloverHelp")}
            </p>
          </div>
        </form>
      </Card>
    </>
  );
}

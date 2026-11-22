import { Head, Link } from "@inertiajs/react";
import { ArrowUpRight, PiggyBank } from "lucide-react";
import BarChart from "../components/charts/BarChart";
import DonutChart from "../components/charts/DonutChart";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import UpcomingList from "../components/recurrences/UpcomingList";
import Stat from "../components/Stat";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "../components/ui/empty";
import { Progress } from "../components/ui/progress";
import { HelpHint } from "../help/HelpHint";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Dashboard({ dashboard, upcoming }: Page<"Dashboard">) {
  const { t, month: monthText, locale } = useFinanceI18n();
  const { currency, month, balanceCents, months, categories, budgets } = dashboard;
  const current = months[months.length - 1];
  const income = current?.incomeCents ?? 0;
  const expense = current?.expenseCents ?? 0;
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const figure = (cents: number, kind?: "income" | "expense", colored = true) => (
    <MoneyText
      cents={kind === "expense" ? -cents : cents}
      currency={currency}
      locale={locale}
      kind={kind}
      colored={colored}
      labels={labels}
      size="lg"
    />
  );
  return (
    <>
      <Head title={t("dashboard")} />
      <PageHeader title={t("overview")} description={`${monthText(month)} · ${t("overviewDescription")}`} />
      <section data-tour="dashboard-month" aria-label={monthText(month)} className="flex flex-col gap-6">
        <div data-tour="dashboard-balance">
          <Stat
            label={t("balance")}
            size="hero"
            help={<HelpHint id="hintDashboardBalance" />}
            value={<MoneyText cents={balanceCents} currency={currency} locale={locale} size="hero" labels={labels} />}
          />
        </div>
        <div className="grid gap-6 sm:grid-cols-3">
          <Stat
            label={t("incomeMonth")}
            help={<HelpHint id="hintDashboardIncome" />}
            value={figure(income, "income")}
          />
          <Stat
            label={t("expenseMonth")}
            help={<HelpHint id="hintDashboardExpenses" />}
            value={figure(expense, "expense")}
          />
          <Stat label={t("dashNet")} value={figure(income - expense, undefined)} />
        </div>
      </section>
      <div className="grid gap-6 lg:grid-cols-3">
        <Card data-tour="dashboard-cashflow" className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center gap-1.5">
              <CardTitle>{t("cashFlow")}</CardTitle>
              <HelpHint id="hintDashboardCashFlow" />
            </div>
            <CardDescription>{monthText(month)}</CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart points={months} currency={currency} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("spending")}</CardTitle>
            <CardDescription>{t("expenseMonth")}</CardDescription>
          </CardHeader>
          <CardContent>
            <DonutChart slices={categories} currency={currency} />
          </CardContent>
        </Card>
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        <Card data-tour="dashboard-upcoming">
          <CardHeader>
            <CardTitle>{t("upcoming")}</CardTitle>
            <CardDescription>{t("upcomingDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <UpcomingList items={upcoming} currency={currency} />
            <Link href="/recurrences" className="text-body text-brand-text underline underline-offset-4">
              {t("upcomingAll")}
            </Link>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("budgets")}</CardTitle>
            <CardDescription>{t("budgetDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            {budgets.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <PiggyBank />
                  </EmptyMedia>
                  <EmptyTitle>{t("emptyBudgets")}</EmptyTitle>
                </EmptyHeader>
                <Button asChild variant="outline">
                  <Link href="/budgets">
                    {t("setBudget")}
                    <ArrowUpRight data-icon="inline-end" />
                  </Link>
                </Button>
              </Empty>
            ) : (
              <ul className="flex flex-col gap-5">
                {budgets.map((b) => {
                  const name = b.overall ? t("budgetsOverall") : b.category;
                  return (
                    <li key={b.id} className="flex flex-col gap-2">
                      <div className="flex justify-between gap-4 text-body">
                        <span>{name}</span>
                        <span className="tabular-nums text-muted-foreground">
                          <MoneyText cents={b.spentCents} currency={currency} locale={locale} /> {t("of")}{" "}
                          <MoneyText cents={b.limitCents} currency={currency} locale={locale} />
                        </span>
                      </div>
                      <Progress value={b.percent} max={100} aria-label={name} />
                    </li>
                  );
                })}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}

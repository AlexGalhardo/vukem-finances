import { Head, router, usePage } from "@inertiajs/react";
import { type FormEvent, type ReactNode, useState } from "react";
import BarChart from "../components/charts/BarChart";
import ChartWithTable from "../components/charts/ChartWithTable";
import DonutChart from "../components/charts/DonutChart";
import LineChart from "../components/charts/LineChart";
import ExportMenu from "../components/ExportMenu";
import PageHeader from "../components/PageHeader";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Field, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { canExportJson, transactionsCsvUrl, workspaceJsonUrl } from "../lib/exports";
import { type Key, useFinanceI18n } from "../lib/i18n";
import { percentText, periodQuery, presets } from "../lib/reports";
import type { CategoryRow, Page, SharedProps } from "../page-props";
import { useExportAction } from "../shortcuts/useExportAction";

const presetKeys: Record<(typeof presets)[number], Key> = {
  this_month: "rptThisMonth",
  last_month: "rptLastMonth",
  last_3_months: "rptLast3Months",
  last_12_months: "rptLast12Months",
  this_year: "rptThisYear",
  custom: "rptCustom",
};

function fill(text: string, values: Record<string, string>): string {
  return Object.entries(values).reduce((out, [key, value]) => out.replaceAll(`{${key}}`, value), text);
}

function num(children: ReactNode, head = false) {
  return head ? (
    <TableHead className="text-right">{children}</TableHead>
  ) : (
    <TableCell className="text-right tabular-nums">{children}</TableCell>
  );
}

export default function Reports(props: Page<"Reports">) {
  const { t, money, locale, date, month: monthText } = useFinanceI18n();
  const { period, currency, cashFlow, expenseCategories, incomeCategories, accounts, tags, netWorth, compare } = props;
  const role = usePage<SharedProps>().props.workspace?.role;
  const [preset, setPreset] = useState<string>(period.preset);
  const [from, setFrom] = useState(period.from);
  const [to, setTo] = useState(period.to);

  const visit = (nextPreset: string, nextFrom = from, nextTo = to): void => {
    router.get("/reports", periodQuery(nextPreset, nextFrom, nextTo), { preserveScroll: true, preserveState: true });
  };
  const submit = (e: FormEvent): void => {
    e.preventDefault();
    visit(preset);
  };

  const csvFilters = { from: period.from, to: period.to };
  useExportAction(() => transactionsCsvUrl(csvFilters));

  const categoryName = (row: CategoryRow): string => (row.uncategorized ? t("rptUncategorized") : row.name);
  const categoryTable = (report: typeof expenseCategories, title: string) => (
    <Table>
      <TableCaption className="sr-only">{title}</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>{t("rptCategory")}</TableHead>
          {num(t("rptTotal"), true)}
          {num(t("rptShare"), true)}
        </TableRow>
      </TableHeader>
      <TableBody>
        {report.rows.flatMap((row) => [
          <TableRow key={row.uid || "uncategorized"}>
            <TableCell className="font-medium">{categoryName(row)}</TableCell>
            {num(money(row.cents, currency))}
            {num(percentText(row.shareBps, locale))}
          </TableRow>,
          ...row.children.map((child) => (
            <TableRow key={child.uid}>
              <TableCell className="pl-6 text-muted-foreground">{`– ${child.name}`}</TableCell>
              {num(money(child.cents, currency))}
              {num(percentText(child.shareBps, locale))}
            </TableRow>
          )),
        ])}
        <TableRow>
          <TableHead>{t("rptTotal")}</TableHead>
          {num(money(report.totalCents, currency), true)}
          {num("", true)}
        </TableRow>
      </TableBody>
    </Table>
  );

  const categorySection = (report: typeof expenseCategories, title: string) => (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {report.rows.length === 0 ? (
          <p className="text-muted-foreground">{t("rptEmpty")}</p>
        ) : (
          <ChartWithTable
            title={title}
            summary={`${title}: ${money(report.totalCents, currency)}`}
            chart={
              <DonutChart
                slices={report.rows.map((row) => ({ category: categoryName(row), cents: row.cents }))}
                currency={currency}
              />
            }
            table={categoryTable(report, title)}
          />
        )}
      </CardContent>
    </Card>
  );

  const sumRows = (
    rows: { uid: string; name: string; incomeCents: number; expenseCents: number; netCents: number }[],
  ) =>
    rows.map((row) => (
      <TableRow key={row.uid}>
        <TableCell className="font-medium">{row.name}</TableCell>
        {num(money(row.incomeCents, currency))}
        {num(money(row.expenseCents, currency))}
        {num(money(row.netCents, currency))}
      </TableRow>
    ));
  const sumHead = (first: string) => (
    <TableHeader>
      <TableRow>
        <TableHead>{first}</TableHead>
        {num(t("rptIncome"), true)}
        {num(t("rptExpense"), true)}
        {num(t("rptNet"), true)}
      </TableRow>
    </TableHeader>
  );

  const flowTotals = cashFlow.totals;
  const compareLabel: Record<string, Key> = { income: "rptIncome", expense: "rptExpense", net: "rptNet" };

  return (
    <>
      <Head title={t("reports")} />
      <PageHeader
        title={t("reports")}
        description={`${date(period.from)} – ${date(period.to)} · ${t("rptDescription")}`}
        actions={
          <ExportMenu
            csvUrl={transactionsCsvUrl(csvFilters)}
            csvExcelUrl={transactionsCsvUrl(csvFilters, { excel: true })}
            jsonUrl={canExportJson(role) ? workspaceJsonUrl : undefined}
            onPrint={() => window.print()}
          />
        }
      />

      <form onSubmit={submit} className="flex flex-wrap items-end gap-3" data-print="hide">
        <FieldGroup className="flex flex-row flex-wrap items-end gap-3">
          <Field className="w-auto">
            <FieldLabel htmlFor="report-period">{t("rptPeriod")}</FieldLabel>
            <NativeSelect
              id="report-period"
              value={preset}
              onChange={(e) => {
                setPreset(e.target.value);
                if (e.target.value !== "custom") visit(e.target.value);
              }}
            >
              {presets.map((p) => (
                <NativeSelectOption key={p} value={p}>
                  {t(presetKeys[p])}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          {preset === "custom" && (
            <>
              <Field className="w-auto">
                <FieldLabel htmlFor="report-from">{t("rptFrom")}</FieldLabel>
                <Input id="report-from" type="date" value={from} onChange={(e) => setFrom(e.target.value)} required />
              </Field>
              <Field className="w-auto">
                <FieldLabel htmlFor="report-to">{t("rptTo")}</FieldLabel>
                <Input id="report-to" type="date" value={to} onChange={(e) => setTo(e.target.value)} required />
              </Field>
              <Button type="submit">{t("rptApply")}</Button>
            </>
          )}
        </FieldGroup>
      </form>

      <Tabs defaultValue="cashflow">
        <TabsList data-print="hide" aria-label={t("reports")}>
          <TabsTrigger value="cashflow">{t("rptTabCashFlow")}</TabsTrigger>
          <TabsTrigger value="categories">{t("rptTabCategories")}</TabsTrigger>
          <TabsTrigger value="accounts">{t("rptTabAccounts")}</TabsTrigger>
          <TabsTrigger value="networth">{t("rptTabNetWorth")}</TabsTrigger>
          <TabsTrigger value="compare">{t("rptTabCompare")}</TabsTrigger>
        </TabsList>

        <TabsContent value="cashflow" forceMount data-report-panel className="data-[state=inactive]:hidden">
          <Card>
            <CardHeader>
              <CardTitle>{t("rptTabCashFlow")}</CardTitle>
            </CardHeader>
            <CardContent>
              <ChartWithTable
                title={t("rptTabCashFlow")}
                summary={fill(t("rptCashFlowSummary"), {
                  income: money(flowTotals.incomeCents, currency),
                  expense: money(flowTotals.expenseCents, currency),
                  net: money(flowTotals.netCents, currency),
                  from: date(period.from),
                  to: date(period.to),
                })}
                chart={<BarChart points={cashFlow.months} currency={currency} />}
                table={
                  <Table>
                    <TableCaption className="sr-only">{t("rptTabCashFlow")}</TableCaption>
                    {sumHead(t("rptMonth"))}
                    <TableBody>
                      {cashFlow.months.map((m) => (
                        <TableRow key={m.month}>
                          <TableCell className="font-medium">{monthText(m.month)}</TableCell>
                          {num(money(m.incomeCents, currency))}
                          {num(money(m.expenseCents, currency))}
                          {num(money(m.netCents, currency))}
                        </TableRow>
                      ))}
                      <TableRow>
                        <TableHead>{t("rptTotal")}</TableHead>
                        {num(money(flowTotals.incomeCents, currency), true)}
                        {num(money(flowTotals.expenseCents, currency), true)}
                        {num(money(flowTotals.netCents, currency), true)}
                      </TableRow>
                    </TableBody>
                  </Table>
                }
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="categories" forceMount data-report-panel className="data-[state=inactive]:hidden">
          <div className="flex flex-col gap-5">
            {categorySection(expenseCategories, t("rptKindExpense"))}
            {categorySection(incomeCategories, t("rptKindIncome"))}
          </div>
        </TabsContent>

        <TabsContent value="accounts" forceMount data-report-panel className="data-[state=inactive]:hidden">
          <div className="flex flex-col gap-5">
            <Card>
              <CardHeader>
                <CardTitle>{t("rptAccount")}</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto">
                {accounts.rows.length === 0 ? (
                  <p className="text-muted-foreground">{t("rptEmpty")}</p>
                ) : (
                  <Table>
                    <TableCaption className="sr-only">{t("rptAccount")}</TableCaption>
                    {sumHead(t("rptAccount"))}
                    <TableBody>{sumRows(accounts.rows)}</TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>{t("rptTag")}</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 overflow-x-auto">
                {tags.rows.length === 0 ? (
                  <p className="text-muted-foreground">{t("rptEmpty")}</p>
                ) : (
                  <>
                    <Table>
                      <TableCaption className="sr-only">{t("rptTag")}</TableCaption>
                      {sumHead(t("rptTag"))}
                      <TableBody>{sumRows(tags.rows)}</TableBody>
                    </Table>
                    <p className="text-sm text-muted-foreground">{t("rptTagNote")}</p>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="networth" forceMount data-report-panel className="data-[state=inactive]:hidden">
          <Card>
            <CardHeader>
              <CardTitle>{t("rptTabNetWorth")}</CardTitle>
            </CardHeader>
            <CardContent>
              <ChartWithTable
                title={t("rptTabNetWorth")}
                summary={fill(t("rptNetWorthSummary"), {
                  first: money(netWorth.points[0]?.balanceCents ?? 0, currency),
                  last: money(netWorth.points[netWorth.points.length - 1]?.balanceCents ?? 0, currency),
                  n: String(netWorth.points.length),
                })}
                chart={<LineChart points={netWorth.points} currency={currency} />}
                table={
                  <Table>
                    <TableCaption className="sr-only">{t("rptTabNetWorth")}</TableCaption>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("rptMonth")}</TableHead>
                        {num(t("rptBalance"), true)}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {netWorth.points.map((p) => (
                        <TableRow key={p.month}>
                          <TableCell className="font-medium">{monthText(p.month)}</TableCell>
                          {num(money(p.balanceCents, currency))}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                }
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="compare" forceMount data-report-panel className="data-[state=inactive]:hidden">
          <Card>
            <CardHeader>
              <CardTitle>{t("rptTabCompare")}</CardTitle>
              <p className="text-sm text-muted-foreground">
                {fill(t("rptComparedWith"), { from: date(compare.previous.from), to: date(compare.previous.to) })}
              </p>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              <Table>
                <TableCaption className="sr-only">{t("rptTabCompare")}</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("rptPeriod")}</TableHead>
                    {num(t("rptCurrent"), true)}
                    {num(t("rptPrevious"), true)}
                    {num(t("rptDifference"), true)}
                    {num(t("rptChange"), true)}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {compare.rows.map((row) => (
                    <TableRow key={row.key}>
                      <TableCell className="font-medium">{t(compareLabel[row.key] ?? "rptNet")}</TableCell>
                      {num(money(row.currentCents, currency))}
                      {num(money(row.previousCents, currency))}
                      {num(money(row.diffCents, currency))}
                      {num(row.notApplicable ? t("rptNotApplicable") : percentText(row.changeBps, locale, true))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </>
  );
}

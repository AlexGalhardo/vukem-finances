import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { FileUp } from "lucide-react";
import { type FormEvent, useId } from "react";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { type Key, useFinanceI18n } from "../lib/i18n";
import type { ImportView, MappedPreview, Page, SharedProps } from "../page-props";

type PreviewProps = NonNullable<Page<"Import">["preview"]>;

/** Picks the header that looks like what the column is for. */
function guess(headers: string[], words: string[]): string {
  return headers.find((h) => words.some((w) => h.toLowerCase().includes(w))) ?? headers[0] ?? "";
}

/** "2006-01-02" as the person knows it: YYYY-MM-DD. */
function formatLayout(layout: string): string {
  return layout.replace("2006", "YYYY").replace("01", "MM").replace("02", "DD");
}

const statusKeys: Record<string, Key> = {
  uploaded: "statusUploaded",
  queued: "statusQueued",
  done: "statusDone",
  failed: "statusFailed",
};

function Summary({ mapped }: { mapped: MappedPreview }) {
  const { t, date } = useFinanceI18n();
  const items: { label: Key; value: number }[] = [
    { label: "rowsRead", value: mapped.read },
    { label: "rowsNew", value: mapped.new },
    { label: "rowsDuplicate", value: mapped.duplicates },
    { label: "rowsFailed", value: mapped.failed },
  ];
  return (
    <div className="flex flex-col gap-2">
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {items.map((item) => (
          <div key={item.label} className="rounded-lg border p-3">
            <dt className="text-sm text-muted-foreground">{t(item.label)}</dt>
            <dd className="text-2xl font-semibold tabular-nums">{item.value}</dd>
          </div>
        ))}
      </dl>
      {mapped.from !== "" && (
        <p className="text-sm text-muted-foreground">
          {t("dateRange")}: {date(mapped.from)} – {date(mapped.to)}
        </p>
      )}
    </div>
  );
}

function MappedTable({ mapped, currency }: { mapped: MappedPreview; currency: string }) {
  const { t, locale, date } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  return (
    <div className="flex flex-col gap-2">
      <p className="text-sm text-muted-foreground">
        {t("previewFirstRows")} {mapped.rows.length} / {mapped.read}
      </p>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("line")}</TableHead>
            <TableHead>{t("date")}</TableHead>
            <TableHead>{t("description")}</TableHead>
            <TableHead>{t("category")}</TableHead>
            <TableHead className="text-end">{t("amount")}</TableHead>
            <TableHead>{t("rowStatus")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {mapped.rows.map((row) => (
            <TableRow key={row.line}>
              <TableCell className="tabular-nums text-muted-foreground">{row.line}</TableCell>
              <TableCell className="whitespace-nowrap">{row.problem ? "—" : date(row.date)}</TableCell>
              <TableCell>{row.description || "—"}</TableCell>
              <TableCell>{row.category || "—"}</TableCell>
              <TableCell className="text-end">
                {row.problem ? (
                  "—"
                ) : (
                  <MoneyText cents={row.cents} currency={currency} locale={locale} sign="always" labels={labels} />
                )}
              </TableCell>
              <TableCell>
                {row.problem ? (
                  <Badge variant="destructive">
                    {t("rowStatusFailed")}: {row.problem}
                  </Badge>
                ) : row.duplicate ? (
                  <Badge variant="secondary">{t("rowStatusDuplicate")}</Badge>
                ) : (
                  <Badge variant="outline">{t("rowStatusNew")}</Badge>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function Statement({
  preview,
  dateFormats,
  currency,
}: {
  preview: PreviewProps;
  dateFormats: string[];
  currency: string;
}) {
  const { t } = useFinanceI18n();
  const isCSV = preview.format === "csv";
  const mapped = preview.mapped ?? null;
  const allowId = useId();
  const form = useForm({
    date: guess(preview.headers, ["date", "data"]),
    description: guess(preview.headers, ["descr", "memo", "histor", "name"]),
    amount: guess(preview.headers, ["amount", "valor", "value"]),
    category: "",
    dateFormat: dateFormats[0] ?? "2006-01-02",
    invert: "false",
    allowDuplicates: "false",
  });

  const columns = preview.headers.map((h) => (
    <NativeSelectOption key={h} value={h}>
      {h}
    </NativeSelectOption>
  ));
  const allow = form.data.allowDuplicates === "true";
  const importing = mapped ? (allow ? mapped.new + mapped.duplicates : mapped.new) : 0;

  function previewIt(event: FormEvent): void {
    event.preventDefault();
    form.post(`/import/${preview.importId}/preview`, { preserveScroll: true });
  }

  function confirm(): void {
    form.post(`/import/${preview.importId}`);
  }

  const columnField = (key: "date" | "description" | "amount", label: Key) => (
    <Field data-invalid={!!form.errors[key]}>
      <FieldLabel htmlFor={`map-${key}`}>{t(label)}</FieldLabel>
      <NativeSelect id={`map-${key}`} value={form.data[key]} onChange={(e) => form.setData(key, e.target.value)}>
        {columns}
      </NativeSelect>
      <FieldError>{form.errors[key]}</FieldError>
    </Field>
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {preview.filename} <Badge variant="secondary">{preview.format.toUpperCase()}</Badge>
        </CardTitle>
        <CardDescription>{t(isCSV ? "mapDescription" : "ofxDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        {isCSV && (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    {preview.headers.map((h) => (
                      <TableHead key={h}>{h}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {preview.rows.map((row, i) => (
                    // biome-ignore lint/suspicious/noArrayIndexKey: a static CSV sample, never reordered
                    <TableRow key={i}>
                      {row.map((cell, j) => (
                        // biome-ignore lint/suspicious/noArrayIndexKey: a static CSV sample, never reordered
                        <TableCell key={j}>{cell}</TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <form onSubmit={previewIt} className="flex flex-col gap-4">
              <FieldGroup className="grid gap-4 sm:grid-cols-3">
                {columnField("date", "dateColumn")}
                {columnField("description", "descriptionColumn")}
                {columnField("amount", "amountColumn")}
                <Field>
                  <FieldLabel htmlFor="map-category">{t("categoryColumn")}</FieldLabel>
                  <NativeSelect
                    id="map-category"
                    value={form.data.category}
                    onChange={(e) => form.setData("category", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("none")}</NativeSelectOption>
                    {columns}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="map-format">{t("dateFormat")}</FieldLabel>
                  <NativeSelect
                    id="map-format"
                    value={form.data.dateFormat}
                    onChange={(e) => form.setData("dateFormat", e.target.value)}
                  >
                    {dateFormats.map((f) => (
                      <NativeSelectOption key={f} value={f}>
                        {formatLayout(f)}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field orientation="horizontal" className="items-end pb-2">
                  <input
                    id="map-invert"
                    type="checkbox"
                    checked={form.data.invert === "true"}
                    onChange={(e) => form.setData("invert", String(e.target.checked))}
                  />
                  <FieldLabel htmlFor="map-invert">{t("positiveAreExpenses")}</FieldLabel>
                </Field>
              </FieldGroup>
              <div>
                <Button type="submit" variant={mapped ? "outline" : "default"} disabled={form.processing}>
                  {t(mapped ? "refreshPreview" : "previewImport")}
                </Button>
              </div>
            </form>
          </>
        )}

        {mapped && (
          <>
            <Summary mapped={mapped} />
            <MappedTable mapped={mapped} currency={currency} />
            {mapped.failures.length > 0 && (
              <details className="rounded-lg border p-3">
                <summary className="cursor-pointer text-sm font-medium">
                  {t("failedRows")} ({mapped.failed})
                </summary>
                <ul className="mt-2 text-sm">
                  {mapped.failures.map((f) => (
                    <li key={f.line}>
                      {t("line")} {f.line}: {f.reason}
                    </li>
                  ))}
                </ul>
              </details>
            )}
            <Field orientation="horizontal">
              <input
                id={allowId}
                type="checkbox"
                checked={allow}
                onChange={(e) => form.setData("allowDuplicates", String(e.target.checked))}
              />
              <FieldLabel htmlFor={allowId}>{t("allowDuplicates")}</FieldLabel>
            </Field>
            <FieldDescription>{t("allowDuplicatesHelp")}</FieldDescription>
            <div className="flex flex-wrap items-center gap-3">
              <Button type="button" disabled={form.processing || importing === 0} onClick={confirm}>
                <FileUp data-icon="inline-start" />
                {t("importRows")} ({importing})
              </Button>
              <Button asChild variant="outline">
                <Link href="/import">{t("startOver")}</Link>
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function ImportRow({ item }: { item: ImportView }) {
  const { t } = useFinanceI18n();
  const status = statusKeys[item.status];
  return (
    <li className="flex flex-col gap-1 py-3 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <strong>{item.filename}</strong>
        <Badge variant="secondary">{item.format.toUpperCase()}</Badge>
        <Badge variant={item.status === "failed" ? "destructive" : "outline"}>{status ? t(status) : item.status}</Badge>
      </div>
      {item.status === "done" && (
        <p className="tabular-nums text-muted-foreground">
          {item.imported} {t("reportImported")}, {item.duplicates} {t("reportDuplicates")}, {item.failed}{" "}
          {t("reportFailed")}
        </p>
      )}
      {item.status === "failed" && item.error && <p className="text-destructive">{item.error}</p>}
      {item.failures.length > 0 && (
        <details>
          <summary className="cursor-pointer">
            {t("failedRows")} ({item.failures.length}
            {item.failed > item.failures.length ? `/${item.failed}` : ""})
          </summary>
          <ul className="mt-1">
            {item.failures.map((f) => (
              <li key={f.line}>
                {t("line")} {f.line}: {f.reason}
              </li>
            ))}
          </ul>
        </details>
      )}
    </li>
  );
}

export default function Import({ accounts, imports, preview, dateFormats, enabled }: Page<"Import">) {
  const { t } = useFinanceI18n();
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const form = useForm<{ accountId: string; file: File | null }>({
    accountId: String((accounts.find((a) => !a.archived_at) ?? accounts[0])?.id ?? ""),
    file: null,
  });

  function submit(event: FormEvent): void {
    event.preventDefault();
    form.post("/import", { forceFormData: true });
  }

  return (
    <>
      <Head title={t("importTitle")} />
      <PageHeader title={t("importTitle")} description={t("importDescription")} />
      {!enabled && (
        <p role="note" className="rounded-md border px-4 py-3">
          {t("importProOnly")}{" "}
          <Link href="/billing" className="font-medium underline">
            {t("importTrial")}
          </Link>
        </p>
      )}
      {preview ? (
        <Statement key={preview.importId} preview={preview} dateFormats={dateFormats} currency={currency} />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>{t("uploadStatement")}</CardTitle>
            <CardDescription>{t("uploadHelp")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="flex flex-col gap-4">
              <FieldGroup className="grid gap-4 sm:grid-cols-3">
                <Field data-invalid={!!form.errors.accountId}>
                  <FieldLabel htmlFor="import-account">{t("account")}</FieldLabel>
                  <NativeSelect
                    id="import-account"
                    value={form.data.accountId}
                    onChange={(e) => form.setData("accountId", e.target.value)}
                  >
                    {accounts
                      .filter((a) => !a.archived_at)
                      .map((a) => (
                        <NativeSelectOption key={a.id} value={a.id}>
                          {a.name}
                        </NativeSelectOption>
                      ))}
                  </NativeSelect>
                  <FieldError>{form.errors.accountId}</FieldError>
                </Field>
                <Field className="sm:col-span-2" data-invalid={!!form.errors.file}>
                  <FieldLabel htmlFor="import-file">{t("statementFile")}</FieldLabel>
                  <Input
                    id="import-file"
                    type="file"
                    accept=".csv,.ofx,.qfx,text/csv,application/x-ofx"
                    required
                    onChange={(e) => form.setData("file", e.target.files?.[0] ?? null)}
                  />
                  <FieldError>{form.errors.file}</FieldError>
                </Field>
              </FieldGroup>
              <div>
                <Button type="submit" disabled={form.processing || !form.data.file}>
                  {t("upload")}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      <Card>
        <CardHeader>
          <CardTitle>{t("latestImports")}</CardTitle>
        </CardHeader>
        <CardContent>
          {imports.length === 0 ? (
            <p className="text-muted-foreground">{t("nothingImported")}</p>
          ) : (
            <ul className="divide-y">
              {imports.map((item) => (
                <ImportRow key={item.id} item={item} />
              ))}
            </ul>
          )}
          <Button
            type="button"
            variant="link"
            className="mt-2 px-0"
            onClick={() => router.reload({ only: ["imports"] })}
          >
            {t("refresh")}
          </Button>
        </CardContent>
      </Card>
    </>
  );
}

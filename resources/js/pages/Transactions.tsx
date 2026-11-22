import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { ArrowLeftRight, Pencil, Search, Trash2, Upload } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import { Mark } from "../components/categories/look";
import PageHeader from "../components/PageHeader";
import Pagination from "../components/Pagination";
import { centsToInput } from "../components/transactions/entryForm";
import { useQuickAdd } from "../components/transactions/QuickAdd";
import { setSearchFocus } from "../components/transactions/searchFocus";
import TagsField from "../components/transactions/TagsField";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "../components/ui/empty";
import { Field, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { transactionsCsvUrl } from "../lib/exports";
import { useFinanceI18n } from "../lib/i18n";
import type { Page, SharedProps } from "../page-props";
import { useExportAction } from "../shortcuts/useExportAction";
/** The query string of the list: only the filters that are set, with their URL names. */
function queryOf(filters: Record<string, string>, page = 1): Record<string, string> {
  const query: Record<string, string> = {};
  for (const [key, value] of Object.entries(filters)) if (value !== "") query[key] = value;
  if (page > 1) query.page = String(page);
  return query;
}

export default function Transactions({
  transactions,
  accounts,
  categories,
  tags,
  filters,
  today,
  total,
  page: pageNumber,
  pageSize,
}: Page<"Transactions">) {
  const { t, money, date } = useFinanceI18n();
  const page = usePage<SharedProps>();
  const currency = page.props.workspace?.current.currency ?? "USD";
  const { openTransaction, openTransfer, canRecord } = useQuickAdd();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<number | null>(null);
  const asked = useRef(false);
  // `x` downloads the CSV of the filters the list is showing (the applied ones, not the draft in the form).
  useExportAction(() =>
    transactionsCsvUrl({
      q: filters.q,
      category: filters.categoryId,
      tag: filters.tagId,
      account: filters.accountId,
      kind: filters.kind,
      from: filters.from,
      to: filters.to,
      min: filters.min,
      max: filters.max,
    }),
  );
  const searchRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft] = useState({
    q: filters.q,
    category: filters.categoryId,
    tag: filters.tagId,
    account: filters.accountId,
    kind: filters.kind,
    from: filters.from,
    to: filters.to,
    min: filters.min,
    max: filters.max,
  });
  const setFilter = (key: keyof typeof draft, value: string): void =>
    setDraft((current) => ({ ...current, [key]: value }));

  // The `search-transactions` action (`/`) focuses the search field here; /transactions#search does it on arrival.
  useEffect(() => {
    const focus = (): void => searchRef.current?.focus();
    // The shell moves focus to the page heading one frame after a navigation: arrive after it.
    if (window.location.hash === "#search") requestAnimationFrame(() => requestAnimationFrame(focus));
    return setSearchFocus(focus);
  }, []);

  // The dashboard links here with ?create=1: open the shared new transaction dialog once.
  useEffect(() => {
    if (asked.current || new URLSearchParams(window.location.search).get("create") !== "1") return;
    asked.current = true;
    openTransaction("expense");
  }, [openTransaction]);
  const form = useForm({
    accountId: String(accounts[0]?.id ?? ""),
    categoryId: "",
    kind: "expense",
    amount: "",
    description: "",
    date: today,
    tags: [] as string[],
  });
  function edit(item: (typeof transactions)[number]) {
    if (item.transferId !== null && item.transferAccountId !== null) {
      const outgoing = item.kind === "expense";
      openTransfer({
        legId: item.id,
        draft: {
          amount: centsToInput(item.amount_cents),
          fromAccountId: String(outgoing ? item.account_id : item.transferAccountId),
          toAccountId: String(outgoing ? item.transferAccountId : item.account_id),
          description: item.description,
          date: item.occurred_on.slice(0, 10),
        },
        accounts: accounts.map((a) => ({ id: a.id, name: a.name, kind: a.kind })),
      });
      return;
    }
    setEditing(item.id);
    form.clearErrors();
    form.setData({
      accountId: String(item.account_id),
      categoryId: item.categoryId ? String(item.categoryId) : "",
      kind: item.kind,
      amount: centsToInput(item.amount_cents),
      description: item.description,
      date: item.occurred_on.slice(0, 10),
      tags: item.tags.map((tag) => tag.name),
    });
    setOpen(true);
  }
  function submit(e: FormEvent) {
    e.preventDefault();
    const options = {
      preserveState: true as const,
      onSuccess: () => {
        setOpen(false);
        form.reset();
      },
      onError: () => setOpen(true),
    };
    if (editing !== null) form.patch(`/transactions/${editing}`, options);
  }
  function applyFilters(event: FormEvent): void {
    event.preventDefault();
    router.get("/transactions", queryOf(draft), { preserveState: true, replace: true });
  }
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const pageHref = (target: number): string => {
    const query = new URLSearchParams(queryOf(draft, target)).toString();
    return query === "" ? "/transactions" : `/transactions?${query}`;
  };
  const kindCategories = categories.filter((c) => c.kind === form.data.kind);
  return (
    <>
      <Head title={t("transactions")} />
      <PageHeader
        title={t("transactions")}
        description={t("transactionDescription")}
        actions={
          canRecord && (
            <Button variant="outline" onClick={() => openTransfer()}>
              <ArrowLeftRight data-icon="inline-start" />
              {t("newTransfer")}
            </Button>
          )
        }
      />
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("editTransaction")}</DialogTitle>
            <DialogDescription>{t("modalDescription")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={submit} className="flex flex-col gap-6">
            <FieldGroup>
              <Field data-invalid={!!form.errors.kind}>
                <FieldLabel htmlFor="transaction-kind">{t("kind")}</FieldLabel>
                <NativeSelect
                  id="transaction-kind"
                  value={form.data.kind}
                  aria-invalid={!!form.errors.kind}
                  onChange={(e) => form.setData({ ...form.data, kind: e.target.value, categoryId: "" })}
                >
                  <NativeSelectOption value="expense">{t("expense")}</NativeSelectOption>
                  <NativeSelectOption value="income">{t("income")}</NativeSelectOption>
                </NativeSelect>
                <FieldError>{form.errors.kind}</FieldError>
              </Field>
              {(["amount", "date", "description"] as const).map((key) => (
                <Field key={key} data-invalid={!!form.errors[key]}>
                  <FieldLabel htmlFor={`transaction-${key}`}>{t(key)}</FieldLabel>
                  <Input
                    id={`transaction-${key}`}
                    type={key === "date" ? "date" : "text"}
                    inputMode={key === "amount" ? "decimal" : undefined}
                    required={key !== "description"}
                    placeholder={key === "amount" ? "12.50" : undefined}
                    value={form.data[key]}
                    aria-invalid={!!form.errors[key]}
                    onChange={(e) => form.setData(key, e.target.value)}
                  />
                  <FieldError>{form.errors[key]}</FieldError>
                </Field>
              ))}
              <Field data-invalid={!!form.errors.accountId}>
                <FieldLabel htmlFor="transaction-account">{t("account")}</FieldLabel>
                <NativeSelect
                  id="transaction-account"
                  value={form.data.accountId}
                  required
                  aria-invalid={!!form.errors.accountId}
                  onChange={(e) => form.setData("accountId", e.target.value)}
                >
                  {accounts.map((a) => (
                    <NativeSelectOption key={a.id} value={a.id}>
                      {a.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <FieldError>{form.errors.accountId}</FieldError>
              </Field>
              <Field data-invalid={!!form.errors.categoryId}>
                <FieldLabel htmlFor="transaction-category">{t("category")}</FieldLabel>
                <NativeSelect
                  id="transaction-category"
                  value={form.data.categoryId}
                  aria-invalid={!!form.errors.categoryId}
                  onChange={(e) => form.setData("categoryId", e.target.value)}
                >
                  <NativeSelectOption value="">{t("none")}</NativeSelectOption>
                  {kindCategories
                    .filter((c) => !c.archived_at || String(c.id) === form.data.categoryId)
                    .map((c) => (
                      <NativeSelectOption key={c.id} value={c.id}>
                        {c.name}
                      </NativeSelectOption>
                    ))}
                </NativeSelect>
                <FieldError>{form.errors.categoryId}</FieldError>
              </Field>
              <TagsField
                tags={form.data.tags}
                onChange={(next) => form.setData("tags", next)}
                suggestions={tags.map((tag) => tag.name)}
                error={form.errors.tags}
              />
            </FieldGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={form.processing || accounts.length === 0}>
                {t(form.processing ? "working" : "save")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Card>
        <CardHeader>
          <CardTitle>{t("activity")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <search>
            <form onSubmit={applyFilters} aria-label={t("filters")} className="flex flex-col gap-4">
              <Field>
                <FieldLabel htmlFor="filter-q">{t("searchLabel")}</FieldLabel>
                <div className="relative">
                  <Search
                    aria-hidden="true"
                    className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                  />
                  <Input
                    id="filter-q"
                    ref={searchRef}
                    name="q"
                    type="search"
                    className="ps-9"
                    maxLength={200}
                    autoComplete="off"
                    autoCorrect="off"
                    autoCapitalize="none"
                    spellCheck={false}
                    enterKeyHint="search"
                    placeholder={t("searchPlaceholder")}
                    value={draft.q}
                    onChange={(e) => setFilter("q", e.target.value)}
                  />
                </div>
              </Field>
              <FieldGroup className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <Field>
                  <FieldLabel htmlFor="filter-category">{t("category")}</FieldLabel>
                  <NativeSelect
                    id="filter-category"
                    name="category"
                    value={draft.category}
                    onChange={(e) => setFilter("category", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("any")}</NativeSelectOption>
                    {categories.map((c) => (
                      <NativeSelectOption key={c.id} value={c.id}>
                        {c.parent_id.Valid ? "– " : ""}
                        {c.name}
                        {c.archived_at ? ` (${t("archived")})` : ""}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="filter-tag">{t("tag")}</FieldLabel>
                  <NativeSelect
                    id="filter-tag"
                    name="tag"
                    value={draft.tag}
                    onChange={(e) => setFilter("tag", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("any")}</NativeSelectOption>
                    {tags.map((tag) => (
                      <NativeSelectOption key={tag.id} value={tag.id}>
                        {tag.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="filter-account">{t("account")}</FieldLabel>
                  <NativeSelect
                    id="filter-account"
                    name="account"
                    value={draft.account}
                    onChange={(e) => setFilter("account", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("any")}</NativeSelectOption>
                    {accounts.map((a) => (
                      <NativeSelectOption key={a.id} value={a.id}>
                        {a.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="filter-kind">{t("kind")}</FieldLabel>
                  <NativeSelect
                    id="filter-kind"
                    name="kind"
                    value={draft.kind}
                    onChange={(e) => setFilter("kind", e.target.value)}
                  >
                    <NativeSelectOption value="">{t("any")}</NativeSelectOption>
                    <NativeSelectOption value="expense">{t("expense")}</NativeSelectOption>
                    <NativeSelectOption value="income">{t("income")}</NativeSelectOption>
                  </NativeSelect>
                </Field>
                {(["from", "to"] as const).map((key) => (
                  <Field key={key}>
                    <FieldLabel htmlFor={`filter-${key}`}>{t(key)}</FieldLabel>
                    <Input
                      id={`filter-${key}`}
                      name={key}
                      type="date"
                      value={draft[key]}
                      onChange={(e) => setFilter(key, e.target.value)}
                    />
                  </Field>
                ))}
                {(["min", "max"] as const).map((key) => (
                  <Field key={key}>
                    <FieldLabel htmlFor={`filter-${key}`}>{t(key === "min" ? "minAmount" : "maxAmount")}</FieldLabel>
                    <Input
                      id={`filter-${key}`}
                      name={key}
                      type="text"
                      inputMode="decimal"
                      autoComplete="off"
                      placeholder="0.00"
                      value={draft[key]}
                      onChange={(e) => setFilter(key, e.target.value)}
                    />
                  </Field>
                ))}
              </FieldGroup>
              <div className="flex flex-wrap items-center gap-2">
                <Button type="submit">{t("applyFilters")}</Button>
                <Button type="button" variant="outline" onClick={() => router.get("/transactions")}>
                  {t("clearFilters")}
                </Button>
                <p role="status" className="ms-auto text-sm text-muted-foreground tabular-nums">
                  {total} {t("transactionsWord")}
                </p>
              </div>
            </form>
          </search>
          {transactions.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ArrowLeftRight />
                </EmptyMedia>
                <EmptyTitle>{t("emptyTransactions")}</EmptyTitle>
                <EmptyDescription>{t("emptyTransactionsDescription")}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant="outline" asChild>
                  <Link href="/import">
                    <Upload data-icon="inline-start" />
                    {t("import")}
                  </Link>
                </Button>
              </EmptyContent>
            </Empty>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  {(["date", "description", "category", "account", "amount", "actions"] as const).map((key) => (
                    <TableHead key={key}>{t(key)}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {transactions.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="whitespace-nowrap text-muted-foreground">{date(item.occurred_on)}</TableCell>
                    <TableCell className="font-medium">
                      {item.transferId !== null ? (
                        <span className="flex flex-wrap items-center gap-2">
                          <Badge variant="secondary">{t("transfer")}</Badge>
                          <span>
                            {t(item.kind === "expense" ? "transferTo" : "transferFrom")} {item.transferAccountName}
                          </span>
                          {item.description && (
                            <span className="font-normal text-muted-foreground">{item.description}</span>
                          )}
                        </span>
                      ) : (
                        <span className="flex flex-wrap items-center gap-2">
                          <span>{item.description || t(item.kind === "income" ? "income" : "expense")}</span>
                          {item.tags.map((tag) => (
                            <Badge key={tag.id} variant="outline" className="gap-1 font-normal">
                              <Mark color={tag.color} icon="tag" />
                              {tag.name}
                            </Badge>
                          ))}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>{item.categoryName || "—"}</TableCell>
                    <TableCell>{item.accountName}</TableCell>
                    <TableCell className="whitespace-nowrap tabular-nums">
                      {item.kind === "income" ? "+" : "−"}
                      {money(item.amount_cents, currency)}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          disabled={!canRecord}
                          onClick={() => edit(item)}
                          aria-label={`${t("edit")} ${item.description}`}
                        >
                          <Pencil />
                        </Button>
                        <ConfirmDelete
                          href={item.transferId !== null ? `/transfers/${item.id}` : `/transactions/${item.id}`}
                          name={item.description || t(item.kind === "income" ? "income" : "expense")}
                          trigger={
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              aria-label={`${t("delete")} ${item.description}`}
                            >
                              <Trash2 />
                            </Button>
                          }
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <Pagination page={pageNumber} pageCount={pageCount} hrefFor={pageHref} />
        </CardContent>
      </Card>
    </>
  );
}

import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { HelpHint } from "../help/HelpHint";
import { Archive, ArchiveRestore, CreditCard, Pencil, Plus, Trash2, Wallet } from "lucide-react";
import { type FormEvent, useState } from "react";
import AmountInput from "../components/AmountInput";
import ConfirmDelete from "../components/ConfirmDelete";
import DataTable, { type DataTableColumn } from "../components/DataTable";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import { centsToInput } from "../components/transactions/entryForm";
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
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "../components/ui/empty";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { type Key, useFinanceI18n } from "../lib/i18n";
import type { Account, AccountBalance, Page, SharedProps } from "../page-props";

const kindLabels: Record<string, Key> = {
  checking: "checking",
  savings: "savings",
  cash: "cash",
  investment: "investment",
  wallet: "wallet",
  credit_card: "creditCard",
};

interface Row {
  account: Account;
  balance: AccountBalance | undefined;
}

export default function Accounts({ accounts, limits, balances, netWorthCents, kinds }: Page<"Accounts">) {
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const canManage = usePage<SharedProps>().props.workspace?.role !== "viewer";
  const { t, locale } = useFinanceI18n();
  const [showArchived, setShowArchived] = useState(false);
  const [editing, setEditing] = useState<Account | null>(null);
  const add = useForm({ name: "", kind: kinds[0] ?? "checking", opening: "" });
  const edit = useForm({ name: "", kind: "checking", opening: "" });
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const kindName = (kind: string): string => {
    const key = kindLabels[kind];
    return key ? t(key) : kind;
  };

  const rows: Row[] = accounts
    .map((account) => ({ account, balance: balances.find((b) => b.accountId === account.id) }))
    .filter((row) => showArchived || !row.account.archived_at);

  function startEdit(account: Account): void {
    edit.setData({ name: account.name, kind: account.kind, opening: centsToInput(account.opening_cents) });
    edit.clearErrors();
    setEditing(account);
  }

  function submitEdit(event: FormEvent): void {
    event.preventDefault();
    if (!editing) return;
    edit.patch(`/accounts/${editing.id}`, { preserveScroll: true, onSuccess: () => setEditing(null) });
  }

  function submitAdd(event: FormEvent): void {
    event.preventDefault();
    add.post("/accounts", { onSuccess: () => add.reset("name", "opening") });
  }

  const columns: DataTableColumn<Row>[] = [
    {
      id: "name",
      header: t("name"),
      cell: ({ account }) => (
        <span className="flex flex-wrap items-center gap-2 font-medium">
          {account.name}
          {account.archived_at && <Badge variant="secondary">{t("archived")}</Badge>}
        </span>
      ),
    },
    { id: "kind", header: t("kind"), cell: ({ account }) => kindName(account.kind) },
    {
      id: "balance",
      header: t("accountBalance"),
      align: "end",
      cell: ({ balance }) => (
        <MoneyText cents={balance?.balanceCents ?? 0} currency={currency} locale={locale} labels={labels} />
      ),
    },
    {
      id: "actions",
      header: <span className="sr-only">{t("actions")}</span>,
      align: "end",
      cell: ({ account, balance }) => (
        <div className="flex justify-end gap-1">
          {canManage && (
            <>
              {account.kind === "credit_card" ? (
                <Button asChild variant="ghost" size="icon">
                  <Link href="/cards" aria-label={`${t("cards")}: ${account.name}`}>
                    <CreditCard aria-hidden="true" />
                  </Link>
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`${t("edit")} ${account.name}`}
                  onClick={() => startEdit(account)}
                >
                  <Pencil aria-hidden="true" />
                </Button>
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`${t(account.archived_at ? "unarchive" : "archive")} ${account.name}`}
                onClick={() =>
                  router.post(`/accounts/${account.id}/${account.archived_at ? "unarchive" : "archive"}`, undefined, {
                    preserveScroll: true,
                  })
                }
              >
                {account.archived_at ? <ArchiveRestore aria-hidden="true" /> : <Archive aria-hidden="true" />}
              </Button>
              <ConfirmDelete
                href={`/accounts/${account.id}`}
                name={`${account.name} (${balance?.transactions ?? 0} ${t("transactionsWord")}). ${t("confirmDeleteAccount")}`}
                trigger={
                  <Button type="button" variant="ghost" size="icon" aria-label={`${t("delete")} ${account.name}`}>
                    <Trash2 aria-hidden="true" />
                  </Button>
                }
              />
            </>
          )}
        </div>
      ),
    },
  ];

  return (
    <>
      <Head title={t("accounts")} />
      <PageHeader
        title={t("accounts")}
        description={t("accountsDescription")}
        actions={
          <div className="flex flex-col items-end text-right" data-tour="accounts-networth">
            <span className="flex items-center gap-1 text-caption text-muted-foreground">
              {t("netWorth")}
              <HelpHint id="hintAcctNetWorth" />
            </span>
            <MoneyText
              cents={netWorthCents}
              currency={currency}
              locale={locale}
              labels={labels}
              className="text-2xl font-semibold"
            />
            <span className="text-caption text-muted-foreground">{t("netWorthNote")}</span>
          </div>
        }
      />
      {limits.accounts >= 0 && (
        <p className="text-body text-muted-foreground">
          {t("accountsPlanLimit").replace("{plan}", limits.plan).replace("{count}", String(limits.accounts))}{" "}
          <Link href="/billing" className="text-brand-text underline underline-offset-4">
            {t("accountsUpgrade")}
          </Link>
        </p>
      )}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <CardTitle>{t("accounts")}</CardTitle>
          <label className="flex items-center gap-2 text-body pointer-coarse:min-h-11" data-tour="accounts-archived">
            <input
              type="checkbox"
              checked={showArchived}
              onChange={(event) => setShowArchived(event.target.checked)}
              className="size-4"
            />
            {t("showArchived")}
            <HelpHint id="hintAcctArchive" />
          </label>
        </CardHeader>
        <CardContent data-tour="accounts-table">
          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(row) => row.account.id}
            caption={t("accountsTable")}
            empty={
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Wallet />
                  </EmptyMedia>
                  <EmptyTitle>{t("emptyAccounts")}</EmptyTitle>
                  <EmptyDescription>{t("emptyAccountsDescription")}</EmptyDescription>
                </EmptyHeader>
              </Empty>
            }
          />
          <p className="mt-3 text-sm text-muted-foreground">{t("archiveHint")}</p>
        </CardContent>
      </Card>
      {canManage && (
        <Card>
          <CardHeader>
            <CardTitle>{t("newAccount")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={submitAdd} className="grid gap-3 sm:grid-cols-3" noValidate data-tour="accounts-new">
              <Field data-invalid={!!add.errors.name}>
                <FieldLabel htmlFor="account-name">{t("name")}</FieldLabel>
                <Input
                  id="account-name"
                  required
                  value={add.data.name}
                  aria-invalid={!!add.errors.name}
                  onChange={(event) => add.setData("name", event.target.value)}
                />
                <FieldError>{add.errors.name}</FieldError>
              </Field>
              <Field data-invalid={!!add.errors.kind}>
                <FieldLabel htmlFor="account-kind">{t("kind")}</FieldLabel>
                <NativeSelect
                  id="account-kind"
                  value={add.data.kind}
                  onChange={(event) => add.setData("kind", event.target.value)}
                >
                  {kinds.map((kind) => (
                    <NativeSelectOption key={kind} value={kind}>
                      {kindName(kind)}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <FieldError>{add.errors.kind}</FieldError>
              </Field>
              <Field data-invalid={!!add.errors.opening}>
                <span className="flex items-center gap-1">
                  <FieldLabel htmlFor="account-opening">{t("opening")}</FieldLabel>
                  <HelpHint id="hintAcctOpening" />
                </span>
                <AmountInput
                  id="account-opening"
                  placeholder="0.00"
                  value={add.data.opening}
                  aria-invalid={!!add.errors.opening}
                  onChange={(event) => add.setData("opening", event.target.value)}
                />
                <FieldError>{add.errors.opening}</FieldError>
              </Field>
              <div className="sm:col-span-3">
                <Button type="submit" disabled={add.processing}>
                  <Plus data-icon="inline-start" />
                  {t("addAccount")}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      <Dialog open={editing !== null} onOpenChange={(next) => !next && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("editAccount")}</DialogTitle>
            <DialogDescription>{editing?.name}</DialogDescription>
          </DialogHeader>
          <form onSubmit={submitEdit} className="flex flex-col gap-6" noValidate>
            <FieldGroup>
              <Field data-invalid={!!edit.errors.name}>
                <FieldLabel htmlFor="edit-account-name">{t("name")}</FieldLabel>
                <Input
                  id="edit-account-name"
                  required
                  autoFocus
                  value={edit.data.name}
                  aria-invalid={!!edit.errors.name}
                  onChange={(event) => edit.setData("name", event.target.value)}
                />
                <FieldError>{edit.errors.name}</FieldError>
              </Field>
              <Field data-invalid={!!edit.errors.kind}>
                <FieldLabel htmlFor="edit-account-kind">{t("kind")}</FieldLabel>
                <NativeSelect
                  id="edit-account-kind"
                  value={edit.data.kind}
                  onChange={(event) => edit.setData("kind", event.target.value)}
                >
                  {kinds.map((kind) => (
                    <NativeSelectOption key={kind} value={kind}>
                      {kindName(kind)}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <FieldError>{edit.errors.kind}</FieldError>
              </Field>
              <Field data-invalid={!!edit.errors.opening}>
                <FieldLabel htmlFor="edit-account-opening">{t("opening")}</FieldLabel>
                <AmountInput
                  id="edit-account-opening"
                  value={edit.data.opening}
                  aria-invalid={!!edit.errors.opening}
                  onChange={(event) => edit.setData("opening", event.target.value)}
                />
                <FieldDescription>{t("openingHelp")}</FieldDescription>
                <FieldError>{edit.errors.opening}</FieldError>
              </Field>
            </FieldGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={edit.processing}>
                {t(edit.processing ? "working" : "saveAccount")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}

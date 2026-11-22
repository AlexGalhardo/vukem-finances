import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { HelpHint } from "../help/HelpHint";
import { Archive, ArchiveRestore, ArrowLeft, HandCoins, Pencil, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import AmountInput from "../components/AmountInput";
import ConfirmDelete from "../components/ConfirmDelete";
import CardForm, { type CardFormData } from "../components/cards/CardForm";
import LimitBar from "../components/cards/LimitBar";
import StatusBadge from "../components/cards/StatusBadge";
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
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { useFinanceI18n } from "../lib/i18n";
import { cn } from "../lib/utils";
import type { Page, SharedProps, TxRow } from "../page-props";

export default function CardStatement({ card, statements, statement, accounts, brands, today }: Page<"CardStatement">) {
  const shared = usePage<SharedProps>().props;
  const currency = shared.workspace?.current.currency ?? "USD";
  const canManage = shared.workspace?.role !== "viewer";
  const { t, locale, date } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const [editing, setEditing] = useState(false);
  const [paying, setPaying] = useState(false);

  const defaultPayer = card.payment_account_id.Valid
    ? String(card.payment_account_id.V)
    : String(accounts[0]?.id ?? "");
  const edit = useForm<CardFormData>({
    name: card.name,
    limit: centsToInput(card.limit_cents),
    closingDay: String(card.closing_day),
    dueDay: String(card.due_day),
    paymentAccountId: card.payment_account_id.Valid ? String(card.payment_account_id.V) : "",
    brand: card.brand,
    last4: card.last4,
  });
  const pay = useForm({ fromAccountId: defaultPayer, amount: "", date: today });

  function submitEdit(event: FormEvent): void {
    event.preventDefault();
    edit.patch(`/cards/${card.id}`, { preserveScroll: true, onSuccess: () => setEditing(false) });
  }

  function submitPay(event: FormEvent): void {
    event.preventDefault();
    pay.post(`/cards/${card.id}/statements/${statement.ref}/pay`, {
      preserveScroll: true,
      onSuccess: () => {
        setPaying(false);
        pay.reset("amount");
      },
    });
  }

  const columns: DataTableColumn<TxRow>[] = [
    { id: "date", header: t("date"), cell: (row) => date(row.occurred_on) },
    {
      id: "description",
      header: t("description"),
      cell: (row) => (
        <span className="flex flex-wrap items-center gap-2">
          {row.description}
          {row.transfer_uid.Valid && <Badge variant="secondary">{t("cardPayment")}</Badge>}
        </span>
      ),
    },
    {
      id: "amount",
      header: t("amount"),
      align: "end",
      cell: (row) => (
        <MoneyText
          cents={row.kind === "expense" ? -row.amount_cents : row.amount_cents}
          currency={currency}
          locale={locale}
          labels={labels}
          sign="always"
          tone
        />
      ),
    },
  ];

  const money = (cents: number) => (
    <MoneyText cents={cents} currency={currency} locale={locale} labels={labels} className="font-medium" />
  );

  return (
    <>
      <Head title={card.name} />
      <Link
        href="/cards"
        className="inline-flex items-center gap-1 text-body text-brand-text underline underline-offset-4"
      >
        <ArrowLeft aria-hidden="true" className="size-4" />
        {t("cardAllCards")}
      </Link>
      <PageHeader
        title={card.name}
        description={`${card.last4 ? `•••• ${card.last4} · ` : ""}${t("cardClosesOn")} ${card.closing_day} · ${t("cardDueOn")} ${card.due_day}`}
        actions={
          canManage && (
            <>
              <Button type="button" variant="outline" onClick={() => setEditing(true)}>
                <Pencil data-icon="inline-start" />
                {t("cardEdit")}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() =>
                  router.post(`/cards/${card.id}/${card.archived ? "unarchive" : "archive"}`, undefined, {
                    preserveScroll: true,
                  })
                }
              >
                {card.archived ? <ArchiveRestore data-icon="inline-start" /> : <Archive data-icon="inline-start" />}
                {t(card.archived ? "unarchive" : "archive")}
              </Button>
              <ConfirmDelete
                href={`/cards/${card.id}`}
                name={`${card.name}. ${t("cardConfirmDelete")}`}
                trigger={
                  <Button type="button" variant="ghost" size="icon" aria-label={`${t("delete")} ${card.name}`}>
                    <Trash2 aria-hidden="true" />
                  </Button>
                }
              />
            </>
          )
        }
      />
      <Card>
        <CardContent>
          <LimitBar card={card} currency={currency} />
        </CardContent>
      </Card>
      <div className="grid gap-4 lg:grid-cols-[16rem_1fr]">
        <Card data-tour="statement-picker">
          <CardHeader>
            <CardTitle>{t("cardStatements")}</CardTitle>
          </CardHeader>
          <CardContent>
            <ul aria-label={t("cardStatementList")} className="flex flex-col gap-1">
              {statements.map((item) => (
                <li key={item.ref}>
                  <Link
                    href={`/cards/${card.id}?ref=${item.ref}`}
                    preserveScroll
                    aria-current={item.ref === statement.ref ? "page" : undefined}
                    className={cn(
                      "flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted",
                      item.ref === statement.ref && "bg-muted font-medium",
                    )}
                  >
                    <span className="flex flex-col">
                      <span>{date(item.closingOn)}</span>
                      <MoneyText
                        cents={item.totalCents}
                        currency={currency}
                        locale={locale}
                        labels={labels}
                        className="text-xs text-muted-foreground"
                      />
                    </span>
                    <StatusBadge status={item.status} />
                  </Link>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
            <CardTitle className="flex items-center gap-2">
              {t("cardClosesOn")} {date(statement.closingOn)} <StatusBadge status={statement.status} />
            </CardTitle>
            {canManage && accounts.length > 0 && (
              <Button type="button" variant="brand" data-tour="statement-pay" onClick={() => setPaying(true)}>
                <HandCoins data-icon="inline-start" />
                {t("cardPay")}
              </Button>
            )}
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <dl className="grid grid-cols-2 gap-3 text-body sm:grid-cols-4" data-tour="statement-totals">
              <div>
                <dt className="flex items-center gap-1 text-caption text-muted-foreground">
                  {t("cardStatementTotal")}
                  <HelpHint id="hintStmtTotal" />
                </dt>
                <dd>{money(statement.totalCents)}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">{t("cardStatementPaid")}</dt>
                <dd>{money(statement.paidCents)}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">{t("cardStatementRemaining")}</dt>
                <dd>{money(statement.remainingCents)}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">{t("cardDueOn")}</dt>
                <dd className="font-medium">{date(statement.dueOn)}</dd>
              </div>
            </dl>
            <p className="text-xs text-muted-foreground">
              {t("cardStatementPurchases")} {money(statement.purchasesCents)} · {t("cardStatementRefunds")}{" "}
              {money(statement.creditsCents)}
            </p>
            <DataTable
              columns={columns}
              rows={statement.rows}
              rowKey={(row) => row.id}
              caption={t("cardStatementEntries")}
              empty={<p className="py-6 text-center text-sm text-muted-foreground">{t("cardStatementEmpty")}</p>}
            />
          </CardContent>
        </Card>
      </div>
      <Dialog open={editing} onOpenChange={setEditing}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("cardEdit")}</DialogTitle>
            <DialogDescription>{card.name}</DialogDescription>
          </DialogHeader>
          <form onSubmit={submitEdit} className="flex flex-col gap-6" noValidate>
            <CardForm
              idPrefix="edit-card"
              data={edit.data}
              errors={edit.errors}
              setData={edit.setData}
              accounts={accounts}
              brands={brands}
            />
            <p className="text-xs text-muted-foreground">{t("cardArchiveHint")}</p>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditing(false)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={edit.processing}>
                {t(edit.processing ? "working" : "cardSave")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog open={paying} onOpenChange={setPaying}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("cardPayTitle")}</DialogTitle>
            <DialogDescription>{t("cardPayDescription")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={submitPay} className="flex flex-col gap-6" noValidate>
            <FieldGroup>
              <Field data-invalid={!!pay.errors.fromAccountId}>
                <span className="flex items-center gap-1">
                  <FieldLabel htmlFor="pay-from">{t("cardPayFrom")}</FieldLabel>
                  <HelpHint id="hintStmtPayFrom" />
                </span>
                <NativeSelect
                  id="pay-from"
                  value={pay.data.fromAccountId}
                  onChange={(event) => pay.setData("fromAccountId", event.target.value)}
                >
                  {accounts.map((account) => (
                    <NativeSelectOption key={account.id} value={String(account.id)}>
                      {account.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <FieldError>{pay.errors.fromAccountId}</FieldError>
              </Field>
              <Field data-invalid={!!pay.errors.amount}>
                <FieldLabel htmlFor="pay-amount">{t("cardPayAmount")}</FieldLabel>
                <AmountInput
                  id="pay-amount"
                  placeholder={centsToInput(statement.remainingCents)}
                  value={pay.data.amount}
                  aria-invalid={!!pay.errors.amount}
                  onChange={(event) => pay.setData("amount", event.target.value)}
                />
                <FieldDescription>{t("cardPayAmountHelp")}</FieldDescription>
                <FieldError>{pay.errors.amount}</FieldError>
              </Field>
              <Field data-invalid={!!pay.errors.date}>
                <FieldLabel htmlFor="pay-date">{t("cardPayDate")}</FieldLabel>
                <Input
                  id="pay-date"
                  type="date"
                  value={pay.data.date}
                  onChange={(event) => pay.setData("date", event.target.value)}
                />
                <FieldError>{pay.errors.date}</FieldError>
              </Field>
            </FieldGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setPaying(false)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={pay.processing}>
                {t(pay.processing ? "working" : "cardPaySubmit")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}

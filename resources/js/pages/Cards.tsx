import { Head, Link, useForm, usePage } from "@inertiajs/react";
import { CreditCard, Plus } from "lucide-react";
import type { FormEvent } from "react";
import CardForm, { type CardFormData, emptyCardForm } from "../components/cards/CardForm";
import LimitBar from "../components/cards/LimitBar";
import StatusBadge from "../components/cards/StatusBadge";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "../components/ui/empty";
import { useFinanceI18n } from "../lib/i18n";
import type { Page, SharedProps } from "../page-props";

export default function Cards({ cards, accounts, brands }: Page<"Cards">) {
  const shared = usePage<SharedProps>().props;
  const currency = shared.workspace?.current.currency ?? "USD";
  const canManage = shared.workspace?.role !== "viewer";
  const { t, locale, date } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const add = useForm<CardFormData>(emptyCardForm);

  function submit(event: FormEvent): void {
    event.preventDefault();
    add.post("/cards", { onSuccess: () => add.reset() });
  }

  return (
    <>
      <Head title={t("cards")} />
      <PageHeader title={t("cards")} description={t("cardsDescription")} />
      <p className="text-sm text-muted-foreground">
        <strong>{t("cardsHelpTitle")}.</strong> {t("cardsHelp")}
      </p>
      {cards.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <CreditCard />
            </EmptyMedia>
            <EmptyTitle>{t("cardEmpty")}</EmptyTitle>
            <EmptyDescription>{t("cardEmptyDescription")}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ul aria-label={t("cardsTable")} className="grid gap-4 md:grid-cols-2 xl:grid-cols-3" data-tour="cards-list">
          {cards.map((card) => (
            <li key={card.id}>
              <Card>
                <CardHeader className="flex flex-row items-start justify-between gap-2">
                  <CardTitle className="flex flex-wrap items-center gap-2">
                    <Link href={`/cards/${card.id}`} className="hover:underline">
                      {card.name}
                    </Link>
                    {card.last4 && (
                      <span className="text-caption font-normal text-muted-foreground">•••• {card.last4}</span>
                    )}
                    {card.archived && <Badge variant="secondary">{t("archived")}</Badge>}
                  </CardTitle>
                  <StatusBadge status={card.current.status} />
                </CardHeader>
                <CardContent className="flex flex-col gap-3">
                  <div data-tour={card.id === cards[0]?.id ? "cards-limit" : undefined}>
                    <LimitBar card={card} currency={currency} help={card.id === cards[0]?.id} />
                  </div>
                  <dl className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <dt className="text-xs text-muted-foreground">{t("cardCurrentStatement")}</dt>
                      <dd>
                        <MoneyText
                          cents={card.current.totalCents}
                          currency={currency}
                          locale={locale}
                          labels={labels}
                          className="font-medium"
                        />
                        <span className="block text-xs text-muted-foreground">
                          {t("cardClosesOn")} {date(card.current.closingOn)}
                        </span>
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs text-muted-foreground">{t("cardNextPayment")}</dt>
                      <dd>
                        {card.due ? (
                          <>
                            <MoneyText
                              cents={card.due.remainingCents}
                              currency={currency}
                              locale={locale}
                              labels={labels}
                              className="font-medium"
                            />
                            <span className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
                              {t("cardDueOn")} {date(card.due.dueOn)} <StatusBadge status={card.due.status} />
                            </span>
                          </>
                        ) : (
                          <span className="text-muted-foreground">{t("cardNothingToPay")}</span>
                        )}
                      </dd>
                    </div>
                  </dl>
                  <Link href={`/cards/${card.id}`} className="text-sm text-primary hover:underline">
                    {t("cardStatements")}
                  </Link>
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}
      {canManage && (
        <Card data-tour="cards-new">
          <CardHeader>
            <CardTitle>{t("cardNew")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="flex flex-col gap-4" noValidate>
              <CardForm
                idPrefix="card"
                data={add.data}
                errors={add.errors}
                setData={add.setData}
                accounts={accounts}
                brands={brands}
              />
              <div>
                <Button type="submit" disabled={add.processing}>
                  <Plus data-icon="inline-start" />
                  {t("cardAdd")}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
    </>
  );
}

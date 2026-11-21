import { useFinanceI18n } from "../../lib/i18n";
import type { UpcomingItem } from "../../page-props";
import DataTable, { type DataTableColumn } from "../DataTable";
import { MoneyText } from "../MoneyText";
import { Badge } from "../ui/badge";

export interface UpcomingListProps {
  items: readonly UpcomingItem[];
  currency: string;
}

/** What is due soon: stored future-dated transactions and the occurrences of rules that are not posted yet. */
export default function UpcomingList({ items, currency }: UpcomingListProps) {
  const { t, locale, date } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };

  const sourceLabel = (item: UpcomingItem): string =>
    item.source === "installment"
      ? t("upcomingInstallment")
      : item.source === "recurrence"
        ? t("upcomingRecurring")
        : t("upcomingTransaction");

  const labelOf = (item: UpcomingItem): string =>
    item.source === "installment" && item.installmentTotal > 0
      ? `${item.description} (${item.installmentNo}/${item.installmentTotal})`
      : item.description;

  const columns: DataTableColumn<UpcomingItem>[] = [
    { id: "date", header: t("date"), cell: (item) => date(item.date) },
    { id: "description", header: t("description"), cell: (item) => labelOf(item) },
    {
      id: "source",
      header: t("upcomingSource"),
      cell: (item) => <Badge variant="secondary">{sourceLabel(item)}</Badge>,
    },
    {
      id: "amount",
      header: t("amount"),
      align: "end",
      cell: (item) => (
        <MoneyText
          cents={item.kind === "expense" ? -item.amountCents : item.amountCents}
          currency={currency}
          locale={locale}
          labels={labels}
          tone
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={items}
      rowKey={(item) => `${item.date}-${item.transactionId}-${item.recurrenceId}`}
      caption={t("upcomingTable")}
      empty={<p className="py-4 text-sm text-muted-foreground">{t("upcomingEmpty")}</p>}
    />
  );
}

import { type ReactNode, useId, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import { Button } from "../ui/button";

/**
 * A chart and the table with the same numbers. The table is always in the page: visually hidden while the chart
 * shows, so a screen reader reads it (the chart itself is `aria-hidden`, and `summary` says in one sentence what it
 * draws). The "Show as table" button swaps them for everyone. Printing shows the table (`print.css`).
 */
export default function ChartWithTable({
  chart,
  table,
  summary,
  title,
}: {
  chart: ReactNode;
  table: ReactNode;
  summary: string;
  title: string;
}) {
  const { t } = useFinanceI18n();
  const [asTable, setAsTable] = useState(false);
  const tableId = useId();

  return (
    <figure className="flex flex-col gap-3" aria-label={title}>
      <figcaption className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-sm text-muted-foreground">{summary}</span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-pressed={asTable}
          aria-controls={tableId}
          data-print="hide"
          onClick={() => setAsTable((on) => !on)}
        >
          {asTable ? t("rptShowChart") : t("rptShowTable")}
        </Button>
      </figcaption>
      <div aria-hidden="true" hidden={asTable} data-report-chart>
        {chart}
      </div>
      <div id={tableId} className={asTable ? "overflow-x-auto" : "sr-only"} data-report-table>
        {table}
      </div>
    </figure>
  );
}

import { type Key, useFinanceI18n } from "../../lib/i18n";
import { Badge } from "../ui/badge";

const statusKeys: Record<string, Key> = {
  open: "cardStatusOpen",
  closed: "cardStatusClosed",
  paid: "cardStatusPaid",
  overdue: "cardStatusOverdue",
};

/** The status is always spelled out; the badge style only reinforces it. */
export default function StatusBadge({ status }: { status: string }) {
  const { t } = useFinanceI18n();
  const key = statusKeys[status];
  return (
    <Badge variant={status === "overdue" ? "destructive" : status === "paid" ? "default" : "secondary"}>
      {key ? t(key) : status}
    </Badge>
  );
}

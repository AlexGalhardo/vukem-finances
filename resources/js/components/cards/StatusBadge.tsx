import { CircleAlert, CircleCheck, Clock, Info } from "lucide-react";
import { type Key, useFinanceI18n } from "../../lib/i18n";
import { Badge } from "../ui/badge";

const statusKeys: Record<string, Key> = {
  open: "cardStatusOpen",
  closed: "cardStatusClosed",
  paid: "cardStatusPaid",
  overdue: "cardStatusOverdue",
};

const looks = {
  open: { variant: "neutral", Icon: Clock },
  closed: { variant: "info", Icon: Info },
  paid: { variant: "success", Icon: CircleCheck },
  overdue: { variant: "danger", Icon: CircleAlert },
} as const;

/** The status is always spelled out and carries its own icon: the badge color only reinforces it. */
export default function StatusBadge({ status }: { status: string }) {
  const { t } = useFinanceI18n();
  const key = statusKeys[status];
  const look = looks[status as keyof typeof looks] ?? looks.open;
  return (
    <Badge variant={look.variant}>
      <look.Icon aria-hidden="true" />
      {key ? t(key) : status}
    </Badge>
  );
}

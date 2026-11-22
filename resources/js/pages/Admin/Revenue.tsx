import { router } from "@inertiajs/react";
import { useState } from "react";
import DataTable from "../../components/DataTable";
import Stat from "../../components/Stat";
import { Button } from "../../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import AdminLayout from "../../layouts/AdminLayout";
import { useFinanceI18n } from "../../lib/i18n";
import type { Page } from "../../page-props";

export default function Revenue({
  mrrCents,
  pixMrrCents,
  pixActive,
  active,
  trialing,
  pastDue,
  workspaces,
  users,
  recentPix,
}: Page<"Admin/Revenue">) {
  const { t, money } = useFinanceI18n();
  const [refunding, setRefunding] = useState<string | null>(null);
  const stats: [string, string][] = [
    [t("revenueMrr"), money(mrrCents, "USD")],
    [t("revenuePixMrr"), money(pixMrrCents, "BRL")],
    [t("revenuePixActive"), String(pixActive)],
    [t("revenueActive"), String(active)],
    [t("revenueTrialing"), String(trialing)],
    [t("revenuePastDue"), String(pastDue)],
    [t("revenueWorkspaces"), String(workspaces)],
    [t("revenueUsers"), String(users)],
  ];

  const markRefunded = () => {
    if (refunding) router.post(`/admin/pix-payments/${encodeURIComponent(refunding)}/refund`);
    setRefunding(null);
  };

  return (
    <AdminLayout title={t("revenueTitle")}>
      <div data-tour="admin-revenue-summary" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(([label, value]) => (
          <div key={label} className="rounded-lg border border-border p-4">
            <Stat label={label} value={value} />
          </div>
        ))}
      </div>
      <p className="max-w-[65ch] text-body text-pretty text-muted-foreground">{t("revenueNote")}</p>

      <section className="flex flex-col gap-3">
        <h2 className="text-h3">{t("revenuePixTitle")}</h2>
        <DataTable
          caption={t("revenuePixTitle")}
          rows={recentPix}
          rowKey={(p) => p.ref}
          empty={t("revenuePixNone")}
          columns={[
            {
              id: "workspace",
              header: t("revenuePixWorkspace"),
              cell: (p) => <span className="tabular-nums">#{p.workspace}</span>,
            },
            { id: "status", header: t("revenuePixStatus"), cell: (p) => p.status },
            { id: "amount", header: t("revenuePixAmount"), align: "end", cell: (p) => money(p.amountCents, "BRL") },
            { id: "months", header: t("revenuePixMonths"), align: "end", cell: (p) => p.months },
            {
              id: "actions",
              header: <span className="sr-only">{t("revenuePixMarkRefunded")}</span>,
              align: "end",
              cell: (p) =>
                p.canRefund && (
                  <Button type="button" size="sm" variant="outline" onClick={() => setRefunding(p.ref)}>
                    {t("revenuePixMarkRefunded")}
                  </Button>
                ),
            },
          ]}
        />
      </section>

      <Dialog open={refunding !== null} onOpenChange={(open) => !open && setRefunding(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admRefundTitle")}</DialogTitle>
            <DialogDescription>{t("revenuePixConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setRefunding(null)}>
              {t("admCancel")}
            </Button>
            <Button type="button" variant="destructive" onClick={markRefunded}>
              {t("revenuePixMarkRefunded")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </AdminLayout>
  );
}

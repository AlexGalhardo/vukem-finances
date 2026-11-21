import { Head, Link, router } from "@inertiajs/react";

import { Card } from "../../components/ui";
import { Button } from "../../components/ui/button";
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

  const markRefunded = (ref: string) => {
    if (window.confirm(t("revenuePixConfirm"))) router.post(`/admin/pix-payments/${encodeURIComponent(ref)}/refund`);
  };

  return (
    <>
      <Head title={t("revenueTitle")} />
      <div className="flex items-baseline justify-between">
        <h1 className="text-2xl font-semibold">{t("revenueTitle")}</h1>
        <Link href="/admin" className="text-indigo-600 hover:underline dark:text-indigo-400">
          {t("revenueAdminPanel")}
        </Link>
      </div>
      <div className="grid gap-4 sm:grid-cols-4">
        {stats.map(([label, value]) => (
          <Card key={label} title={label}>
            <p className="text-2xl font-semibold tabular-nums">{value}</p>
          </Card>
        ))}
      </div>
      <p className="text-sm text-muted-foreground">{t("revenueNote")}</p>

      <Card title={t("revenuePixTitle")}>
        {recentPix.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("revenuePixNone")}</p>
        ) : (
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wide text-muted-foreground">
                <th className="py-1 pr-3 font-medium">{t("revenuePixWorkspace")}</th>
                <th className="py-1 pr-3 font-medium">{t("revenuePixStatus")}</th>
                <th className="py-1 pr-3 font-medium">{t("revenuePixAmount")}</th>
                <th className="py-1 pr-3 font-medium">{t("revenuePixMonths")}</th>
                <th className="py-1" />
              </tr>
            </thead>
            <tbody>
              {recentPix.map((p) => (
                <tr key={p.ref} className="border-t">
                  <td className="py-1.5 pr-3 tabular-nums">#{p.workspace}</td>
                  <td className="py-1.5 pr-3">{p.status}</td>
                  <td className="py-1.5 pr-3 tabular-nums">{money(p.amountCents, "BRL")}</td>
                  <td className="py-1.5 pr-3 tabular-nums">{p.months}</td>
                  <td className="py-1.5 text-right">
                    {p.canRefund && (
                      <Button type="button" size="sm" variant="outline" onClick={() => markRefunded(p.ref)}>
                        {t("revenuePixMarkRefunded")}
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </>
  );
}

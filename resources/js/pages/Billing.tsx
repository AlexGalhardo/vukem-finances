import { Head, Link, router, useForm } from "@inertiajs/react";
import { useState } from "react";
import PageHeader from "../components/PageHeader";
import PixDialog from "../components/PixDialog";
import { Card, Select, secondary } from "../components/ui";
import { Button } from "../components/ui/button";
import { type Key, useFinanceI18n } from "../lib/i18n";
import { fill, type PixCharge } from "../lib/pix";
import type { Page } from "../page-props";

const revokedKeys: Record<string, Key> = {
  refund: "billingRevokedRefund",
  dispute: "billingRevokedDispute",
  dispute_lost: "billingRevokedDisputeLost",
  admin: "billingRevokedAdmin",
};

const statusKeys: Record<string, Key> = {
  trialing: "billingStatusTrialing",
  active: "billingStatusActive",
  past_due: "billingStatusPastDue",
  unpaid: "billingStatusUnpaid",
  incomplete: "billingStatusIncomplete",
  canceled: "billingStatusCanceled",
};

export default function Billing({
  limits,
  subscription,
  trialDays,
  canManage,
  configured,
  checkout,
  pix,
  pixAccess,
}: Page<"Billing">) {
  const { t, money, date } = useFinanceI18n();
  const form = useForm({});
  const pro = limits.plan === "pro";
  const [months, setMonths] = useState(pix?.options[0]?.months ?? 1);
  const [starting, setStarting] = useState(false);
  const [startFailed, setStartFailed] = useState(false);
  const [charge, setCharge] = useState<PixCharge | null>(null);

  const total = pix?.options.find((o) => o.months === months)?.totalCents ?? 0;

  const startPix = async () => {
    setStarting(true);
    setStartFailed(false);
    try {
      const res = await fetch("/billing/pix", {
        method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ months: String(months) }),
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(String(res.status));
      setCharge((await res.json()) as PixCharge);
    } catch {
      setStartFailed(true);
    } finally {
      setStarting(false);
    }
  };

  // One sentence says where the plan stands, whichever way it was paid.
  const revoked = subscription?.revokedReason || pixAccess?.revokedReason || "";
  let state = t("billingStateFree");
  if (pro && subscription?.valid) {
    const end = subscription.currentPeriodEnd ? date(subscription.currentPeriodEnd) : "";
    if (subscription.status === "trialing") state = fill(t("billingStateTrialing"), { date: end });
    else if (subscription.cancelAtPeriodEnd) state = fill(t("billingStateEnds"), { date: end });
    else state = end ? fill(t("billingStateActive"), { date: end }) : t("billingStateActiveNoDate");
  } else if (pro && pixAccess?.active) {
    state = fill(t("billingStatePix"), { date: date(pixAccess.validUntil) });
  } else if (revoked) {
    state = fill(t("billingStateRevoked"), { reason: t(revokedKeys[revoked] ?? "billingRevokedAdmin") });
  }

  return (
    <>
      <Head title={t("billing")} />
      <PageHeader title={t("billing")} />
      {checkout && (
        <p
          role="status"
          className="rounded-md bg-indigo-50 px-4 py-3 text-indigo-900 dark:bg-indigo-950 dark:text-indigo-100"
        >
          {t("billingThanks")}
        </p>
      )}
      {!configured && !pix && <p className="text-sm text-amber-700">{t("billingStripeMissing")}</p>}

      <section
        aria-labelledby="plan-state"
        className={`flex flex-col gap-1 rounded-xl border-l-4 px-4 py-3 ${
          pro
            ? "border-emerald-500 bg-emerald-50/60 dark:bg-emerald-950/40"
            : revoked
              ? "border-amber-500 bg-amber-50/60 dark:bg-amber-950/40"
              : "border-border bg-muted/40"
        }`}
      >
        <h2 id="plan-state" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t("billingStateTitle")}: {pro ? t("billingPro") : t("billingFree")}
        </h2>
        <p data-testid="plan-state">{state}</p>
      </section>

      <div className="grid gap-4 sm:grid-cols-2">
        <Card title={t("billingFree")}>
          <ul className="list-disc space-y-1 pl-5 text-sm">
            <li>{t("billingFreeAccounts")}</li>
            <li>{t("billingFreeTransactions")}</li>
            <li>{t("billingFreeMembers")}</li>
          </ul>
        </Card>
        <Card title={t("billingPro")}>
          <ul className="list-disc space-y-1 pl-5 text-sm">
            <li>{t("billingProUnlimited")}</li>
            <li>{t("billingProImport")}</li>
            <li>{t("billingProBudgets")}</li>
          </ul>
          {pro && (
            <p className="mt-3 text-sm font-medium text-emerald-700 dark:text-emerald-400">{t("billingOnPro")}</p>
          )}
          {!canManage && !pro && <p className="mt-3 text-sm text-muted-foreground">{t("billingOwnerOnly")}</p>}
        </Card>
      </div>

      {canManage && !subscription?.valid && (
        <div className="grid gap-4 sm:grid-cols-2">
          <Card title={t("billingCardTitle")}>
            <p className="mb-3 text-sm text-muted-foreground">{t("billingCardNote")}</p>
            <Button
              type="button"
              disabled={form.processing || !configured}
              onClick={() => form.post("/billing/checkout/pro")}
            >
              {subscription
                ? t("billingResubscribe")
                : trialDays > 0
                  ? fill(t("billingStartTrial"), { days: trialDays })
                  : t("billingSubscribe")}
            </Button>
          </Card>
          {pix && (
            <Card title={t("billingPixTitle")}>
              <p className="mb-3 text-sm text-muted-foreground">{t("billingPixNote")}</p>
              <div className="flex flex-wrap items-end gap-3">
                <Select
                  label={t("billingPixPeriod")}
                  value={months}
                  onChange={(e) => setMonths(Number(e.target.value))}
                >
                  {pix.options.map((o) => (
                    <option key={o.months} value={o.months}>
                      {fill(t("billingPixOption"), { months: o.months })}
                    </option>
                  ))}
                </Select>
                <p className="pb-2 text-sm font-medium tabular-nums" aria-live="polite">
                  {fill(t("billingPixTotal"), { total: money(total, pix.currency) })}
                </p>
              </div>
              <Button type="button" className="mt-3" disabled={starting || pix.options.length === 0} onClick={startPix}>
                {starting ? t("billingPixStarting") : t("billingPixPay")}
              </Button>
              {startFailed && (
                <p role="alert" className="mt-2 text-sm text-destructive">
                  {t("billingPixError")}
                </p>
              )}
            </Card>
          )}
        </div>
      )}

      {subscription && (
        <Card title={t("billingStatusTitle")}>
          <p>
            {t("billingStatusLabel")}: <strong>{t(statusKeys[subscription.status] ?? "billingStatusCanceled")}</strong>
            {subscription.currentPeriodEnd &&
              `, ${fill(t("billingPeriodEnds"), { date: date(subscription.currentPeriodEnd) })}`}
            {subscription.cancelAtPeriodEnd && ` (${t("billingCancelsAtEnd")})`}
          </p>
          {!subscription.valid && !subscription.revokedReason && (
            <p className="mt-2 text-sm text-amber-700">{t("billingNotActive")}</p>
          )}
          {canManage && (
            <button
              type="button"
              className={`${secondary} mt-3`}
              disabled={form.processing}
              onClick={() => form.post("/billing/portal")}
            >
              {t("billingOpenPortal")}
            </button>
          )}
        </Card>
      )}

      <p className="text-sm text-muted-foreground">
        {t("billingLimitsReached")}{" "}
        <Link href="/workspace" className="text-indigo-600 hover:underline dark:text-indigo-400">
          {t("billingManageMembers")}
        </Link>
        .
      </p>

      {charge && <PixDialog charge={charge} onClose={() => setCharge(null)} onPaid={() => router.reload()} />}
    </>
  );
}

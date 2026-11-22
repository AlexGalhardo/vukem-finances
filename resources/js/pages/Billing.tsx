import { Head, Link, router, useForm } from "@inertiajs/react";
import { CircleCheck, Info, TriangleAlert } from "lucide-react";
import { useState } from "react";
import PageHeader from "../components/PageHeader";
import PixDialog from "../components/PixDialog";
import SettingsSection from "../components/settings/SettingsSection";
import { Select } from "../components/ui";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { HelpHint } from "../help/HelpHint";
import { type Key, useFinanceI18n } from "../lib/i18n";
import { fill, type PixCharge } from "../lib/pix";
import { cn } from "../lib/utils";
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
  // The plan banner says its state in words and in an icon, never in its color alone.
  const StateIcon = pro ? CircleCheck : revoked ? TriangleAlert : Info;

  return (
    <>
      <Head title={t("billing")} />
      <PageHeader title={t("billing")} />
      {checkout && (
        <Alert variant="success">
          <AlertDescription>{t("billingThanks")}</AlertDescription>
        </Alert>
      )}
      {!configured && !pix && (
        <Alert variant="warning">
          <AlertDescription>{t("billingStripeMissing")}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col">
        <SettingsSection title={t("billingStateTitle")} description={t("billingPlanDescription")} tour="billing-plan">
          <section
            aria-labelledby="plan-state"
            className={cn(
              "flex flex-col gap-1 rounded-lg px-4 py-3",
              pro ? "bg-brand-subtle" : revoked ? "bg-warning-subtle" : "bg-muted",
            )}
          >
            <h3 id="plan-state" className="flex items-center gap-2 text-label">
              <StateIcon aria-hidden="true" className="size-4 shrink-0" />
              {t("billingStateTitle")}: {pro ? t("billingPro") : t("billingFree")}
            </h3>
            <p data-testid="plan-state">{state}</p>
          </section>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-2 rounded-lg border border-border p-4">
              <h3 className="text-h3">{t("billingFree")}</h3>
              <ul className="flex list-disc flex-col gap-1 ps-5 text-body">
                <li>{t("billingFreeAccounts")}</li>
                <li>{t("billingFreeTransactions")}</li>
                <li>{t("billingFreeMembers")}</li>
              </ul>
            </div>
            <div className="flex flex-col gap-2 rounded-lg border border-border p-4">
              <h3 className="flex items-center gap-2 text-h3">
                {t("billingPro")}
                {pro && <Badge variant="success">{t("billingOnPro")}</Badge>}
              </h3>
              <ul className="flex list-disc flex-col gap-1 ps-5 text-body">
                <li>{t("billingProUnlimited")}</li>
                <li>{t("billingProImport")}</li>
                <li>{t("billingProBudgets")}</li>
              </ul>
            </div>
          </div>
          {!canManage && <p className="text-caption text-muted-foreground">{t("billingOwnerOnlyCard")}</p>}
        </SettingsSection>

        {canManage && !subscription?.valid && (
          <SettingsSection
            title={t("billingCardTitle")}
            description={t("billingCardNote")}
            tour="billing-card"
            help={
              trialDays > 0 ? (
                <HelpHint id="hintBillTrial" labelKey="billingHelpTrial" tutorialStep="Billing:1" />
              ) : undefined
            }
          >
            <div>
              <Button
                type="button"
                variant="brand"
                loading={form.processing}
                disabled={!configured}
                onClick={() => form.post("/billing/checkout/pro")}
              >
                {subscription
                  ? t("billingResubscribe")
                  : trialDays > 0
                    ? fill(t("billingStartTrial"), { days: trialDays })
                    : t("billingSubscribe")}
              </Button>
            </div>
            {!configured && pix && <p className="text-caption text-muted-foreground">{t("billingStripeMissing")}</p>}
          </SettingsSection>
        )}

        {canManage && !subscription?.valid && pix && (
          <SettingsSection
            title={t("billingPixTitle")}
            description={t("billingPixNote")}
            tour="billing-pix"
            help={<HelpHint id="hintBillPixPeriod" labelKey="billingHelpPixPeriod" tutorialStep="Billing:2" />}
          >
            <div className="flex flex-wrap items-end gap-x-4 gap-y-2">
              <Select label={t("billingPixPeriod")} value={months} onChange={(e) => setMonths(Number(e.target.value))}>
                {pix.options.map((o) => (
                  <option key={o.months} value={o.months}>
                    {fill(t("billingPixOption"), { months: o.months })}
                  </option>
                ))}
              </Select>
              <p className="pb-2 text-label tabular-nums" aria-live="polite">
                {fill(t("billingPixTotal"), { total: money(total, pix.currency) })}
              </p>
            </div>
            <div>
              <Button type="button" loading={starting} disabled={pix.options.length === 0} onClick={startPix}>
                {starting ? t("billingPixStarting") : t("billingPixPay")}
              </Button>
            </div>
            {startFailed && (
              <p role="alert" className="text-body text-expense">
                {t("billingPixError")}
              </p>
            )}
          </SettingsSection>
        )}

        {subscription && (
          <SettingsSection title={t("billingStatusTitle")}>
            <p>
              {t("billingStatusLabel")}:{" "}
              <strong>{t(statusKeys[subscription.status] ?? "billingStatusCanceled")}</strong>
              {subscription.currentPeriodEnd &&
                `, ${fill(t("billingPeriodEnds"), { date: date(subscription.currentPeriodEnd) })}`}
              {subscription.cancelAtPeriodEnd && ` (${t("billingCancelsAtEnd")})`}
            </p>
            {!subscription.valid && !subscription.revokedReason && (
              <p className="flex items-start gap-2 text-body text-warning">
                <TriangleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
                {t("billingNotActive")}
              </p>
            )}
            {canManage && (
              <div>
                <Button
                  type="button"
                  variant="outline"
                  loading={form.processing}
                  onClick={() => form.post("/billing/portal")}
                >
                  {t("billingOpenPortal")}
                </Button>
              </div>
            )}
          </SettingsSection>
        )}
      </div>

      <p className="text-body text-muted-foreground">
        {t("billingLimitsReached")}{" "}
        <Link href="/workspace" className="text-brand-text underline underline-offset-4">
          {t("billingManageMembers")}
        </Link>
        .
      </p>

      {charge && <PixDialog charge={charge} onClose={() => setCharge(null)} onPaid={() => router.reload()} />}
    </>
  );
}

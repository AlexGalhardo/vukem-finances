import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import type { FormEvent } from "react";

import PageHeader from "../components/PageHeader";
import SettingsSection from "../components/settings/SettingsSection";
import { Button } from "../components/ui/button";
import { Checkbox } from "../components/ui/checkbox";
import { HelpHint } from "../help/HelpHint";
import type { Key } from "../i18n/keys";
import { useFinanceI18n } from "../lib/i18n";
import { type NotificationParams, notificationTitle } from "../lib/notifications";
import type { Page, SharedProps } from "../page-props";

const kindLabel: Record<string, Key> = {
  budget_80: "alertKindBudget80",
  budget_100: "alertKindBudget100",
  overall_budget_100: "alertKindOverallBudget100",
  card_limit_80: "alertKindCardLimit80",
  statement_due: "alertKindStatementDue",
  goal_reached: "alertKindGoalReached",
  recurrence_failed: "alertKindRecurrenceFailed",
};

export default function Notifications({ items, preferences, telegramEnabled }: Page<"Notifications">) {
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const { t, money, date } = useFinanceI18n();
  const formats = { money, date, currency };
  const form = useForm({
    inApp: preferences.filter((p) => p.inApp).map((p) => p.kind),
    email: preferences.filter((p) => p.email).map((p) => p.kind),
  });
  const emailAllowed = preferences.every((p) => p.emailAllowed);
  const unread = items.some((n) => !n.read);

  const toggle = (field: "inApp" | "email", kind: string, on: boolean) =>
    form.setData(field, on ? [...form.data[field], kind] : form.data[field].filter((k) => k !== kind));

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.put("/notifications/preferences", { preserveScroll: true });
  };

  return (
    <>
      <Head title={t("notifications")} />
      <PageHeader
        title={t("notifications")}
        description={t("notificationsDescription")}
        actions={
          unread && (
            <Button type="button" variant="outline" onClick={() => router.post("/notifications/read", {})}>
              {t("notificationsMarkAllRead")}
            </Button>
          )
        }
      />
      <section aria-label={t("notificationsListTitle")} data-tour="notifications-list">
        {items.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-muted-foreground">
            {t("notificationsEmpty")}
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-border rounded-lg border border-border">
            {items.map((n) => (
              <li key={n.id} className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2 px-4 py-3">
                <div className="flex min-w-0 flex-col gap-0.5">
                  <span className={n.read ? "text-muted-foreground" : "text-label"}>
                    {notificationTitle(n.titleKey, n.params as NotificationParams, t, formats)}
                  </span>
                  <span className="text-caption text-muted-foreground">
                    {n.read ? t("notificationsReadState") : t("notificationsUnread")} · {date(n.createdAt)}
                  </span>
                </div>
                {!n.read && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => router.post(`/notifications/${n.id}/read`, {}, { preserveScroll: true })}
                  >
                    {t("notificationsMarkRead")}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
      <SettingsSection
        title={t("notificationsPreferences")}
        description={t("notificationsPreferencesHelp")}
        tour="notifications-preferences"
        help={<HelpHint id="hintNotifChannels" labelKey="notificationsHelpChannels" tutorialStep="Notifications:1" />}
      >
        <form onSubmit={submit} className="flex flex-col gap-4">
          <div className="overflow-x-auto">
            <table className="w-full text-body">
              <thead>
                <tr className="text-start text-caption text-muted-foreground">
                  <th scope="col" className="py-2 text-start font-medium">
                    {t("notificationsKind")}
                  </th>
                  <th scope="col" className="px-3 py-2 text-start font-medium">
                    {t("notificationsInApp")}
                  </th>
                  <th scope="col" className="px-3 py-2 text-start font-medium">
                    {t("notificationsEmail")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {preferences.map((p) => (
                  <tr key={p.kind} className="border-t border-border">
                    <th scope="row" className="py-3 pe-3 text-start font-normal">
                      {t(kindLabel[p.kind] ?? "notifications")}
                    </th>
                    <td className="px-3">
                      <Checkbox
                        aria-label={`${t(kindLabel[p.kind] ?? "notifications")}: ${t("notificationsInApp")}`}
                        checked={form.data.inApp.includes(p.kind)}
                        onCheckedChange={(checked) => toggle("inApp", p.kind, checked === true)}
                      />
                    </td>
                    <td className="px-3">
                      <Checkbox
                        aria-label={`${t(kindLabel[p.kind] ?? "notifications")}: ${t("notificationsEmail")}`}
                        disabled={!p.emailAllowed}
                        checked={p.emailAllowed && form.data.email.includes(p.kind)}
                        onCheckedChange={(checked) => toggle("email", p.kind, checked === true)}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!emailAllowed && <p className="text-caption text-muted-foreground">{t("notificationsEmailViewer")}</p>}
          {telegramEnabled && (
            <p className="text-body">
              <Link href="/settings/telegram" className="text-brand-text underline underline-offset-4">
                {t("telegramSettings")}
              </Link>
            </p>
          )}
          <div>
            <Button type="submit" loading={form.processing}>
              {t("notificationsSave")}
            </Button>
          </div>
        </form>
      </SettingsSection>
    </>
  );
}

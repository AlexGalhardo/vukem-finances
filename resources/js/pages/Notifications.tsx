import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import type { FormEvent } from "react";

import PageHeader from "../components/PageHeader";
import { Card, primary, secondary } from "../components/ui";
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
            <button type="button" className={secondary} onClick={() => router.post("/notifications/read", {})}>
              {t("notificationsMarkAllRead")}
            </button>
          )
        }
      />
      <Card>
        {items.length === 0 ? (
          <p className="text-muted-foreground">{t("notificationsEmpty")}</p>
        ) : (
          <ul className="divide-y divide-border">
            {items.map((n) => (
              <li key={n.id} className="flex items-start justify-between gap-3 py-3">
                <div className="flex flex-col gap-0.5">
                  <span className={n.read ? "text-muted-foreground" : "font-medium"}>
                    {notificationTitle(n.titleKey, n.params as NotificationParams, t, formats)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {n.read ? t("notificationsReadState") : t("notificationsUnread")} · {date(n.createdAt)}
                  </span>
                </div>
                {!n.read && (
                  <button
                    type="button"
                    className={secondary}
                    onClick={() => router.post(`/notifications/${n.id}/read`, {}, { preserveScroll: true })}
                  >
                    {t("notificationsMarkRead")}
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </Card>
      <Card title={t("notificationsPreferences")}>
        <form onSubmit={submit} className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">{t("notificationsPreferencesHelp")}</p>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left">
                <th scope="col" className="py-1 font-medium">
                  {t("notificationsKind")}
                </th>
                <th scope="col" className="py-1 font-medium">
                  {t("notificationsInApp")}
                </th>
                <th scope="col" className="py-1 font-medium">
                  {t("notificationsEmail")}
                </th>
              </tr>
            </thead>
            <tbody>
              {preferences.map((p) => (
                <tr key={p.kind} className="border-t border-border">
                  <th scope="row" className="py-2 text-left font-normal">
                    {t(kindLabel[p.kind] ?? "notifications")}
                  </th>
                  <td>
                    <input
                      type="checkbox"
                      aria-label={`${t(kindLabel[p.kind] ?? "notifications")}: ${t("notificationsInApp")}`}
                      checked={form.data.inApp.includes(p.kind)}
                      onChange={(e) => toggle("inApp", p.kind, e.target.checked)}
                    />
                  </td>
                  <td>
                    <input
                      type="checkbox"
                      aria-label={`${t(kindLabel[p.kind] ?? "notifications")}: ${t("notificationsEmail")}`}
                      disabled={!p.emailAllowed}
                      checked={p.emailAllowed && form.data.email.includes(p.kind)}
                      onChange={(e) => toggle("email", p.kind, e.target.checked)}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!emailAllowed && <p className="text-xs text-muted-foreground">{t("notificationsEmailViewer")}</p>}
          {telegramEnabled && (
            <p className="text-sm">
              <Link href="/settings/telegram" className="underline">
                {t("telegramSettings")}
              </Link>
            </p>
          )}
          <div>
            <button type="submit" disabled={form.processing} className={primary}>
              {t("notificationsSave")}
            </button>
          </div>
        </form>
      </Card>
    </>
  );
}

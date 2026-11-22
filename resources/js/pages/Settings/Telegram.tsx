import { Head, router, useForm, usePage } from "@inertiajs/react";
import { type FormEvent, useEffect, useState } from "react";
import ConfirmDelete from "../../components/ConfirmDelete";
import PageHeader from "../../components/PageHeader";
import { Card, danger, primary } from "../../components/ui";
import { Button } from "../../components/ui/button";
import { useFinanceI18n } from "../../lib/i18n";
import type { Page } from "../../page-props";

/** m:ss for a countdown. */
function clock(seconds: number): string {
  const s = Math.max(0, seconds);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

export default function Telegram({ username, links, alertsOn, maxLinks, created }: Page<"Settings/Telegram">) {
  const { t, date } = useFinanceI18n();
  const { edition } = usePage().props as { edition?: string };
  const connect = useForm({});
  const [dismissed, setDismissed] = useState(false);
  const [left, setLeft] = useState(created?.seconds ?? 0);

  // The countdown reads the clock against a deadline, so a throttled tab or a jumped clock cannot show a stale time.
  useEffect(() => {
    setDismissed(false);
    setLeft(created?.seconds ?? 0);
    if (!created) return;
    const deadline = Date.now() + created.seconds * 1000;
    const timer = window.setInterval(() => setLeft(Math.max(0, Math.ceil((deadline - Date.now()) / 1000))), 1000);
    return () => window.clearInterval(timer);
  }, [created]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    connect.post("/settings/telegram/connect", { preserveScroll: true });
  };

  // Optimistic, so the box answers the click at once; the server's answer comes back as the prop.
  const [on, setOn] = useState(alertsOn);
  useEffect(() => setOn(alertsOn), [alertsOn]);

  const setAlerts = (enabled: boolean) => {
    setOn(enabled);
    router.put("/settings/telegram/alerts", { enabled: String(enabled) }, { preserveScroll: true });
  };

  const expired = created !== null && created !== undefined && left <= 0;

  return (
    <>
      <Head title={t("telegram")} />
      <PageHeader title={t("telegram")} description={t("telegramIntro")} />

      <Card>
        <p className="text-sm text-muted-foreground">{t("telegramPrivacy")}</p>
        {edition === "personal" && <p className="mt-2 text-sm text-muted-foreground">{t("telegramNeedsInternet")}</p>}
      </Card>

      {created && !dismissed && (
        <Card title={t("telegramCreatedTitle")}>
          <p className="mb-2">{t("telegramCreatedBody")}</p>
          {expired ? (
            <p role="status" data-testid="telegram-expired" className="mb-3 font-medium text-destructive">
              {t("telegramExpired")}
            </p>
          ) : (
            <>
              <p role="timer" data-testid="telegram-countdown" className="mb-3 font-medium">
                {t("telegramExpiresIn").replace("{time}", clock(left))}
              </p>
              <p className="mb-2 break-all">
                <a
                  data-testid="telegram-link"
                  href={created.link}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="underline"
                >
                  {created.link}
                </a>
              </p>
              <p className="mb-3 break-all text-sm">
                {t("telegramCodeHint")} <code data-testid="telegram-code">{`/start ${created.code}`}</code>
              </p>
              <div className="flex flex-wrap gap-2">
                <Button asChild>
                  <a href={created.link} target="_blank" rel="noreferrer noopener">
                    {t("telegramOpen")}
                  </a>
                </Button>
                <Button type="button" variant="outline" onClick={() => navigator.clipboard?.writeText(created.link)}>
                  {t("telegramCopy")}
                </Button>
              </div>
            </>
          )}
          <div className="mt-3">
            <Button type="button" variant="outline" onClick={() => setDismissed(true)}>
              {t("telegramDismiss")}
            </Button>
          </div>
        </Card>
      )}

      <Card title={t("telegramConnect")}>
        <form onSubmit={submit} className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            {t("telegramConnectHint")} (@{username})
          </p>
          <p className="text-sm text-muted-foreground">{t("telegramLimit").replace("{max}", String(maxLinks))}</p>
          <div>
            <button type="submit" disabled={connect.processing || links.length >= maxLinks} className={primary}>
              {t("telegramConnect")}
            </button>
          </div>
        </form>
      </Card>

      <Card title={t("telegramLinksTitle")}>
        {links.length === 0 ? (
          <p>{t("telegramNone")}</p>
        ) : (
          <ul className="divide-y divide-border">
            {links.map((link) => (
              <li key={link.id} className="flex items-center justify-between gap-3 py-3">
                <div>
                  <p className="font-medium">{t("telegramChat").replace("{hint}", link.chatHint)}</p>
                  <p className="text-sm text-muted-foreground">
                    {t("telegramLinkedOn")} {date(link.linkedAt)} · {t("telegramLastSeen")}:{" "}
                    {link.lastSeenAt ? date(link.lastSeenAt) : t("telegramNeverSeen")} · {link.language}
                  </p>
                </div>
                <ConfirmDelete
                  href={`/settings/telegram/links/${link.id}`}
                  name={t("telegramChat").replace("{hint}", link.chatHint)}
                  trigger={
                    <button type="button" className={danger}>
                      {t("telegramDisconnect")}
                    </button>
                  }
                />
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card title={t("telegramAlertsTitle")}>
        <label className="flex items-center gap-2">
          <input type="checkbox" className="size-4" checked={on} onChange={(e) => setAlerts(e.target.checked)} />
          {t("telegramAlertsLabel")}
        </label>
        <p className="mt-1 text-sm text-muted-foreground">{t("telegramAlertsHint")}</p>
      </Card>
    </>
  );
}

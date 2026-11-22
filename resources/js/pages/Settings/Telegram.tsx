import { Head, router, useForm, usePage } from "@inertiajs/react";
import { type FormEvent, useEffect, useState } from "react";
import ConfirmDelete from "../../components/ConfirmDelete";
import PageHeader from "../../components/PageHeader";
import SettingsSection from "../../components/settings/SettingsSection";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { Checkbox } from "../../components/ui/checkbox";
import { Label } from "../../components/ui/label";
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

      <Alert variant="info">
        <AlertDescription>
          <p>{t("telegramPrivacy")}</p>
          {edition === "personal" && <p className="mt-2">{t("telegramNeedsInternet")}</p>}
        </AlertDescription>
      </Alert>

      <div className="flex flex-col">
        <SettingsSection
          title={t("telegramConnect")}
          description={`${t("telegramConnectHint")} (@${username})`}
          tour="telegram-connect"
        >
          {created && !dismissed && (
            <div className="flex flex-col gap-3 rounded-lg border border-border p-4">
              <h3 className="text-h3">{t("telegramCreatedTitle")}</h3>
              <p className="text-body">{t("telegramCreatedBody")}</p>
              {expired ? (
                <p role="status" data-testid="telegram-expired" className="text-label text-expense">
                  {t("telegramExpired")}
                </p>
              ) : (
                <>
                  <p role="timer" data-testid="telegram-countdown" className="text-label tabular-nums">
                    {t("telegramExpiresIn").replace("{time}", clock(left))}
                  </p>
                  <p className="break-all">
                    <a
                      data-testid="telegram-link"
                      href={created.link}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="text-brand-text underline underline-offset-4"
                    >
                      {created.link}
                    </a>
                  </p>
                  <p className="text-body break-all">
                    {t("telegramCodeHint")} <code data-testid="telegram-code">{`/start ${created.code}`}</code>
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button asChild variant="brand">
                      <a href={created.link} target="_blank" rel="noreferrer noopener">
                        {t("telegramOpen")}
                      </a>
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => navigator.clipboard?.writeText(created.link)}
                    >
                      {t("telegramCopy")}
                    </Button>
                  </div>
                </>
              )}
              <div>
                <Button type="button" variant="outline" onClick={() => setDismissed(true)}>
                  {t("telegramDismiss")}
                </Button>
              </div>
            </div>
          )}
          <form onSubmit={submit} className="flex flex-col gap-3">
            <p className="text-body text-muted-foreground">{t("telegramLimit").replace("{max}", String(maxLinks))}</p>
            <div>
              <Button type="submit" loading={connect.processing} disabled={links.length >= maxLinks}>
                {t("telegramConnect")}
              </Button>
            </div>
          </form>
        </SettingsSection>

        <SettingsSection title={t("telegramLinksTitle")} tour="telegram-chats">
          {links.length === 0 ? (
            <p className="text-body text-muted-foreground">{t("telegramNone")}</p>
          ) : (
            <ul className="flex flex-col divide-y divide-border">
              {links.map((link) => (
                <li key={link.id} className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 py-3">
                  <div className="min-w-0">
                    <p className="text-label">{t("telegramChat").replace("{hint}", link.chatHint)}</p>
                    <p className="text-caption text-muted-foreground">
                      {t("telegramLinkedOn")} {date(link.linkedAt)} · {t("telegramLastSeen")}:{" "}
                      {link.lastSeenAt ? date(link.lastSeenAt) : t("telegramNeverSeen")} · {link.language}
                    </p>
                  </div>
                  <ConfirmDelete
                    href={`/settings/telegram/links/${link.id}`}
                    name={t("telegramChat").replace("{hint}", link.chatHint)}
                    trigger={
                      <Button type="button" variant="destructive-outline" size="sm">
                        {t("telegramDisconnect")}
                      </Button>
                    }
                  />
                </li>
              ))}
            </ul>
          )}
        </SettingsSection>

        <SettingsSection title={t("telegramAlertsTitle")} description={t("telegramAlertsHint")}>
          <div className="flex items-center gap-2">
            <Checkbox id="telegram-alerts" checked={on} onCheckedChange={(checked) => setAlerts(checked === true)} />
            <Label htmlFor="telegram-alerts" className="font-normal">
              {t("telegramAlertsLabel")}
            </Label>
          </div>
        </SettingsSection>
      </div>
    </>
  );
}

import { Link, router, usePage } from "@inertiajs/react";
import { Popover } from "radix-ui";
import { useEffect, useRef, useState } from "react";

import { useFinanceI18n } from "../lib/i18n";
import { bellLabel, fill, type NotificationParams, notificationTitle } from "../lib/notifications";
import type { SharedProps } from "../page-props";
import { Button } from "./ui/button";
import { BellButton } from "./BellButton";

/** Header bell: unread count, a popover with the latest notifications, mark as read, and a polite announcement when the count grows after a navigation. */
export default function NotificationBell() {
  const page = usePage<SharedProps>();
  const { t, money, date } = useFinanceI18n();
  const shared = page.props.notifications;
  const unread = shared?.unread ?? 0;
  const currency = page.props.workspace?.current.currency ?? "USD";
  const [open, setOpen] = useState(false);
  const [announcement, setAnnouncement] = useState("");
  const last = useRef(unread);

  useEffect(() => {
    if (unread > last.current) setAnnouncement(fill(t("notificationsLive"), { n: unread }));
    last.current = unread;
  }, [unread, t]);

  useEffect(() => router.on("navigate", () => setOpen(false)), []);

  if (!shared) return null;

  const formats = { money, date, currency };

  return (
    <>
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger asChild>
          <BellButton unread={unread} label={bellLabel(unread, t)} aria-haspopup="dialog" />
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content
            align="end"
            sideOffset={8}
            aria-label={t("notifications")}
            className="z-50 w-96 max-w-[calc(100vw-2rem)] rounded-lg border bg-popover p-3 text-popover-foreground shadow-md outline-none"
          >
            {shared.recent.length === 0 ? (
              <p className="p-2 text-sm text-muted-foreground">{t("notificationsEmpty")}</p>
            ) : (
              <ul className="flex max-h-96 flex-col divide-y divide-border overflow-y-auto">
                {shared.recent.map((n) => (
                  <li key={n.id} className="flex items-start justify-between gap-2 py-2">
                    <div className="flex flex-col gap-0.5">
                      <span className={n.read ? "text-sm text-muted-foreground" : "text-sm font-medium"}>
                        {notificationTitle(n.titleKey, n.params as NotificationParams, t, formats)}
                      </span>
                      <span className="text-xs text-muted-foreground">
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
            <div className="mt-2 flex items-center justify-between gap-2 border-t pt-2 text-sm">
              <Link href="/notifications" className="font-medium underline">
                {t("notificationsViewAll")}
              </Link>
              {unread > 0 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => router.post("/notifications/read", {}, { preserveScroll: true })}
                >
                  {t("notificationsMarkAllRead")}
                </Button>
              )}
            </div>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </>
  );
}

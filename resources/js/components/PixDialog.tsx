import { CheckIcon, CopyIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { fill, isFinal, type PixCharge, POLL_SECONDS, safeQr } from "../lib/pix";
import { Button } from "./ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "./ui/dialog";

type Props = {
  charge: PixCharge;
  onClose: () => void;
  /** Called once when the server reports the payment as received, so the page can reload its plan. */
  onPaid: () => void;
};

/**
 * The PIX charge of a payment in progress: the QR image, the copy-and-paste code, and a status line in a live region.
 * It polls the server every few seconds and stops at paid, expired or refunded; it never decides the outcome itself.
 */
export default function PixDialog({ charge, onClose, onPaid }: Props) {
  const { t, money, locale } = useFinanceI18n();
  const [status, setStatus] = useState<PixCharge["status"]>(charge.status);
  const [failed, setFailed] = useState(false);
  const [copied, setCopied] = useState<"" | "ok" | "failed">("");
  const copyTimer = useRef<number>(0);
  const paidReported = useRef(false);

  useEffect(() => {
    if (isFinal(status) || failed) return;
    const timer = window.setInterval(async () => {
      try {
        const res = await fetch(`/billing/pix/${encodeURIComponent(charge.ref)}`, {
          headers: { Accept: "application/json" },
          credentials: "same-origin",
        });
        if (res.status === 429) return; // limited: try again at the next tick
        if (!res.ok) return setFailed(true);
        const next = (await res.json()) as PixCharge;
        setStatus(next.status);
        if (next.status === "paid" && !paidReported.current) {
          paidReported.current = true;
          onPaid();
        }
      } catch {
        // a dropped connection: the next tick tries again
      }
    }, POLL_SECONDS * 1000);
    return () => window.clearInterval(timer);
  }, [charge.ref, status, failed, onPaid]);

  useEffect(() => () => window.clearTimeout(copyTimer.current), []);

  const copy = async () => {
    window.clearTimeout(copyTimer.current);
    try {
      await navigator.clipboard.writeText(charge.brCode ?? "");
      setCopied("ok");
    } catch {
      setCopied("failed");
    }
    copyTimer.current = window.setTimeout(() => setCopied(""), 2500);
  };

  const qr = safeQr(charge.qr);
  const payBefore = new Date(charge.expiresAt).toLocaleTimeString(locale, { hour: "2-digit", minute: "2-digit" });
  const message = failed
    ? t("billingPixError")
    : status === "paid"
      ? t("billingPixPaid")
      : status === "expired"
        ? t("billingPixExpired")
        : status === "refunded" || status === "disputed"
          ? t("billingPixRefunded")
          : t("billingPixWaiting");
  const waiting = status === "pending" && !failed;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-xl" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>{t("billingPixDialogTitle")}</DialogTitle>
          <DialogDescription>
            {fill(t("billingPixOption"), { months: charge.months })} · {money(charge.amountCents, "BRL")}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 sm:grid-cols-[auto_1fr] sm:items-start">
          {qr ? (
            // A QR code needs a light ground to scan, in dark mode too.
            <img src={qr} alt={t("billingPixQrAlt")} className="size-44 rounded-lg border bg-white p-2" />
          ) : (
            <div className="size-44 rounded-lg border border-dashed" aria-hidden="true" />
          )}
          <div className="flex min-w-0 flex-col gap-3">
            <p className="text-sm text-muted-foreground">{t("billingPixScan")}</p>
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t("billingPixCode")}
              </span>
              <code className="max-h-24 overflow-auto rounded-md border border-dashed bg-muted/50 p-2 text-xs break-all">
                {charge.brCode}
              </code>
              <div className="flex items-center gap-2">
                <Button type="button" variant="outline" size="sm" onClick={copy}>
                  {copied === "ok" ? <CheckIcon aria-hidden="true" /> : <CopyIcon aria-hidden="true" />}
                  {t("billingPixCopy")}
                </Button>
                <span role="status" className="text-xs text-muted-foreground">
                  {copied === "ok" ? t("billingPixCopied") : copied === "failed" ? t("billingPixCopyFailed") : ""}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div
          role="status"
          aria-live="polite"
          className={`rounded-lg px-3 py-2 text-sm ${status === "paid" ? "bg-emerald-50 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-100" : "bg-muted"}`}
        >
          {message}
          {waiting && (
            <span className="ml-1 text-muted-foreground">{fill(t("billingPixPayBefore"), { time: payBefore })}</span>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant={isFinal(status) ? "default" : "outline"} onClick={onClose}>
            {t("billingClose")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

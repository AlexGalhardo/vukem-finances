import { Head, router, usePage } from "@inertiajs/react";
import { type KeyboardEvent, useEffect, useId, useRef, useState } from "react";
import { Kbd } from "../components/Kbd";
import { MoneyText } from "../components/MoneyText";
import PageHeader from "../components/PageHeader";
import { Card, primary, secondary } from "../components/ui";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { useFinanceI18n } from "../lib/i18n";
import type { Page, Pending, SharedProps } from "../page-props";

/** Posts one action and calls `after` once the server answered, whatever the answer. */
function act(path: string, data: Record<string, string>, after: () => void): void {
  router.post(path, data, { preserveScroll: true, preserveState: true, onFinish: after });
}

export default function Suggestions({ pending, applied, aiOn, canRecord }: Page<"Suggestions">) {
  const { t, locale } = useFinanceI18n();
  const currency = usePage<SharedProps>().props.workspace?.current.currency ?? "USD";
  const [active, setActive] = useState(0);
  const [minimum, setMinimum] = useState("90");
  const [announcement, setAnnouncement] = useState("");
  const [busy, setBusy] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);
  const minId = useId();
  const current: Pending | undefined = pending[Math.min(active, Math.max(pending.length - 1, 0))];

  // Keep the active row in range and focused after a reload removed rows.
  useEffect(() => {
    if (active > pending.length - 1) setActive(Math.max(pending.length - 1, 0));
  }, [active, pending.length]);

  const focusActive = (): void => {
    requestAnimationFrame(() => listRef.current?.querySelector<HTMLElement>("[tabindex='0']")?.focus());
  };

  const run = (path: string, data: Record<string, string>, message: string): void => {
    if (busy) return;
    setBusy(true);
    act(path, data, () => {
      setBusy(false);
      setAnnouncement(message);
      focusActive();
    });
  };

  const accept = (): void => {
    if (current && canRecord) run(`/suggestions/${current.id}/accept`, {}, t("suggestionsAnnounceAccepted"));
  };
  const reject = (): void => {
    if (current && canRecord) run(`/suggestions/${current.id}/reject`, {}, t("suggestionsAnnounceRejected"));
  };
  const acceptAll = (): void => {
    if (!canRecord || pending.length === 0) return;
    const above = pending.filter((p) => p.confidence >= Number(minimum)).length;
    run("/suggestions/accept-all", { min: minimum }, t("suggestionsAnnounceMany").replace("{count}", String(above)));
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    if (event.nativeEvent.isComposing || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive((i) => Math.min(i + 1, pending.length - 1));
      focusActive();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive((i) => Math.max(i - 1, 0));
      focusActive();
    } else if (event.key === "Enter") {
      event.preventDefault();
      accept();
    } else if (event.key === "Backspace") {
      event.preventDefault();
      reject();
    } else if (event.key === "A" && event.shiftKey) {
      // Shift+A, not a plain "a": that key starts the `a s` sequence of the assistant.
      event.preventDefault();
      acceptAll();
    }
  };

  const sourceLabel = (source: string): string =>
    source === "rule" ? t("suggestionsSourceRule") : t("suggestionsSourceAi");

  return (
    <>
      <Head title={t("suggestions")} />
      <PageHeader
        title={t("suggestions")}
        description={t("suggestionsIntro")}
        actions={
          canRecord && (
            <button
              type="button"
              className={primary}
              disabled={busy}
              onClick={() => run("/suggestions/run", {}, t("suggestionsRun"))}
            >
              {t("suggestionsRun")}
            </button>
          )
        }
      />
      {!aiOn && <p className="rounded-lg border bg-muted p-4 text-sm">{t("suggestionsAiOff")}</p>}
      {!canRecord && <p className="rounded-lg border bg-muted p-4 text-sm">{t("suggestionsViewerNote")}</p>}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
      <Card>
        {pending.length === 0 ? (
          <p className="text-muted-foreground">{t("suggestionsEmpty")}</p>
        ) : (
          <div className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">{t("suggestionsKeys")}</p>
            <div className="flex flex-wrap items-end gap-2">
              <button type="button" className={primary} disabled={!canRecord || busy} onClick={accept}>
                {t("suggestionsAccept")} <Kbd>Enter</Kbd>
              </button>
              <button type="button" className={secondary} disabled={!canRecord || busy} onClick={reject}>
                {t("suggestionsReject")} <Kbd>Backspace</Kbd>
              </button>
              <div className="flex items-end gap-2">
                <div className="flex flex-col gap-1">
                  <label htmlFor={minId} className="text-xs text-muted-foreground">
                    {t("suggestionsMin")}
                  </label>
                  <Input
                    id={minId}
                    type="number"
                    min={0}
                    max={100}
                    className="w-24"
                    value={minimum}
                    onChange={(e) => setMinimum(e.target.value)}
                  />
                </div>
                <button type="button" className={secondary} disabled={!canRecord || busy} onClick={acceptAll}>
                  {t("suggestionsAcceptAll")} {minimum}% <Kbd>Shift+A</Kbd>
                </button>
              </div>
            </div>
            <div
              ref={listRef}
              role="listbox"
              aria-label={t("suggestionsList")}
              className="divide-y divide-border rounded-lg border"
            >
              {pending.map((p, i) => {
                const selected = p.id === current?.id;
                return (
                  <div
                    key={p.id}
                    id={`suggestion-${p.id}`}
                    role="option"
                    aria-selected={selected}
                    tabIndex={selected ? 0 : -1}
                    onClick={() => setActive(i)}
                    onKeyDown={onKeyDown}
                    className="flex cursor-pointer flex-wrap items-center justify-between gap-3 px-4 py-3 outline-none focus-visible:ring-2 focus-visible:ring-ring aria-selected:bg-muted"
                  >
                    <span className="flex min-w-0 flex-col">
                      <span className="truncate font-medium">{p.description || "-"}</span>
                      <span className="text-xs text-muted-foreground">
                        {p.date} ·{" "}
                        <MoneyText
                          cents={p.kind === "expense" ? -p.amountCents : p.amountCents}
                          currency={currency}
                          locale={locale}
                        />
                      </span>
                    </span>
                    <span className="flex items-center gap-2">
                      <Badge variant="secondary">{p.categoryName}</Badge>
                      <span className="text-xs text-muted-foreground">
                        {sourceLabel(p.source)} · {t("suggestionsConfidence")} {p.confidence}%
                      </span>
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </Card>
      {applied.length > 0 && (
        <Card title={t("suggestionsAppliedTitle")}>
          <ul className="divide-y divide-border">
            {applied.map((p) => (
              <li key={p.id} className="flex flex-wrap items-center justify-between gap-3 py-3">
                <span className="flex flex-col">
                  <span className="font-medium">{p.description || "-"}</span>
                  <span className="text-xs text-muted-foreground">
                    {p.categoryName} · {sourceLabel(p.source)} · {p.confidence}% · {p.appliedAt}
                  </span>
                </span>
                {canRecord && (
                  <button
                    type="button"
                    className={secondary}
                    disabled={busy}
                    onClick={() => run(`/suggestions/${p.id}/undo`, {}, t("suggestionsAnnounceUndone"))}
                  >
                    {t("suggestionsUndo")}
                  </button>
                )}
              </li>
            ))}
          </ul>
        </Card>
      )}
    </>
  );
}

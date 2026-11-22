import { CircleAlert, CircleCheck, X } from "lucide-react";
import { createContext, type ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";

export type ToastTone = "status" | "error";

export interface ToastAction {
  label: string;
  onAction: () => void;
}

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
  action?: ToastAction;
}

interface ToastApi {
  push(message: string, tone?: ToastTone, action?: ToastAction): void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function useToast(): ToastApi {
  const api = useContext(ToastContext);
  if (!api) throw new Error("useToast must be used inside a ToastProvider");
  return api;
}

const DISMISS_AFTER_MS = 5000;
const DISMISS_WITH_ACTION_AFTER_MS = 8000;
const MAX_VISIBLE = 3;

/** Success and info go after 5 s (8 s with an action) and wait while the pointer or focus is on them; errors stay. */
function ToastMessage({ item, onDismiss }: { item: ToastItem; onDismiss: (id: number) => void }) {
  const { t } = useFinanceI18n();
  const [paused, setPaused] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => {
    if (item.tone === "error" || paused) return;
    timer.current = setTimeout(() => onDismiss(item.id), item.action ? DISMISS_WITH_ACTION_AFTER_MS : DISMISS_AFTER_MS);
    return () => clearTimeout(timer.current);
  }, [item.id, item.tone, item.action, paused, onDismiss]);
  const Icon = item.tone === "error" ? CircleAlert : CircleCheck;
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: pausing the timer on hover and focus is an enhancement; the buttons inside stay the real controls
    <div
      data-side="top"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
      className="vf-float pointer-events-auto flex items-start gap-3 rounded-lg border border-border bg-popover p-3 text-body text-popover-foreground shadow-md"
    >
      <Icon
        aria-hidden="true"
        className={item.tone === "error" ? "mt-0.5 size-4 shrink-0 text-expense" : "mt-0.5 size-4 shrink-0 text-income"}
      />
      <p className="flex-1">{item.message}</p>
      {item.action && (
        <Button
          type="button"
          variant="link"
          size="sm"
          onClick={() => {
            item.action?.onAction();
            onDismiss(item.id);
          }}
        >
          {item.action.label}
        </Button>
      )}
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("dismiss")} onClick={() => onDismiss(item.id)}>
        <X aria-hidden="true" />
      </Button>
    </div>
  );
}

/** Two always-mounted live regions: polite `role="status"` for confirmations, `role="alert"` for errors. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const { t } = useFinanceI18n();
  const [items, setItems] = useState<ToastItem[]>([]);
  const dismiss = useCallback((id: number) => setItems((all) => all.filter((item) => item.id !== id)), []);
  const push = useCallback((message: string, tone: ToastTone = "status", action?: ToastAction) => {
    setItems((all) => [...all, { id: Date.now() + Math.random(), message, tone, action }].slice(-MAX_VISIBLE));
  }, []);
  const api = useMemo<ToastApi>(() => ({ push }), [push]);
  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed inset-x-4 bottom-[max(1rem,env(safe-area-inset-bottom))] z-(--vf-z-toast) flex flex-col gap-2 sm:inset-x-auto sm:end-4 sm:w-96">
        <div role="status" aria-label={t("notifications")} className="flex flex-col gap-2">
          {items
            .filter((item) => item.tone === "status")
            .map((item) => (
              <ToastMessage key={item.id} item={item} onDismiss={dismiss} />
            ))}
        </div>
        <div role="alert" className="flex flex-col gap-2">
          {items
            .filter((item) => item.tone === "error")
            .map((item) => (
              <ToastMessage key={item.id} item={item} onDismiss={dismiss} />
            ))}
        </div>
      </div>
    </ToastContext.Provider>
  );
}

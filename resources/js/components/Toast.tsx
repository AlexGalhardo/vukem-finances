import { X } from "lucide-react";
import { createContext, type ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";

export type ToastTone = "status" | "error";

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

interface ToastApi {
  push(message: string, tone?: ToastTone): void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function useToast(): ToastApi {
  const api = useContext(ToastContext);
  if (!api) throw new Error("useToast must be used inside a ToastProvider");
  return api;
}

const DISMISS_AFTER_MS = 6000;

function ToastMessage({ item, onDismiss }: { item: ToastItem; onDismiss: (id: number) => void }) {
  const { t } = useFinanceI18n();
  useEffect(() => {
    const handle = setTimeout(() => onDismiss(item.id), DISMISS_AFTER_MS);
    return () => clearTimeout(handle);
  }, [item.id, onDismiss]);
  return (
    <div className="pointer-events-auto flex items-start gap-3 rounded-lg border bg-popover p-3 text-sm text-popover-foreground shadow-md">
      <p className="flex-1">{item.message}</p>
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
  const push = useCallback((message: string, tone: ToastTone = "status") => {
    setItems((all) => [...all, { id: Date.now() + Math.random(), message, tone }]);
  }, []);
  const api = useMemo<ToastApi>(() => ({ push }), [push]);
  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed right-4 bottom-4 z-[60] flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2">
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

import { router, usePage } from "@inertiajs/react";
import { createContext, type ReactNode, useCallback, useContext, useMemo, useRef, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import type { SharedProps } from "../../page-props";
import { useAction } from "../../shortcuts/useAction";
import { type EntryKind, loadOptions, type QuickAddOptions } from "./entryForm";
import { focusTransactionSearch } from "./searchFocus";
import TransactionDialog from "./TransactionDialog";
import TransferDialog, { type TransferEdit } from "./TransferDialog";

interface QuickAddApi {
  openTransaction: (kind: EntryKind) => void;
  openTransfer: (edit?: TransferEdit) => void;
  /** False for a viewer: the server refuses the writes, and the controls are not offered. */
  canRecord: boolean;
}

const QuickAddContext = createContext<QuickAddApi | null>(null);

export function useQuickAdd(): QuickAddApi {
  const api = useContext(QuickAddContext);
  if (!api) throw new Error("useQuickAdd must be used inside QuickAdd");
  return api;
}

type Open = { dialog: "transaction"; kind: EntryKind } | { dialog: "transfer"; edit: TransferEdit | null } | null;

/**
 * Owns the new expense, new income and new transfer dialogs for every signed-in page and registers their actions
 * (`n e`, `n i`, `n t`) through the shortcut registry. The pickers are fetched when a dialog opens, so they are
 * current on any page, and a save reloads the page underneath.
 */
export default function QuickAdd({ children }: { children: ReactNode }) {
  const { t } = useFinanceI18n();
  const role = usePage<SharedProps>().props.workspace?.role;
  const canRecord = role !== undefined && role !== "viewer";
  const [open, setOpen] = useState<Open>(null);
  const [options, setOptions] = useState<QuickAddOptions | null>(null);
  const [announcement, setAnnouncement] = useState("");
  const loading = useRef(false);
  const opener = useRef<HTMLElement | null>(null);

  const show = useCallback(async (next: NonNullable<Open>): Promise<void> => {
    if (loading.current) return;
    loading.current = true;
    opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setOptions(await loadOptions());
    loading.current = false;
    setAnnouncement("");
    setOpen(next);
  }, []);

  const api = useMemo<QuickAddApi>(
    () => ({
      canRecord,
      openTransaction: (kind) => void show({ dialog: "transaction", kind }),
      openTransfer: (edit) => void show({ dialog: "transfer", edit: edit ?? null }),
    }),
    [canRecord, show],
  );

  useAction({
    id: "new-expense",
    labelKey: "newExpense",
    group: "actions",
    keys: "n e",
    run: () => api.openTransaction("expense"),
    enabled: () => canRecord,
  });
  useAction({
    id: "new-income",
    labelKey: "newIncome",
    group: "actions",
    keys: "n i",
    run: () => api.openTransaction("income"),
    enabled: () => canRecord,
  });
  useAction({
    id: "new-transfer",
    labelKey: "newTransfer",
    group: "actions",
    keys: "n t",
    run: () => api.openTransfer(),
    enabled: () => canRecord,
  });

  // `/` focuses the search field of the transactions page, and takes the person there from any other page.
  useAction({
    id: "search-transactions",
    labelKey: "searchTransactions",
    group: "actions",
    keys: "/",
    run: () => {
      if (!focusTransactionSearch()) router.visit("/transactions#search");
    },
  });

  const returnFocus = (): void => {
    const target = opener.current;
    if (target?.isConnected) target.focus();
  };

  const saved = (message: string): void => {
    setOpen(null);
    setAnnouncement(message);
    router.reload();
  };

  return (
    <QuickAddContext.Provider value={api}>
      {children}
      <TransactionDialog
        open={open?.dialog === "transaction"}
        onOpenChange={(next) => !next && setOpen(null)}
        returnFocus={returnFocus}
        options={options}
        kind={open?.dialog === "transaction" ? open.kind : "expense"}
        onSaved={() => saved(t("savedTransaction"))}
      />
      <TransferDialog
        open={open?.dialog === "transfer"}
        onOpenChange={(next) => !next && setOpen(null)}
        returnFocus={returnFocus}
        options={options}
        edit={open?.dialog === "transfer" ? open.edit : null}
        onSaved={(updated) => saved(t(updated ? "updatedTransfer" : "savedTransfer"))}
      />
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </QuickAddContext.Provider>
  );
}

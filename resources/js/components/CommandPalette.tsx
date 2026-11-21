import { router } from "@inertiajs/react";
import { Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { rank } from "../shortcuts/fuzzy";
import { hasModifier, parseBinding } from "../shortcuts/keys";
import type { Action, ActionGroup } from "../shortcuts/registry";
import { useSingleKeyShortcuts } from "../shortcuts/singleKey";
import { useActions } from "../shortcuts/useAction";
import { useIsMac } from "../shortcuts/useIsMac";
import { useReturnFocus } from "../shortcuts/useReturnFocus";
import Combobox, { type ComboboxItem } from "./Combobox";
import { Shortcut } from "./Kbd";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "./ui/dialog";

const SEARCH_ID = "__search-transactions";
const groupOrder: readonly ActionGroup[] = ["navigation", "actions"];

export default function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useFinanceI18n();
  const isMac = useIsMac();
  const actions = useActions();
  const [singleKeys] = useSingleKeyShortcuts();
  const [query, setQuery] = useState("");
  const returnFocus = useReturnFocus(open);
  // The chosen action runs after the dialog gave focus back, so an action that opens something keeps its focus.
  const pending = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!open) setQuery("");
  }, [open]);

  const enabled = actions.filter((action) => action.enabled?.() ?? true);
  const ranked = rank(query, enabled, (action) => t(action.labelKey));
  const matches = groupOrder.flatMap((group) => ranked.filter((action) => action.group === group));

  const trimmed = query.trim();
  const offerSearch = trimmed !== "" && matches.length === 0;

  const groupName = (group: ActionGroup): string => t(group === "navigation" ? "groupNavigation" : "groupActions");
  const items: ComboboxItem[] = matches.map((action) => ({
    id: action.id,
    label: t(action.labelKey),
    group: groupName(action.group),
    hint:
      action.keys && (singleKeys || hasModifier(parseBinding(action.keys))) ? (
        <Shortcut keys={action.keys} isMac={isMac} />
      ) : undefined,
  }));
  if (offerSearch) {
    items.push({ id: SEARCH_ID, label: `${t("searchTransactionsFor")} “${trimmed}”`, group: groupName("actions") });
  }

  function choose(item: ComboboxItem): void {
    const action: Action | undefined = actions.find((candidate) => candidate.id === item.id);
    pending.current =
      item.id === SEARCH_ID
        ? () => router.visit(`/transactions?q=${encodeURIComponent(trimmed)}`)
        : (action?.run ?? null);
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-vf-overlay=""
        showCloseButton={false}
        className="top-[15%] translate-y-0 gap-0 p-0 sm:max-w-xl"
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus();
          const run = pending.current;
          pending.current = null;
          run?.();
        }}
      >
        <DialogHeader className="sr-only">
          <DialogTitle>{t("paletteTitle")}</DialogTitle>
          <DialogDescription>{t("paletteDescription")}</DialogDescription>
        </DialogHeader>
        <div className="relative">
          <Search
            aria-hidden="true"
            className="pointer-events-none absolute top-3 left-3 size-4 text-muted-foreground"
          />
          <Combobox
            alwaysOpen
            items={items}
            query={query}
            onQueryChange={setQuery}
            onSelect={choose}
            label={t("paletteTitle")}
            placeholder={t("palettePlaceholder")}
            resultsLabel={(count) => `${count} ${t(count === 1 ? "resultOne" : "resultMany")}`}
            inputClassName="rounded-none border-0 border-b bg-transparent py-3 pl-9 focus-visible:ring-0"
            listClassName="px-2"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

import { useFinanceI18n } from "../lib/i18n";
import type { ActionGroup } from "../shortcuts/registry";
import { setSingleKeyEnabled, useSingleKeyShortcuts } from "../shortcuts/singleKey";
import { useActions } from "../shortcuts/useAction";
import { useIsMac } from "../shortcuts/useIsMac";
import { useReturnFocus } from "../shortcuts/useReturnFocus";
import { Shortcut } from "./Kbd";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";

const groups: readonly ActionGroup[] = ["navigation", "actions"];

export default function ShortcutHelp({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useFinanceI18n();
  const isMac = useIsMac();
  const actions = useActions();
  const [singleKeys] = useSingleKeyShortcuts();
  const returnFocus = useReturnFocus(open);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-vf-overlay=""
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus();
        }}
        showCloseButton={false}
        className="max-h-[85svh] overflow-y-auto sm:max-w-lg"
      >
        <DialogHeader>
          <DialogTitle>{t("shortcutsTitle")}</DialogTitle>
          <DialogDescription>{t("shortcutsDescription")}</DialogDescription>
        </DialogHeader>
        <p className="text-sm font-medium">{t("shortcutsNote")}</p>
        {groups.map((group) => {
          const rows = actions.filter((action) => action.group === group);
          if (rows.length === 0) return null;
          return (
            <table key={group} className="w-full text-sm">
              <caption className="pb-2 text-left text-xs font-medium tracking-wide text-muted-foreground uppercase">
                {t(group === "navigation" ? "groupNavigation" : "groupActions")}
              </caption>
              <thead className="sr-only">
                <tr>
                  <th scope="col">{t("shortcutAction")}</th>
                  <th scope="col">{t("shortcutKeys")}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((action) => (
                  <tr key={action.id} className="border-t">
                    <th scope="row" className="py-2 pr-3 text-left font-normal">
                      {t(action.labelKey)}
                    </th>
                    <td className="py-2 text-right">
                      {action.keys ? (
                        <Shortcut keys={action.keys} isMac={isMac} />
                      ) : (
                        <span className="text-muted-foreground">{t("paletteOnly")}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          );
        })}
        <label className="flex items-start gap-3 text-sm">
          <input
            type="checkbox"
            checked={singleKeys}
            onChange={(event) => setSingleKeyEnabled(event.target.checked)}
            className="mt-0.5 size-4 accent-primary pointer-coarse:size-6"
          />
          <span>
            <span className="font-medium">{t("singleKey")}</span>
            <span className="block text-muted-foreground">{t("singleKeyHelp")}</span>
          </span>
        </label>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">{t("close")}</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

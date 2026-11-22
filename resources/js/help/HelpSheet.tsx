import type { ReactElement, ReactNode } from "react";
import { Shortcut } from "../components/Kbd";
import { Button } from "../components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "../components/ui/sheet";
import { useFinanceI18n } from "../lib/i18n";
import { useActions } from "../shortcuts/useAction";
import { useIsMac } from "../shortcuts/useIsMac";

export interface HelpSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The screen name. */
  title: string;
  /** Why the screen exists, already translated. */
  why?: string;
  /** What to do here, already translated. */
  steps?: readonly { title: string; body: string }[];
  /** Ids of registered actions whose shortcuts apply to this screen. */
  shortcutActions?: readonly string[];
  /** Extra controls, such as the tutorial buttons. */
  footer?: ReactNode;
}

/** The help of one screen: why it exists, the steps, its shortcuts and the tutorial buttons. Opened by `h p`. */
export function HelpSheet({
  open,
  onOpenChange,
  title,
  why,
  steps = [],
  shortcutActions = [],
  footer,
}: HelpSheetProps): ReactElement {
  const { t } = useFinanceI18n();
  const isMac = useIsMac();
  const actions = useActions();
  const shown = shortcutActions.flatMap((id) => {
    const action = actions.find((candidate) => candidate.id === id);
    return action?.keys ? [{ id: action.id, labelKey: action.labelKey, keys: action.keys }] : [];
  });
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent data-vf-overlay className="overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription className="sr-only">{t("helpPageHeading")}</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col gap-6 px-4 pb-4">
          {why && (
            <section className="flex flex-col gap-1">
              <h3 className="text-h3">{t("helpWhy")}</h3>
              <p className="text-body text-pretty text-muted-foreground">{why}</p>
            </section>
          )}
          {steps.length > 0 && (
            <ol className="flex list-decimal flex-col gap-3 ps-5 text-body marker:text-muted-foreground">
              {steps.map((step) => (
                <li key={step.title}>
                  <p className="text-label">{step.title}</p>
                  <p className="text-pretty text-muted-foreground">{step.body}</p>
                </li>
              ))}
            </ol>
          )}
          {shown.length > 0 && (
            <section className="flex flex-col gap-2">
              <h3 className="text-h3">{t("helpShortcuts")}</h3>
              <ul className="flex flex-col gap-2 text-body">
                {shown.map((action) => (
                  <li key={action.id} className="flex items-center justify-between gap-3">
                    <span>{t(action.labelKey)}</span>
                    <Shortcut keys={action.keys} isMac={isMac} />
                  </li>
                ))}
              </ul>
            </section>
          )}
          {footer}
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} className="self-start">
            {t("helpClose")}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}

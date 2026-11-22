import { CircleHelp } from "lucide-react";
import { type ReactElement, useId } from "react";
import { Button } from "../components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "../components/ui/popover";
import type { Key } from "../i18n/keys";
import { useFinanceI18n } from "../lib/i18n";
import { useHelpActions } from "./context";
import { type HelpId, helpBodyKey, helpTitleKey } from "./ids";

export interface HelpHintProps {
  /** Key in the `help` i18n domain, resolved to `<id>Title` and `<id>Body`. */
  id: HelpId;
  /** Accessible name; defaults to "About <title>", for example "About Total balance". */
  labelKey?: Key;
  side?: "top" | "right" | "bottom" | "left";
  /** Opens the page tutorial at this step, written `<screen id>:<step index>`. */
  tutorialStep?: string;
}

/**
 * A toggletip: a visible CircleHelp button that opens one to three sentences on click, tap or Enter. Opening does
 * not trap focus, Esc closes and returns focus to the button, and what a person needs to act is never only here.
 */
export function HelpHint({ id, labelKey, side = "top", tutorialStep }: HelpHintProps): ReactElement {
  const { t } = useFinanceI18n();
  const { openTutorialStep } = useHelpActions();
  const titleId = useId();
  const [screen, step] = tutorialStep?.split(":") ?? [];
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={labelKey ? t(labelKey) : `${t("helpAbout")} ${t(helpTitleKey(id))}`}
          className="size-6 text-subtle hover:text-foreground"
          data-print="hide"
        >
          <CircleHelp aria-hidden="true" className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        side={side}
        role="dialog"
        aria-labelledby={titleId}
        onOpenAutoFocus={(event) => event.preventDefault()}
        className="flex flex-col gap-2"
      >
        <h3 id={titleId} className="text-h3">
          {t(helpTitleKey(id))}
        </h3>
        <p className="text-body text-pretty text-muted-foreground">{t(helpBodyKey(id))}</p>
        {screen && step && openTutorialStep && (
          <Button
            type="button"
            variant="link"
            size="sm"
            className="self-start px-0"
            onClick={() => openTutorialStep(screen, step)}
          >
            {t("helpLearnMore")}
          </Button>
        )}
      </PopoverContent>
    </Popover>
  );
}

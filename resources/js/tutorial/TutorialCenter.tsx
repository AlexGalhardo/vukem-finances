import { CircleCheck, Minus } from "lucide-react";
import { type ReactElement, useState } from "react";
import { Button } from "../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { useFinanceI18n } from "../lib/i18n";
import { useTutorial } from "./useTutorial";

/** Every screen of the tour with its progress, one button to start it, and Restart with a confirmation. */
export default function TutorialCenter({ onStart }: { onStart?: () => void }): ReactElement {
  const tutorial = useTutorial();
  const { t } = useFinanceI18n();
  const [confirming, setConfirming] = useState(false);
  const { state, screens } = tutorial;
  return (
    <section className="flex flex-col gap-3" data-slot="tutorial-center">
      <h3 className="text-h3">{t("tutorialStart")}</h3>
      <ul className="flex flex-col gap-1">
        {screens.map((screen) => {
          const done = state.completed[screen.id] === true;
          const skipped = state.skipped[screen.id] === true;
          const name = screen.nameKey ? t(screen.nameKey) : screen.id;
          return (
            <li key={screen.id} className="flex items-center justify-between gap-2 text-body">
              <span className="flex items-center gap-2">
                {done ? (
                  <CircleCheck aria-hidden="true" className="size-4 text-brand" />
                ) : (
                  <Minus aria-hidden="true" className="size-4 text-subtle" />
                )}
                {name}
                {skipped && !done && <span className="sr-only">{t("tutorialSkipScreen")}</span>}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={`${t("tutorialShowScreen")}: ${name}`}
                onClick={() => {
                  tutorial.start(screen.id);
                  onStart?.();
                }}
              >
                {t("tutorialStart")}
              </Button>
            </li>
          );
        })}
      </ul>
      <Button type="button" variant="outline" size="sm" className="self-start" onClick={() => setConfirming(true)}>
        {t("tutorialRestart")}
      </Button>
      <Dialog open={confirming} onOpenChange={setConfirming}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tutorialRestart")}</DialogTitle>
            <DialogDescription>{t("tutorialRestartConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setConfirming(false)}>
              {t("cancel")}
            </Button>
            <Button
              type="button"
              onClick={() => {
                tutorial.restart();
                setConfirming(false);
              }}
            >
              {t("tutorialRestart")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}

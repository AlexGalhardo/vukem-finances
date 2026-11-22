import { usePage } from "@inertiajs/react";
import { X } from "lucide-react";
import { Dialog } from "radix-ui";
import type { ReactElement } from "react";
import { Button } from "../components/ui/button";
import { screenIdForPath } from "../help/screens";
import { useFinanceI18n } from "../lib/i18n";
import { shellActionSpecs } from "../shortcuts/builtin";
import { useAction } from "../shortcuts/useAction";
import { helpAction } from "./state";
import TutorialStepCard from "./TutorialStepCard";
import { useTutorial } from "./useTutorial";

function OpenTutorialAction(): null {
  const tutorial = useTutorial();
  const spec = shellActionSpecs.find((candidate) => candidate.id === "open-tutorial");
  if (!spec) throw new Error("the open-tutorial action spec is missing");
  useAction({
    ...spec,
    run: () => (helpAction(tutorial.state) === "resume" ? tutorial.resume() : tutorial.start()),
    enabled: () => helpAction(tutorial.state) !== "none",
  });
  return null;
}

/** The last card of a run: where to go next. Non-modal, like the step card. */
function DoneCard({ onClose }: { onClose: () => void }): ReactElement {
  const { t } = useFinanceI18n();
  return (
    <Dialog.Root open modal={false} onOpenChange={(open) => !open && onClose()}>
      <Dialog.Portal>
        <Dialog.Content
          data-slot="tutorial-step"
          data-vf-overlay=""
          aria-describedby="tutorial-done-body"
          onOpenAutoFocus={(event) => event.preventDefault()}
          onCloseAutoFocus={(event) => event.preventDefault()}
          onInteractOutside={(event) => event.preventDefault()}
          className="vf-overlay fixed inset-x-4 bottom-4 z-(--vf-z-tutorial) flex flex-col gap-3 rounded-lg border border-border bg-popover p-4 shadow-md sm:inset-auto sm:start-1/2 sm:top-1/2 sm:w-[340px] sm:-translate-x-1/2 sm:-translate-y-1/2"
        >
          <div className="flex items-start justify-between gap-2">
            <Dialog.Title className="text-h3">{t("tutorialDoneTitle")}</Dialog.Title>
            <Button type="button" variant="ghost" size="icon-xs" aria-label={t("tutorialClose")} onClick={onClose}>
              <X aria-hidden="true" />
            </Button>
          </div>
          <p id="tutorial-done-body" className="text-body text-pretty text-muted-foreground">
            {t("tutorialDoneBody")}
          </p>
          <Button type="button" size="sm" className="self-start" onClick={onClose}>
            {t("helpClose")}
          </Button>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

/** Mounted once in the Shell: the step card while a run is going on its own screen, the final card after it. */
export default function TutorialHost(): ReactElement {
  const tutorial = useTutorial();
  const page = usePage();
  const onScreen = tutorial.current !== null && screenIdForPath(page.url) === tutorial.current.id;
  return (
    <>
      <OpenTutorialAction />
      {tutorial.state.status === "running" && onScreen && <TutorialStepCard />}
      {tutorial.finished && tutorial.state.status === "done" && <DoneCard onClose={tutorial.dismissFinished} />}
    </>
  );
}

import { X } from "lucide-react";
import { Dialog, Popover } from "radix-ui";
import { type ReactElement, useEffect, useRef, useState } from "react";
import { Button } from "../components/ui/button";
import { Progress } from "../components/ui/progress";
import { useFinanceI18n } from "../lib/i18n";
import { useTutorial } from "./useTutorial";

/** How long to wait for the highlighted element before showing the card without it. */
const WAIT_MS = 1000;

interface Found {
  element: HTMLElement | null;
  settled: boolean;
  /** The step this result belongs to: a result of the previous step never counts. */
  key: string;
}

/** Finds `[data-tour=<target>]`, waiting up to a second with a MutationObserver (no polling). */
function useTarget(target: string, key: string): Found {
  const [found, setFound] = useState<Found>({ element: null, settled: false, key: "" });
  useEffect(() => {
    const query = (): HTMLElement | null => document.querySelector<HTMLElement>(`[data-tour="${target}"]`);
    const first = query();
    if (first) {
      setFound({ element: first, settled: true, key });
      return;
    }
    setFound({ element: null, settled: false, key });
    const observer = new MutationObserver(() => {
      const element = query();
      if (element) {
        observer.disconnect();
        clearTimeout(timer);
        setFound({ element, settled: true, key });
      }
    });
    const timer = setTimeout(() => {
      observer.disconnect();
      setFound({ element: null, settled: true, key });
    }, WAIT_MS);
    observer.observe(document.body, { childList: true, subtree: true });
    return () => {
      observer.disconnect();
      clearTimeout(timer);
    };
  }, [target, key]);
  return found.key === key ? found : { element: null, settled: false, key };
}

function useWide(): boolean {
  const [wide, setWide] = useState(true);
  useEffect(() => {
    const query = matchMedia("(min-width: 40rem)");
    const update = (): void => setWide(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return wide;
}

/** Marks the target while its step is shown, scrolls it into view and clears the mark afterwards. */
function useHighlight(element: HTMLElement | null): void {
  useEffect(() => {
    if (!element) return;
    element.setAttribute("data-tour-active", "");
    const smooth = matchMedia("(prefers-reduced-motion: no-preference)").matches;
    element.scrollIntoView({ block: "center", behavior: smooth ? "smooth" : "auto" });
    return () => element.removeAttribute("data-tour-active");
  }, [element]);
}

const cardClass =
  "z-(--vf-z-tutorial) flex w-[340px] max-w-[calc(100vw-2rem)] flex-col gap-3 rounded-lg border border-border bg-popover p-4 text-popover-foreground shadow-md outline-none";

/** The step card of the running tutorial: anchored to its target, or centered (a bottom sheet on phones). */
export default function TutorialStepCard(): ReactElement | null {
  const tutorial = useTutorial();
  const { t } = useFinanceI18n();
  const { state, current, step, screens } = tutorial;
  const wide = useWide();
  const heading = useRef<HTMLHeadingElement | null>(null);
  const key = `${state.screenId}:${state.stepIndex}`;
  const { element, settled } = useTarget(step?.target ?? "", key);
  useHighlight(element);

  // The card lives in a portal that mounts a render later, so the heading may not exist yet when the step appears:
  // the effect records the wish and the ref callback below fulfils it as soon as the heading is there.
  const wantsFocus = useRef(false);
  // biome-ignore lint/correctness/useExhaustiveDependencies: key is the step that just appeared
  useEffect(() => {
    if (!settled || !tutorial.takeFocusRequest()) return;
    wantsFocus.current = true;
    if (heading.current) {
      heading.current.focus();
      wantsFocus.current = false;
    }
  }, [settled, key]);
  const headingRef = (node: HTMLHeadingElement | null): void => {
    heading.current = node;
    if (node && wantsFocus.current) {
      node.focus();
      wantsFocus.current = false;
    }
  };

  if (!current || !step || !settled) return null;

  const order = current.core ? screens.filter((screen) => screen.core) : [current];
  const following = order[order.findIndex((screen) => screen.id === current.id) + 1];
  const last = state.stepIndex === current.steps.length - 1;
  const first = state.stepIndex === 0;
  const canGoBack = !first || order[0]?.id !== current.id;

  const titleId = "tutorial-step-title";
  const body = (
    <>
      <div className="flex items-start justify-between gap-2">
        <p className="text-caption text-muted-foreground">
          {t("tutorialStepOf")
            .replace("{current}", String(state.stepIndex + 1))
            .replace("{total}", String(current.steps.length))}
        </p>
        <Button type="button" variant="ghost" size="icon-xs" aria-label={t("tutorialClose")} onClick={tutorial.close}>
          <X aria-hidden="true" />
        </Button>
      </div>
      <Progress value={state.stepIndex + 1} max={current.steps.length} aria-label={t("tutorialStepOf")} />
      {first && (
        <div className="flex flex-col gap-1 rounded-md bg-muted p-3">
          <p className="text-overline text-muted-foreground">{t("tutorialWhy")}</p>
          <p className="text-body text-pretty">{t(current.whyKey)}</p>
        </div>
      )}
      <h3 id={titleId} ref={headingRef} tabIndex={-1} className="text-h3 outline-none">
        {t(step.titleKey)}
      </h3>
      <p className="text-body text-pretty text-muted-foreground">{t(step.bodyKey)}</p>
      {!element && <p className="text-caption text-subtle">{t("tutorialMissingTarget")}</p>}
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" variant="outline" size="sm" disabled={!canGoBack} onClick={tutorial.back}>
          {t("tutorialBack")}
        </Button>
        <Button type="button" size="sm" onClick={tutorial.next}>
          {last && following?.nameKey
            ? t("tutorialNextScreen").replace("{screen}", t(following.nameKey))
            : t("tutorialNext")}
        </Button>
        <Button type="button" variant="ghost" size="sm" className="ms-auto" onClick={tutorial.skipScreen}>
          {t("tutorialSkipScreen")}
        </Button>
      </div>
    </>
  );

  // Non-modal on purpose: the page stays usable, outside clicks do nothing, and the shortcut registry keeps
  // working (data-vf-overlay). Esc pauses through the primitive, so the tutorial has no keydown listener.
  if (element && wide) {
    return (
      <Popover.Root open modal={false} onOpenChange={(open) => !open && tutorial.close()}>
        <Popover.Anchor virtualRef={{ current: element }} />
        <Popover.Portal>
          <Popover.Content
            data-slot="tutorial-step"
            data-vf-overlay=""
            role="dialog"
            aria-labelledby={titleId}
            side={step.side ?? "bottom"}
            align="start"
            sideOffset={12}
            collisionPadding={16}
            onOpenAutoFocus={(event) => event.preventDefault()}
            onCloseAutoFocus={(event) => event.preventDefault()}
            onInteractOutside={(event) => event.preventDefault()}
            className={`vf-float ${cardClass}`}
          >
            {body}
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    );
  }

  return (
    <Dialog.Root open modal={false} onOpenChange={(open) => !open && tutorial.close()}>
      <Dialog.Portal>
        <Dialog.Content
          data-slot="tutorial-step"
          data-vf-overlay=""
          aria-labelledby={titleId}
          aria-describedby={undefined}
          onOpenAutoFocus={(event) => event.preventDefault()}
          onCloseAutoFocus={(event) => event.preventDefault()}
          onInteractOutside={(event) => event.preventDefault()}
          className={`vf-overlay ${cardClass} fixed inset-x-4 bottom-4 w-auto max-w-none sm:inset-auto sm:start-1/2 sm:top-1/2 sm:w-[340px] sm:-translate-x-1/2 sm:-translate-y-1/2`}
        >
          {body}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

import { router, usePage } from "@inertiajs/react";
import {
  createContext,
  type ReactElement,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { ToastProvider, useToast } from "../components/Toast";
import { HelpActionsContext } from "../help/context";
import { screenIdForPath } from "../help/screens";
import { useFinanceI18n } from "../lib/i18n";
import type { SharedProps } from "../page-props";
import { tutorialScreens } from "./screens";
import * as machine from "./state";
import { browserStorage, loadState, saveState } from "./store";
import type { TutorialScreen, TutorialState, TutorialStep } from "./types";

export interface TutorialApi {
  state: TutorialState;
  /** The tour screens this person can see, in order. */
  screens: readonly TutorialScreen[];
  current: TutorialScreen | null;
  step: TutorialStep | null;
  /** Set when a run just ended, for the final card. */
  finished: boolean;
  start(screenId?: string, stepIndex?: number): void;
  next(): void;
  back(): void;
  skipScreen(): void;
  /** Close or Esc: pauses and keeps the position. */
  close(): void;
  resume(): void;
  restart(): void;
  dismissWelcome(): void;
  dismissFinished(): void;
  /** True once, right after the person started or moved the tutorial: the step card may take focus. */
  takeFocusRequest(): boolean;
}

const TutorialContext = createContext<TutorialApi | null>(null);

export function useTutorial(): TutorialApi {
  const api = useContext(TutorialContext);
  if (!api) throw new Error("useTutorial must be used inside a TutorialProvider");
  return api;
}

function routeOf(screenId: string): string | undefined {
  return tutorialScreens.find((screen) => screen.id === screenId)?.route;
}

function TutorialStore({ children }: { children: ReactNode }): ReactElement {
  const page = usePage<SharedProps>();
  const { t } = useFinanceI18n();
  const toast = useToast();
  const userId = page.props.auth?.user?.id;
  const propsRef = useRef(page.props);
  propsRef.current = page.props;
  // The rules read the edition and the role only, so the list is rebuilt when those change, not on every visit.
  const rulesKey = `${page.props.edition}|${page.props.workspace?.role}|${page.props.workspace?.admin}`;
  // biome-ignore lint/correctness/useExhaustiveDependencies: rulesKey stands for the props the rules read
  const screens = useMemo(() => machine.visibleScreens(propsRef.current), [rulesKey]);
  const [state, setState] = useState<TutorialState>(machine.initialState);
  const [loadedFor, setLoadedFor] = useState<number | null>(null);
  const [finished, setFinished] = useState(false);
  const latest = useRef({ state, screens, url: page.url });
  latest.current = { state, screens, url: page.url };
  const expected = useRef<string | null>(null);
  const focusRequest = useRef(false);

  // Storage is read after mount so server and client markup match.
  useEffect(() => {
    if (userId === undefined) return;
    setState(loadState(browserStorage(), userId));
    setLoadedFor(userId);
  }, [userId]);

  useEffect(() => {
    if (userId !== undefined && loadedFor === userId) saveState(browserStorage(), userId, state);
  }, [state, userId, loadedFor]);

  const apply = useCallback((next: TutorialState, byPerson: boolean) => {
    if (byPerson) focusRequest.current = true;
    setState(next);
    setFinished(next.status === "done" && latest.current.state.status === "running");
    if (next.status !== "running" || next.screenId === null) return;
    const route = routeOf(next.screenId);
    if (route && screenIdForPath(latest.current.url) !== next.screenId) {
      expected.current = route;
      router.visit(route);
    }
  }, []);

  // Navigating away from the screen being taught pauses the run; a visit the tutorial asked for does not.
  useEffect(
    () =>
      router.on("navigate", (event) => {
        const { state: now } = latest.current;
        const path = event.detail.page.url.split("?")[0] ?? "";
        if (expected.current !== null && path === expected.current) {
          expected.current = null;
          return;
        }
        if (now.status === "running" && screenIdForPath(path) !== now.screenId) {
          setState(machine.pause(now));
          toast.push(t("tutorialPaused"));
        }
      }),
    [toast, t],
  );

  const api = useMemo<TutorialApi>(() => {
    const current = screens.find((screen) => screen.id === state.screenId) ?? null;
    return {
      state,
      screens,
      current,
      step: current?.steps[state.stepIndex] ?? null,
      finished,
      start: (screenId, stepIndex = 0) => {
        const started = machine.start(latest.current.state, latest.current.screens, screenId);
        apply(stepIndex > 0 ? { ...started, stepIndex } : started, true);
      },
      next: () => apply(machine.next(latest.current.state, latest.current.screens), true),
      back: () => apply(machine.back(latest.current.state, latest.current.screens), true),
      skipScreen: () => apply(machine.skipScreen(latest.current.state, latest.current.screens), true),
      close: () => {
        apply(machine.pause(latest.current.state), false);
        toast.push(t("tutorialPaused"));
        requestAnimationFrame(() => document.querySelector<HTMLElement>('[data-tour="help-button"]')?.focus());
      },
      resume: () => apply(machine.resume(latest.current.state, latest.current.screens), true),
      restart: () => apply(machine.restart(latest.current.state), false),
      dismissWelcome: () => setState((now) => machine.dismissWelcome(now)),
      dismissFinished: () => setFinished(false),
      takeFocusRequest: () => {
        const asked = focusRequest.current;
        focusRequest.current = false;
        return asked;
      },
    };
  }, [state, screens, finished, apply, toast, t]);

  const help = useMemo(
    () => ({
      openTutorialStep: (screenId: string, step: string) => api.start(screenId, Number.parseInt(step, 10) || 0),
    }),
    [api],
  );

  return (
    <TutorialContext.Provider value={api}>
      <HelpActionsContext.Provider value={help}>{children}</HelpActionsContext.Provider>
    </TutorialContext.Provider>
  );
}

/** Holds the progress, the persistence and the navigation of the tutorial, and the toasts it needs. */
export function TutorialProvider({ children }: { children: ReactNode }): ReactElement {
  return (
    <ToastProvider>
      <TutorialStore>{children}</TutorialStore>
    </ToastProvider>
  );
}

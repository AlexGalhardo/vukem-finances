import type { SharedProps } from "../page-props";
import { tutorialScreens } from "./screens";
import type { TutorialScreen, TutorialState } from "./types";

export const initialState: TutorialState = {
  version: 1,
  status: "idle",
  screenId: null,
  stepIndex: 0,
  completed: {},
  skipped: {},
  welcomeDismissed: false,
};

/** The tour screens this person can see: edition, role, plan and feature rules applied, pages with parameters out. */
export function visibleScreens(
  props: SharedProps,
  screens: readonly TutorialScreen[] = tutorialScreens,
): TutorialScreen[] {
  const edition = props.edition === "personal" ? "personal" : "saas";
  return screens.filter(
    (screen) =>
      screen.mode === "tour" &&
      !screen.route.includes("{") &&
      (screen.edition === undefined || screen.edition === edition) &&
      (screen.visible?.(props) ?? true),
  );
}

/** The screens a run walks through: the core path from a core screen, or just the chosen optional one. */
export function sequenceFor(visible: readonly TutorialScreen[], screenId: string): TutorialScreen[] {
  const screen = visible.find((candidate) => candidate.id === screenId);
  if (!screen) return [];
  return screen.core ? visible.filter((candidate) => candidate.core) : [screen];
}

function running(state: TutorialState, screenId: string, stepIndex: number): TutorialState {
  return { ...state, status: "running", screenId, stepIndex };
}

function advance(
  state: TutorialState,
  visible: readonly TutorialScreen[],
  mark: "completed" | "skipped",
): TutorialState {
  const current = state.screenId;
  if (current === null) return state;
  const marked: TutorialState = { ...state, [mark]: { ...state[mark], [current]: true } };
  const order = sequenceFor(visible, current);
  const next = order[order.findIndex((screen) => screen.id === current) + 1];
  if (!next) return { ...marked, status: "done", screenId: null, stepIndex: 0 };
  return running(marked, next.id, 0);
}

/** Starts at a screen, or at the first core screen. */
export function start(state: TutorialState, visible: readonly TutorialScreen[], screenId?: string): TutorialState {
  const first = screenId ?? visible.find((screen) => screen.core)?.id ?? visible[0]?.id;
  if (!first || !visible.some((screen) => screen.id === first)) return state;
  return running({ ...state, welcomeDismissed: true }, first, 0);
}

/** Next step; on the last step of a screen, the next screen of the run, or done. */
export function next(state: TutorialState, visible: readonly TutorialScreen[]): TutorialState {
  const screen = visible.find((candidate) => candidate.id === state.screenId);
  if (!screen || state.status !== "running") return state;
  if (state.stepIndex < screen.steps.length - 1) return running(state, screen.id, state.stepIndex + 1);
  return advance(state, visible, "completed");
}

/** Previous step, also across screens: the first step goes to the last step of the previous screen. */
export function back(state: TutorialState, visible: readonly TutorialScreen[]): TutorialState {
  const screen = visible.find((candidate) => candidate.id === state.screenId);
  if (!screen || state.status !== "running") return state;
  if (state.stepIndex > 0) return running(state, screen.id, state.stepIndex - 1);
  const order = sequenceFor(visible, screen.id);
  const previous = order[order.findIndex((candidate) => candidate.id === screen.id) - 1];
  return previous ? running(state, previous.id, previous.steps.length - 1) : state;
}

export function skipScreen(state: TutorialState, visible: readonly TutorialScreen[]): TutorialState {
  return state.status === "running" ? advance(state, visible, "skipped") : state;
}

/** Close, Esc or navigating away: the position is kept. */
export function pause(state: TutorialState): TutorialState {
  return state.status === "running" ? { ...state, status: "paused" } : state;
}

/** Resume at the saved screen and step; a step that no longer exists restarts that screen. */
export function resume(state: TutorialState, visible: readonly TutorialScreen[]): TutorialState {
  const screen = visible.find((candidate) => candidate.id === state.screenId);
  if (!screen) return state;
  return running(state, screen.id, state.stepIndex < screen.steps.length ? state.stepIndex : 0);
}

/** Clears progress only. */
export function restart(state: TutorialState): TutorialState {
  return { ...initialState, welcomeDismissed: state.welcomeDismissed };
}

export function dismissWelcome(state: TutorialState): TutorialState {
  return { ...state, welcomeDismissed: true };
}

/** What the Help menu offers: start, resume, or nothing when the run is going. */
export function helpAction(state: TutorialState): "start" | "resume" | "none" {
  if (state.status === "paused") return "resume";
  if (state.status === "running") return "none";
  return "start";
}

import type { Key } from "../i18n/keys";
import type { SharedProps } from "../page-props";

export interface TutorialStep {
  /** Value of the data-tour attribute on the element to highlight. */
  target: string;
  titleKey: Key;
  bodyKey: Key;
  side?: "top" | "right" | "bottom" | "left";
}

export interface TutorialScreen {
  id: string;
  route: string;
  whyKey: Key;
  /** `tour` is a step card on a signed-in screen; `guide` is a collapsed "How this works" block under a form. */
  mode: "tour" | "guide";
  /** The name shown in "Next: ..." and in the tutorial center (tour screens). */
  nameKey?: Key;
  /** Part of the first run (Dashboard through Reports); the others are offered one by one. */
  core?: boolean;
  edition?: "saas" | "personal";
  /** Evaluated against the shared props. Hidden when false. */
  visible?: (props: SharedProps) => boolean;
  steps: readonly TutorialStep[];
}

export interface TutorialState {
  version: 1;
  status: "idle" | "running" | "paused" | "done";
  screenId: string | null;
  stepIndex: number;
  completed: Record<string, true>;
  skipped: Record<string, true>;
  welcomeDismissed: boolean;
}

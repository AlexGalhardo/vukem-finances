import { parseBinding, type Step } from "./keys";

const names: Record<string, string> = {
  left: "←",
  right: "→",
  up: "↑",
  down: "↓",
  escape: "Esc",
  space: "Space",
  tab: "Tab",
  enter: "Enter",
};

function stepLabel(step: Step, isMac: boolean): string {
  const parts: string[] = [];
  if (step.mod) parts.push(isMac ? "⌘" : "Ctrl");
  if (step.ctrl) parts.push("Ctrl");
  if (step.meta) parts.push(isMac ? "⌘" : "Win");
  if (step.alt) parts.push(isMac ? "⌥" : "Alt");
  if (step.shift) parts.push("Shift");
  parts.push(names[step.key] ?? (step.key.length === 1 ? step.key.toUpperCase() : step.key));
  return parts.join("+");
}

/** One label per key press: "g d" is two, "mod+k" is one ("Ctrl+K", "⌘+K" on macOS). */
export function formatBindingParts(keys: string, isMac: boolean): string[] {
  return parseBinding(keys).map((step) => stepLabel(step, isMac));
}

const ariaKeyNames: Record<string, string> = {
  left: "ArrowLeft",
  right: "ArrowRight",
  up: "ArrowUp",
  down: "ArrowDown",
  escape: "Escape",
  space: "Space",
  tab: "Tab",
  enter: "Enter",
};

/**
 * The value of `aria-keyshortcuts`, or undefined for a sequence: the attribute separates alternatives with spaces,
 * so it cannot say "g then d". `mod` becomes Control, or Meta on macOS; "?" needs Shift on most layouts.
 */
export function formatAriaKeyShortcuts(keys: string, isMac: boolean): string | undefined {
  const steps = parseBinding(keys);
  const step = steps[0];
  if (steps.length !== 1 || step === undefined) return undefined;
  const parts: string[] = [];
  if (step.mod) parts.push(isMac ? "Meta" : "Control");
  if (step.ctrl) parts.push("Control");
  if (step.meta) parts.push("Meta");
  if (step.alt) parts.push("Alt");
  if (step.shift || step.key === "?") parts.push("Shift");
  parts.push(ariaKeyNames[step.key] ?? (step.key.length === 1 ? step.key.toUpperCase() : step.key));
  return parts.join("+");
}

/** The spoken form for a tooltip: "Ctrl+K", or "G then D" for a sequence (`then` is translated). */
export function formatHintLabel(parts: readonly string[], then: string): string {
  return parts.join(` ${then} `);
}

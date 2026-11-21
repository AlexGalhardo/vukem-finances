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

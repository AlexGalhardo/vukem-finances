import { parseBinding, type Step } from "./keys";

const reservedWithMod = new Set([
  "t",
  "n",
  "w",
  "q",
  "r",
  "l",
  "d",
  "f",
  "p",
  "s",
  "o",
  "u",
  "h",
  "j",
  "tab",
  "pageup",
  "pagedown",
]);

/** True when the browser owns this step and a page must not take it. */
export function isReservedStep(step: Step): boolean {
  const ctrlOrMeta = step.mod || step.ctrl || step.meta;
  if (ctrlOrMeta && (reservedWithMod.has(step.key) || /^[1-9]$/.test(step.key))) return true;
  if (/^f([1-9]|1[0-2])$/.test(step.key)) return true;
  return step.alt && !ctrlOrMeta && (step.key === "left" || step.key === "right" || step.key === "home");
}

export function isReserved(keys: string): boolean {
  return parseBinding(keys).some(isReservedStep);
}

export interface GuardElement {
  tagName: string;
  type?: string;
  isContentEditable?: boolean;
  parentElement?: GuardElement | null;
  getAttribute(name: string): string | null;
}

export interface GuardEvent {
  isComposing: boolean;
  repeat: boolean;
  defaultPrevented: boolean;
  target: GuardElement | null;
}

const nonTextInputs = new Set([
  "checkbox",
  "radio",
  "button",
  "submit",
  "reset",
  "image",
  "file",
  "range",
  "color",
  "hidden",
]);
const typingRoles = new Set(["textbox", "combobox", "searchbox"]);

function isTypingNode(el: GuardElement): boolean {
  const tag = el.tagName.toLowerCase();
  if (tag === "textarea" || tag === "select") return true;
  if (tag === "input") return !nonTextInputs.has((el.type ?? el.getAttribute("type") ?? "text").toLowerCase());
  if (el.isContentEditable) return true;
  const role = el.getAttribute("role");
  if (role !== null && typingRoles.has(role)) return true;
  return el.getAttribute("data-shortcuts") === "off";
}

/** True for a field where the person types, or anything inside one. */
export function isTypingTarget(el: GuardElement | null): boolean {
  for (let node = el; node; node = node.parentElement ?? null) {
    if (isTypingNode(node)) return true;
  }
  return false;
}

export interface GuardEnv {
  /** A modal dialog other than the palette and the help is open. */
  blockingDialogOpen: boolean;
}

export function shouldIgnoreEvent(event: GuardEvent, env: GuardEnv): boolean {
  return (
    event.isComposing ||
    event.repeat ||
    event.defaultPrevented ||
    env.blockingDialogOpen ||
    isTypingTarget(event.target)
  );
}

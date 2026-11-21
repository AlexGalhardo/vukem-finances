export interface Step {
  key: string;
  mod: boolean;
  ctrl: boolean;
  meta: boolean;
  alt: boolean;
  shift: boolean;
}

export interface KeyLike {
  key: string;
  ctrlKey: boolean;
  metaKey: boolean;
  altKey: boolean;
  shiftKey: boolean;
}

const aliases: Record<string, string> = {
  arrowleft: "left",
  arrowright: "right",
  arrowup: "up",
  arrowdown: "down",
  esc: "escape",
  " ": "space",
  spacebar: "space",
};

const modifierKeys = new Set(["shift", "control", "alt", "meta", "capslock", "altgraph", "os"]);

export function normalizeKey(key: string): string {
  const lower = key.toLowerCase();
  return aliases[lower] ?? lower;
}

export function isModifierKey(key: string): boolean {
  return modifierKeys.has(key.toLowerCase());
}

/** Parses "mod+k" or "shift+t" into one step. */
export function parseStep(token: string): Step {
  const parts = token.split("+");
  const key = normalizeKey(parts[parts.length - 1] ?? "");
  const step: Step = { key, mod: false, ctrl: false, meta: false, alt: false, shift: false };
  if (key === "") throw new Error(`shortcut: empty key in "${token}"`);
  for (const part of parts.slice(0, -1)) {
    const name = part.toLowerCase();
    if (name === "mod") step.mod = true;
    else if (name === "ctrl") step.ctrl = true;
    else if (name === "meta" || name === "cmd") step.meta = true;
    else if (name === "alt") step.alt = true;
    else if (name === "shift") step.shift = true;
    else throw new Error(`shortcut: unknown modifier "${part}" in "${token}"`);
  }
  return step;
}

/** Parses "g d" (sequence), "?" or "mod+k". */
export function parseBinding(keys: string): Step[] {
  const tokens = keys.trim().split(/\s+/);
  if (tokens.length === 0 || tokens[0] === "") throw new Error("shortcut: empty binding");
  return tokens.map(parseStep);
}

export function hasModifier(steps: readonly Step[]): boolean {
  return steps.some((s) => s.mod || s.ctrl || s.meta || s.alt);
}

export function formatStep(step: Step): string {
  const parts: string[] = [];
  if (step.mod) parts.push("mod");
  if (step.ctrl) parts.push("ctrl");
  if (step.meta) parts.push("meta");
  if (step.alt) parts.push("alt");
  if (step.shift) parts.push("shift");
  parts.push(step.key);
  return parts.join("+");
}

/** Canonical form, so equal bindings compare equal as strings. */
export function normalizeBinding(keys: string): string {
  return parseBinding(keys).map(formatStep).join(" ");
}

function isPlainKey(key: string): boolean {
  return /^[a-z0-9]$/.test(key);
}

export function stepMatches(step: Step, event: KeyLike, isMac: boolean): boolean {
  const wantCtrl = step.ctrl || (step.mod && !isMac);
  const wantMeta = step.meta || (step.mod && isMac);
  if (event.ctrlKey !== wantCtrl || event.metaKey !== wantMeta || event.altKey !== step.alt) return false;
  if (normalizeKey(event.key) !== step.key) return false;
  if (step.shift) return event.shiftKey;
  // Symbols such as "?" need Shift on some layouts; letters, digits and chords must not carry it.
  return isPlainKey(step.key) || wantCtrl || wantMeta || step.alt ? !event.shiftKey : true;
}

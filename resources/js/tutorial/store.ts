import { initialState } from "./state";
import type { TutorialState } from "./types";

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

/** One key per person and device. It holds screen ids, a step index and flags, never amounts, names or page data. */
export const storageKey = (userId: number | string): string => `vf.tutorial.v1.${userId}`;

// When storage is blocked the tutorial still runs for the session.
const memory = new Map<string, string>();

export function browserStorage(): StorageLike | null {
  try {
    return typeof localStorage === "undefined" ? null : localStorage;
  } catch {
    return null;
  }
}

function isFlags(value: unknown): value is Record<string, true> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value) &&
    Object.values(value).every((v) => v === true)
  );
}

/** Reads a stored value and returns a valid state; an unknown version or a broken value resets to the start. */
export function migrate(raw: unknown): TutorialState {
  if (typeof raw !== "object" || raw === null) return initialState;
  const value = raw as Partial<Record<keyof TutorialState, unknown>>;
  if (value.version !== 1) return initialState;
  const status = value.status;
  if (status !== "idle" && status !== "running" && status !== "paused" && status !== "done") return initialState;
  if (!isFlags(value.completed) || !isFlags(value.skipped)) return initialState;
  const stepIndex = typeof value.stepIndex === "number" && value.stepIndex >= 0 ? Math.floor(value.stepIndex) : 0;
  const screenId = typeof value.screenId === "string" ? value.screenId : null;
  return {
    version: 1,
    // A run that was going when the page closed comes back paused: it never opens by itself.
    status: status === "running" ? "paused" : status,
    screenId,
    stepIndex,
    completed: value.completed,
    skipped: value.skipped,
    welcomeDismissed: value.welcomeDismissed === true,
  };
}

export function loadState(storage: StorageLike | null, userId: number | string): TutorialState {
  const key = storageKey(userId);
  let text: string | null = memory.get(key) ?? null;
  try {
    text = storage?.getItem(key) ?? text;
  } catch {
    /* Fall back to memory. */
  }
  if (text === null) return initialState;
  try {
    return migrate(JSON.parse(text));
  } catch {
    return initialState;
  }
}

export function saveState(storage: StorageLike | null, userId: number | string, state: TutorialState): void {
  const key = storageKey(userId);
  const text = JSON.stringify(state);
  memory.set(key, text);
  try {
    storage?.setItem(key, text);
  } catch {
    /* Storage may be disabled: the memory copy keeps this session going. */
  }
}

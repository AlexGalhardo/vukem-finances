import { useSyncExternalStore } from "react";

export const SINGLE_KEY_STORAGE_KEY = "vf.shortcuts.single";

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

/** Single-key shortcuts are on unless the person switched them off (WCAG 2.1.4). */
export function readSingleKey(storage: StorageLike | null): boolean {
  try {
    return storage?.getItem(SINGLE_KEY_STORAGE_KEY) !== "off";
  } catch {
    return true;
  }
}

const listeners = new Set<() => void>();

function browserStorage(): StorageLike | null {
  try {
    return typeof localStorage === "undefined" ? null : localStorage;
  } catch {
    return null;
  }
}

export function singleKeyEnabled(): boolean {
  return readSingleKey(browserStorage());
}

export function setSingleKeyEnabled(enabled: boolean): void {
  try {
    browserStorage()?.setItem(SINGLE_KEY_STORAGE_KEY, enabled ? "on" : "off");
  } catch {
    /* Storage may be disabled: the choice then lasts until reload. */
  }
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useSingleKeyShortcuts(): [boolean, (enabled: boolean) => void] {
  const enabled = useSyncExternalStore(subscribe, singleKeyEnabled, () => true);
  return [enabled, setSingleKeyEnabled];
}

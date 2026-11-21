import type { Key } from "../i18n/keys";
import { normalizeBinding, parseBinding, type Step } from "./keys";
import type { Binding } from "./matcher";
import { isReservedStep } from "./reserved";

export type ActionGroup = "navigation" | "actions";

export interface Action {
  id: string;
  labelKey: Key;
  group: ActionGroup;
  /** "g d" (sequence), "?" or "mod+k" (chord). Omit for an action that only lives in the palette. */
  keys?: string;
  run: () => void;
  enabled?: () => boolean;
}

export interface Registry {
  register(action: Action): () => void;
  list(): readonly Action[];
  bindings(): Binding[];
  subscribe(listener: () => void): () => void;
}

interface Entry {
  action: Action;
  steps: Step[];
}

function overlaps(a: readonly Step[], b: readonly Step[]): boolean {
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    const x = a[i] as Step;
    const y = b[i] as Step;
    if (
      x.key !== y.key ||
      x.mod !== y.mod ||
      x.ctrl !== y.ctrl ||
      x.meta !== y.meta ||
      x.alt !== y.alt ||
      x.shift !== y.shift
    )
      return false;
  }
  return true;
}

/**
 * `strict` makes duplicate ids and duplicate (or shadowing) key bindings throw; it is on in development and test.
 * Reserved browser combinations are refused in every environment.
 */
export function createRegistry(options: { strict: boolean }): Registry {
  const entries = new Map<string, Entry>();
  const listeners = new Set<() => void>();
  let snapshot: readonly Action[] = [];

  const publish = (): void => {
    snapshot = [...entries.values()].map((e) => e.action);
    for (const listener of listeners) listener();
  };

  return {
    register(action) {
      const steps = action.keys === undefined ? [] : parseBinding(action.keys);
      for (const step of steps) {
        if (isReservedStep(step)) {
          throw new Error(`shortcut: "${action.keys}" of "${action.id}" is reserved by the browser`);
        }
      }
      const problem = (() => {
        if (entries.has(action.id)) return `action id "${action.id}" is already registered`;
        if (steps.length === 0) return undefined;
        for (const other of entries.values()) {
          if (overlaps(steps, other.steps) && other.steps.length > 0) {
            return `"${action.keys}" of "${action.id}" collides with "${other.action.keys}" of "${other.action.id}"`;
          }
        }
        return undefined;
      })();
      if (problem !== undefined) {
        if (options.strict) throw new Error(`shortcut: ${problem}`);
        console.warn(`shortcut: ${problem}; the later registration is ignored`);
        return () => {};
      }

      const entry: Entry = { action, steps };
      entries.set(action.id, entry);
      publish();
      return () => {
        if (entries.get(action.id) === entry) {
          entries.delete(action.id);
          publish();
        }
      };
    },
    list: () => snapshot,
    bindings() {
      return [...entries.values()]
        .filter((e) => e.steps.length > 0 && (e.action.enabled?.() ?? true))
        .map((e) => ({ id: e.action.id, steps: e.steps }));
    },
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}

export const registry: Registry = createRegistry({ strict: !import.meta.env.PROD });

export function registerAction(action: Action): () => void {
  return registry.register(action);
}

export function listActions(): readonly Action[] {
  return registry.list();
}

export { normalizeBinding };

import { isModifierKey, type KeyLike, type Step, stepMatches } from "./keys";

export interface Binding {
  id: string;
  steps: readonly Step[];
}

export type MatchResult = { type: "none" } | { type: "pending" } | { type: "match"; id: string };

export interface Timers {
  set(run: () => void, ms: number): unknown;
  clear(handle: unknown): void;
}

export const defaultTimers: Timers = {
  set: (run, ms) => setTimeout(run, ms),
  clear: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
};

export interface SequenceMatcher {
  feed(event: KeyLike, bindings: readonly Binding[]): MatchResult;
  reset(): void;
}

/** Matches chords and sequences ("g d"); a sequence expires after `timeoutMs` between keys. */
export function createSequenceMatcher(options: {
  isMac: boolean;
  timeoutMs?: number;
  timers?: Timers;
}): SequenceMatcher {
  const timeoutMs = options.timeoutMs ?? 1000;
  const timers = options.timers ?? defaultTimers;
  let buffer: KeyLike[] = [];
  let handle: unknown;

  const reset = (): void => {
    buffer = [];
    if (handle !== undefined) timers.clear(handle);
    handle = undefined;
  };

  const evaluate = (sequence: KeyLike[], bindings: readonly Binding[]): MatchResult => {
    let partial = false;
    for (const binding of bindings) {
      if (binding.steps.length < sequence.length) continue;
      if (!sequence.every((e, i) => stepMatches(binding.steps[i] as Step, e, options.isMac))) continue;
      if (binding.steps.length === sequence.length) {
        reset();
        return { type: "match", id: binding.id };
      }
      partial = true;
    }
    if (!partial) return { type: "none" };
    reset();
    buffer = sequence;
    handle = timers.set(reset, timeoutMs);
    return { type: "pending" };
  };

  return {
    reset,
    feed(event, bindings) {
      if (isModifierKey(event.key)) return { type: "none" };
      const hadPending = buffer.length > 0;
      const extended = evaluate([...buffer, event], bindings);
      if (extended.type !== "none") return extended;
      reset();
      // A key that breaks a sequence may itself start another one.
      return hadPending ? evaluate([event], bindings) : extended;
    },
  };
}

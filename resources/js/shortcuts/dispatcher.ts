import { type GuardEvent, shouldIgnoreEvent } from "./guard";
import { hasModifier, type KeyLike } from "./keys";
import type { SequenceMatcher } from "./matcher";
import type { Registry } from "./registry";

export interface DispatchEvent extends KeyLike, GuardEvent {
  preventDefault(): void;
}

export interface DispatchEnv {
  blockingDialogOpen(): boolean;
  singleKeys(): boolean;
}

/** The one keydown handler: guard, match against the registry, run the action. Returns true when it ran one. */
export function createDispatcher(registry: Registry, matcher: SequenceMatcher, env: DispatchEnv) {
  return (event: DispatchEvent): boolean => {
    if (shouldIgnoreEvent(event, { blockingDialogOpen: env.blockingDialogOpen() })) {
      matcher.reset();
      return false;
    }
    const allowSingle = env.singleKeys();
    const bindings = registry.bindings().filter((b) => allowSingle || hasModifier(b.steps));
    const result = matcher.feed(event, bindings);
    if (result.type !== "match") return false;
    const action = registry.list().find((a) => a.id === result.id);
    if (!action) return false;
    event.preventDefault();
    action.run();
    return true;
  };
}

export function detectMac(): boolean {
  return typeof navigator !== "undefined" && /mac|iphone|ipad/i.test(navigator.platform);
}

import { hasModifier, parseBinding } from "./keys";
import { formatAriaKeyShortcuts, formatBindingParts } from "./format";
import type { Action } from "./registry";
import { useSingleKeyShortcuts } from "./singleKey";
import { useActions } from "./useAction";
import { useIsMac } from "./useIsMac";

export interface ActionHint {
  /** One label per key press, for example ["G", "D"] or ["Ctrl+K"]. */
  parts: string[];
  /** Value for aria-keyshortcuts; undefined for a sequence. */
  ariaKeyShortcuts?: string;
}

export interface HintContext {
  isMac: boolean;
  /** The "Single-key shortcuts" preference. */
  singleKeys: boolean;
}

/**
 * The hint of one action, or null when there is nothing true to show: the action is missing, has no keys, is
 * disabled right now, or its binding has no modifier while single-key shortcuts are off (it would not work).
 */
export function resolveActionHint(
  actions: readonly Action[],
  actionId: string,
  { isMac, singleKeys }: HintContext,
): ActionHint | null {
  const action = actions.find((candidate) => candidate.id === actionId);
  if (!action?.keys) return null;
  if (action.enabled && !action.enabled()) return null;
  if (!singleKeys && !hasModifier(parseBinding(action.keys))) return null;
  return {
    parts: formatBindingParts(action.keys, isMac),
    ariaKeyShortcuts: formatAriaKeyShortcuts(action.keys, isMac),
  };
}

/** Reads the shortcut registry, so a key is never written by hand in JSX. */
export function useActionHint(actionId: string): ActionHint | null {
  const actions = useActions();
  const isMac = useIsMac();
  const [singleKeys] = useSingleKeyShortcuts();
  return resolveActionHint(actions, actionId, { isMac, singleKeys });
}

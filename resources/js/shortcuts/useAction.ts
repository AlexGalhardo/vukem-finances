import { useEffect, useRef, useSyncExternalStore } from "react";
import { type Action, registry } from "./registry";

/** Registers the action while the component is mounted. `run` and `enabled` may change on every render. */
export function useAction(action: Action): void {
  const latest = useRef(action);
  latest.current = action;
  const { id, labelKey, group, keys } = action;
  useEffect(
    () =>
      registry.register({
        id,
        labelKey,
        group,
        keys,
        run: () => latest.current.run(),
        enabled: () => latest.current.enabled?.() ?? true,
      }),
    [id, labelKey, group, keys],
  );
}

export function useActions(): readonly Action[] {
  return useSyncExternalStore(registry.subscribe, registry.list, registry.list);
}

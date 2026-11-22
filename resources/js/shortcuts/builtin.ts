import { navActionId, navItems } from "../nav";
import type { Action } from "./registry";

export type ActionSpec = Omit<Action, "run" | "enabled">;

/** Actions of the shell itself; docs/shortcuts.md lists the same ids and keys (a test compares them). */
export const shellActionSpecs: readonly ActionSpec[] = [
  { id: "open-palette", labelKey: "openPalette", group: "actions", keys: "mod+k" },
  { id: "open-shortcut-help", labelKey: "openShortcutHelp", group: "actions", keys: "?" },
  { id: "toggle-sidebar", labelKey: "toggleSidebar", group: "actions", keys: "[" },
  { id: "open-tutorial", labelKey: "openTutorial", group: "actions", keys: "h t" },
  { id: "open-page-help", labelKey: "openPageHelp", group: "actions", keys: "h p" },
];

export function navActionSpecs(items = navItems): ActionSpec[] {
  return items.map((item) => ({
    id: navActionId(item),
    labelKey: item.labelKey,
    group: "navigation",
    keys: item.keys,
  }));
}

export function builtinActionSpecs(): ActionSpec[] {
  return [...shellActionSpecs, ...navActionSpecs()];
}

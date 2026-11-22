import { router } from "@inertiajs/react";
import { useEffect } from "react";
import { useSidebar } from "../components/ui/sidebar";
import { type NavItem, navActionId } from "../nav";
import { navActionSpecs, shellActionSpecs } from "./builtin";
import { createDispatcher, detectMac } from "./dispatcher";
import type { GuardElement } from "./guard";
import { createSequenceMatcher } from "./matcher";
import { registry } from "./registry";
import { singleKeyEnabled } from "./singleKey";
import { useAction } from "./useAction";

/** Radix dialogs other than the palette and the help (those carry `data-vf-overlay`) block the shortcuts. */
export function blockingDialogOpen(): boolean {
  return (
    document.querySelector('[role="dialog"][data-state="open"]:not([data-vf-overlay]), [role="alertdialog"]') !== null
  );
}

function useKeyboardDispatcher(): void {
  useEffect(() => {
    const dispatch = createDispatcher(registry, createSequenceMatcher({ isMac: detectMac() }), {
      blockingDialogOpen,
      singleKeys: singleKeyEnabled,
    });
    const onKeyDown = (event: KeyboardEvent): void => {
      const target: GuardElement | null = event.target instanceof HTMLElement ? event.target : null;
      dispatch({
        key: event.key,
        ctrlKey: event.ctrlKey,
        metaKey: event.metaKey,
        altKey: event.altKey,
        shiftKey: event.shiftKey,
        isComposing: event.isComposing,
        repeat: event.repeat,
        defaultPrevented: event.defaultPrevented,
        target,
        preventDefault: () => event.preventDefault(),
      });
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);
}

function NavAction({ item }: { item: NavItem }) {
  const spec = navActionSpecs([item])[0];
  if (!spec) throw new Error(`nav item ${navActionId(item)} has no action spec`);
  useAction({ ...spec, run: () => router.visit(item.href) });
  return null;
}

/** Installs the one keydown listener and registers the shell's own actions and one navigation action per item. */
export default function ShellKeyboard({
  items,
  onOpenPalette,
  onOpenHelp,
  onOpenPageHelp,
}: {
  items: readonly NavItem[];
  onOpenPalette: () => void;
  onOpenHelp: () => void;
  onOpenPageHelp: () => void;
}) {
  const { toggleSidebar } = useSidebar();
  useKeyboardDispatcher();
  const spec = (id: string) => {
    const found = shellActionSpecs.find((candidate) => candidate.id === id);
    if (!found) throw new Error(`shell action spec ${id} is missing`);
    return found;
  };
  useAction({ ...spec("open-palette"), run: onOpenPalette });
  useAction({ ...spec("open-shortcut-help"), run: onOpenHelp });
  useAction({ ...spec("toggle-sidebar"), run: toggleSidebar });
  useAction({ ...spec("open-page-help"), run: onOpenPageHelp });
  return (
    <>
      {items.map((item) => (
        <NavAction key={item.id} item={item} />
      ))}
    </>
  );
}

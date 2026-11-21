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
}: {
  items: readonly NavItem[];
  onOpenPalette: () => void;
  onOpenHelp: () => void;
}) {
  const { toggleSidebar } = useSidebar();
  useKeyboardDispatcher();
  const [palette, help, sidebar] = shellActionSpecs;
  if (!palette || !help || !sidebar) throw new Error("shell action specs are incomplete");
  useAction({ ...palette, run: onOpenPalette });
  useAction({ ...help, run: onOpenHelp });
  useAction({ ...sidebar, run: toggleSidebar });
  return (
    <>
      {items.map((item) => (
        <NavAction key={item.id} item={item} />
      ))}
    </>
  );
}

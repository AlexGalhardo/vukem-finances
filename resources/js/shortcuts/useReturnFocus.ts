import { useCallback, useLayoutEffect, useRef } from "react";

/**
 * Radix returns focus to its Trigger, and the palette and help have none (a shortcut or a header button
 * opens them). This remembers the element that had focus when `open` turned true and gives it back.
 */
export function useReturnFocus(open: boolean): () => void {
  const opener = useRef<HTMLElement | null>(null);
  // A layout effect runs before Radix moves focus into the dialog (its effect is passive).
  useLayoutEffect(() => {
    if (open) opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  }, [open]);
  return useCallback(() => {
    const element = opener.current;
    opener.current = null;
    if (element?.isConnected) element.focus();
  }, []);
}

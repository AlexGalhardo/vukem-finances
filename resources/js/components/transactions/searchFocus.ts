// The transactions page says how to focus its search field; the one `search-transactions` action (registered by
// QuickAdd, which every signed-in page mounts) calls it, or sends the person to the page when it is not showing.

let focuser: (() => void) | null = null;

/** Registers the focus function of the search field while the page shows it. Returns the cleanup. */
export function setSearchFocus(fn: () => void): () => void {
  focuser = fn;
  return () => {
    if (focuser === fn) focuser = null;
  };
}

/** Focuses the search field if the transactions page is showing it. */
export function focusTransactionSearch(): boolean {
  if (!focuser) return false;
  focuser();
  return true;
}

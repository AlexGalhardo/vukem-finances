import { startDownload } from "../lib/exports";
import { useAction } from "./useAction";

/**
 * Registers the `export-transactions` action (`x`) on the page that mounts it: it downloads the CSV of the URL the
 * page gives, through a real link. Typing `x` in a field does nothing (the shortcut guard).
 */
export function useExportAction(url: () => string): void {
  useAction({
    id: "export-transactions",
    labelKey: "exportTransactions",
    group: "actions",
    keys: "x",
    run: () => startDownload(url()),
  });
}

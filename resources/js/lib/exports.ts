/** The query string of the transaction list, as the CSV export reads it: only the filters that are set. */
export function transactionsCsvUrl(filters: Record<string, string>, options: { excel?: boolean } = {}): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) if (value !== "") query.set(key, value);
  if (options.excel) {
    query.set("bom", "1");
    query.set("delimiter", "semicolon");
  }
  const text = query.toString();
  return `/transactions/export.csv${text === "" ? "" : `?${text}`}`;
}

/** The full JSON export of the workspace; only owners and admins may take it. */
export const workspaceJsonUrl = "/workspace/export.json";

/** Owners and admins may take the JSON export (the server checks it again). */
export function canExportJson(role: string | undefined): boolean {
  return role === "owner" || role === "admin";
}

/**
 * Starts the download through a real link, so the browser handles the attachment (the page stays where it is and
 * the server's Content-Disposition names the file).
 */
export function startDownload(url: string): void {
  const link = document.createElement("a");
  link.href = url;
  link.rel = "noopener";
  link.style.display = "none";
  document.body.append(link);
  link.click();
  link.remove();
}

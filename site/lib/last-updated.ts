import type { DocPage, LastUpdatedEntry } from "./docs.ts";

/**
 * The entries of content/last-updated.json. A page whose source did not change keeps its entry; a changed or
 * new page gets the date its source was last committed, or `today` while the change is not committed yet.
 */
export function nextEntries(
  pages: DocPage[],
  previous: Record<string, LastUpdatedEntry>,
  hashOf: (page: DocPage) => string,
  dateOf: (page: DocPage) => string,
): Record<string, LastUpdatedEntry> {
  const entries: Record<string, LastUpdatedEntry> = {};
  for (const page of pages) {
    const hash = hashOf(page);
    const before = previous[page.slug];
    entries[page.slug] = before && before.hash === hash ? before : { date: dateOf(page), hash };
  }
  return entries;
}

/** Pages whose source changed after the date written for them, and entries of pages that no longer exist. */
export function staleEntries(
  pages: DocPage[],
  entries: Record<string, LastUpdatedEntry>,
  hashOf: (page: DocPage) => string,
): string[] {
  const problems: string[] = [];
  for (const page of pages) {
    const entry = entries[page.slug];
    if (!entry) problems.push(`${page.file}: no date in content/last-updated.json`);
    else if (entry.hash !== hashOf(page)) problems.push(`${page.file}: changed after its date ${entry.date}`);
  }
  const slugs = new Set(pages.map((p) => p.slug));
  for (const slug of Object.keys(entries)) {
    if (!slugs.has(slug)) problems.push(`content/last-updated.json: "${slug}" is not a page any more`);
  }
  return problems;
}

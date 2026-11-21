/**
 * Subsequence scorer: every character of the query must appear in the text, in order.
 * Returns null when it does not; higher is better. Prefix and word-start hits and runs of
 * adjacent characters score more, and a shorter text wins a tie.
 */
export function scoreSubsequence(query: string, text: string): number | null {
  const q = query.toLocaleLowerCase().replace(/\s+/g, "");
  const t = text.toLocaleLowerCase();
  if (q === "") return 0;
  let score = 0;
  let from = 0;
  let previous = -2;
  for (const char of q) {
    const index = t.indexOf(char, from);
    if (index === -1) return null;
    score += 1;
    if (index === previous + 1) score += 3;
    if (index === 0) score += 5;
    else if (/[\s\-_/]/.test(t[index - 1] ?? "")) score += 3;
    previous = index;
    from = index + 1;
  }
  if (t.startsWith(q)) score += 5;
  return score - t.length / 100;
}

export function rank<T>(query: string, items: readonly T[], label: (item: T) => string): T[] {
  if (query.trim() === "") return [...items];
  return items
    .map((item, order) => ({ item, order, score: scoreSubsequence(query, label(item)) }))
    .filter((entry): entry is { item: T; order: number; score: number } => entry.score !== null)
    .sort((a, b) => b.score - a.score || a.order - b.order)
    .map((entry) => entry.item);
}

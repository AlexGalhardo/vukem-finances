// Pure helpers of the new transaction and new transfer dialogs. The browser never parses or adds money:
// the amount goes to the server as the text the person typed, and the server answers with cents.

export interface PickerAccount {
  id: number;
  name: string;
  kind: string;
}

export interface PickerCategory {
  id: number;
  name: string;
  kind: string;
}

/** What `GET /transactions/options` answers: archived accounts are already left out. */
export interface QuickAddOptions {
  accounts: PickerAccount[];
  categories: PickerCategory[];
  today: string;
  currency: string;
  /** Names of the tags of the workspace, for the suggestions of the tag field. */
  tags: string[];
}

export type EntryKind = "expense" | "income";

export interface TransactionDraft {
  kind: EntryKind;
  amount: string;
  accountId: string;
  categoryId: string;
  description: string;
  date: string;
  /** Tag names; the server creates the ones that do not exist yet. */
  tags: string[];
}

/** A transaction takes at most this many tags. */
export const MAX_TAGS = 10;

/** Adds `name` to `tags` unless a tag with the same name, in any letter case, is there already or the limit is reached. */
export function addTag(tags: readonly string[], name: string): string[] {
  const clean = name.trim();
  if (clean === "" || tags.length >= MAX_TAGS) return [...tags];
  if (tags.some((tag) => tag.toLowerCase() === clean.toLowerCase())) return [...tags];
  return [...tags, clean];
}

/** An installment purchase: the amount of the dialog is the total, the date is the first part. */
export interface InstallmentDraft {
  accountId: string;
  categoryId: string;
  total: string;
  parts: string;
  firstDate: string;
  description: string;
  tags: string[];
}

export interface TransferDraft {
  amount: string;
  fromAccountId: string;
  toAccountId: string;
  description: string;
  date: string;
}

export function emptyTransactionDraft(kind: EntryKind, options: QuickAddOptions): TransactionDraft {
  return {
    kind,
    amount: "",
    accountId: options.accounts[0] ? String(options.accounts[0].id) : "",
    categoryId: "",
    description: "",
    date: options.today,
    tags: [],
  };
}

/** Source defaults to the first account and destination to the next one, so a transfer is amount plus Enter. */
export function emptyTransferDraft(options: QuickAddOptions): TransferDraft {
  return {
    amount: "",
    fromAccountId: options.accounts[0] ? String(options.accounts[0].id) : "",
    toAccountId: options.accounts[1] ? String(options.accounts[1].id) : "",
    description: "",
    date: options.today,
  };
}

/** Cents as the "12.50" an edit field shows, with integer arithmetic only. */
export function centsToInput(cents: number): string {
  const sign = cents < 0 ? "-" : "";
  const abs = Math.abs(cents);
  const whole = Math.trunc(abs / 100);
  const fraction = abs - whole * 100;
  return `${sign}${whole}.${String(fraction).padStart(2, "0")}`;
}

/** Categories that fit the kind of the entry: an expense takes expense categories only. */
export function categoriesFor(options: QuickAddOptions, kind: EntryKind): PickerCategory[] {
  return options.categories.filter((category) => category.kind === kind);
}

/** Case-insensitive "contains" filter of a picker. Typing the label of the chosen item shows every item. */
export function filterPicker<T extends { name: string }>(items: readonly T[], query: string, selected?: T): T[] {
  const needle = query.trim().toLowerCase();
  if (needle === "" || (selected && selected.name.toLowerCase() === needle)) return [...items];
  return items.filter((item) => item.name.toLowerCase().includes(needle));
}

export type SubmitResult = { ok: true } | { ok: false; message: string; fields: Record<string, string> };

interface ProblemBody {
  detail?: unknown;
  errors?: unknown;
}

/** Reads an RFC 9457 problem answer: its `detail` and, for a validation failure, the first message per field. */
export function readProblem(body: unknown, fallback: string): { message: string; fields: Record<string, string> } {
  const problem: ProblemBody = typeof body === "object" && body !== null ? body : {};
  const fields: Record<string, string> = {};
  if (typeof problem.errors === "object" && problem.errors !== null) {
    for (const [name, messages] of Object.entries(problem.errors)) {
      const first: unknown = Array.isArray(messages) ? messages[0] : undefined;
      if (typeof first === "string") fields[name] = first;
    }
  }
  const message =
    Object.values(fields)[0] ??
    (typeof problem.detail === "string" && problem.detail !== "" ? problem.detail : fallback);
  return { message, fields };
}

/** Sends one dialog as JSON. A 2xx is success; anything else becomes a message the dialog can show. */
export async function sendJSON(
  url: string,
  method: "POST" | "PATCH",
  body: TransactionDraft | TransferDraft | InstallmentDraft,
  fallback: string,
): Promise<SubmitResult> {
  let response: Response;
  try {
    response = await fetch(url, {
      method,
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return { ok: false, message: fallback, fields: {} };
  }
  if (response.ok) return { ok: true };
  const parsed: unknown = await response.json().catch(() => null);
  return { ok: false, ...readProblem(parsed, fallback) };
}

export async function loadOptions(signal?: AbortSignal): Promise<QuickAddOptions | null> {
  try {
    const response = await fetch("/transactions/options", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
      signal,
    });
    if (!response.ok) return null;
    return (await response.json()) as QuickAddOptions;
  } catch {
    return null;
  }
}

import type { Database } from "./database";

export interface TransactionFilter {
  /** Matches the description, the category name or the account name. */
  search?: string;
  account_uid?: string;
  category_uid?: string;
  tag_uid?: string;
  from?: string;
  to?: string;
}

export interface TransactionListItem {
  uid: string;
  occurred_on: string;
  kind: string;
  amount_cents: number;
  description: string;
  account_uid: string;
  account_name: string;
  category_uid: string | null;
  category_name: string | null;
  transfer_uid: string | null;
  tag_uids: string[];
}

/** Escapes LIKE wildcards so a search for "50%" finds "50%" and nothing else. */
function likePattern(text: string): string {
  return `%${text.replace(/[\\%_]/gu, (c) => `\\${c}`)}%`;
}

/** The newest transactions first (date, then uid), one page at a time, live rows only. */
export async function listTransactions(
  db: Database,
  filter: TransactionFilter,
  limit: number,
  offset: number,
): Promise<TransactionListItem[]> {
  const where: string[] = ["t.deleted_at IS NULL"];
  const params: (string | number)[] = [];

  const search = filter.search?.trim() ?? "";
  if (search !== "") {
    const p = likePattern(search);
    where.push("(t.description LIKE ? ESCAPE '\\' OR c.name LIKE ? ESCAPE '\\' OR a.name LIKE ? ESCAPE '\\')");
    params.push(p, p, p);
  }
  if (filter.account_uid !== undefined) {
    where.push("t.account_uid = ?");
    params.push(filter.account_uid);
  }
  if (filter.category_uid !== undefined) {
    where.push("t.category_uid = ?");
    params.push(filter.category_uid);
  }
  if (filter.tag_uid !== undefined) {
    where.push("EXISTS (SELECT 1 FROM transaction_tags tt WHERE tt.transaction_uid = t.uid AND tt.tag_uid = ?)");
    params.push(filter.tag_uid);
  }
  if (filter.from !== undefined) {
    where.push("t.occurred_on >= ?");
    params.push(filter.from);
  }
  if (filter.to !== undefined) {
    where.push("t.occurred_on <= ?");
    params.push(filter.to);
  }

  const rows = await db.all<Omit<TransactionListItem, "tag_uids"> & { tag_uids: string }>(
    `SELECT t.uid, t.occurred_on, t.kind, t.amount_cents, t.description, t.account_uid, a.name AS account_name,
		        t.category_uid, c.name AS category_name, t.transfer_uid, t.tag_uids
		 FROM transactions t
		 LEFT JOIN accounts a ON a.uid = t.account_uid
		 LEFT JOIN categories c ON c.uid = t.category_uid
		 WHERE ${where.join(" AND ")}
		 ORDER BY t.occurred_on DESC, t.uid DESC
		 LIMIT ? OFFSET ?`,
    [...params, limit, offset],
  );
  return rows.map((r) => ({ ...r, account_name: r.account_name ?? "", tag_uids: JSON.parse(r.tag_uids) as string[] }));
}

/** Categories of a kind, the most recently used first, then the ones never used, by name. Archived ones are left out. */
export async function recentCategories(
  db: Database,
  kind: string,
  limit: number,
): Promise<{ uid: string; name: string }[]> {
  return db.all<{ uid: string; name: string }>(
    `SELECT c.uid, c.name FROM categories c
		 LEFT JOIN (SELECT category_uid, MAX(occurred_on || t_uid) AS last_used FROM
		            (SELECT category_uid, occurred_on, uid AS t_uid FROM transactions WHERE deleted_at IS NULL)
		            GROUP BY category_uid) u ON u.category_uid = c.uid
		 WHERE c.deleted_at IS NULL AND c.archived_at IS NULL AND c.kind = ?
		 ORDER BY u.last_used IS NULL, u.last_used DESC, c.name
		 LIMIT ?`,
    [kind, limit],
  );
}

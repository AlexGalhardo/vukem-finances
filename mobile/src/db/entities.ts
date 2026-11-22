/** The synced entities, with the wire field names of the protocol (references travel as `<name>_uid`). */

export interface AccountFields {
  name: string;
  kind: string;
  opening_cents: number;
  archived_at: string | null;
  position: number;
}

export interface CategoryFields {
  name: string;
  kind: string;
  parent_uid: string | null;
  color: string;
  icon: string;
  archived_at: string | null;
}

export interface TagFields {
  name: string;
  color: string;
}

export interface TransactionFields {
  account_uid: string;
  category_uid: string | null;
  kind: string;
  amount_cents: number;
  description: string;
  occurred_on: string;
  transfer_uid: string | null;
  tag_uids: string[];
  statement_ref: string | null;
  recurrence_uid: string | null;
  occurrence_on: string | null;
  installment_group_uid: string | null;
  installment_no: number | null;
  installment_total: number | null;
}

export interface BudgetFields {
  /** null is the overall budget of the workspace. */
  category_uid: string | null;
  limit_cents: number;
  rollover: boolean;
}

export interface CreditCardFields {
  account_uid: string;
  limit_cents: number;
  closing_day: number;
  due_day: number;
  payment_account_uid: string | null;
  brand: string;
  last4: string;
}

export interface RecurrenceFields {
  account_uid: string;
  category_uid: string | null;
  kind: string;
  amount_cents: number;
  description: string;
  frequency: string;
  interval: number;
  start_date: string;
  end_date: string | null;
  max_count: number | null;
  generated_through: string | null;
  active: boolean;
}

export interface EntityFields {
  account: AccountFields;
  category: CategoryFields;
  tag: TagFields;
  transaction: TransactionFields;
  budget: BudgetFields;
  credit_card: CreditCardFields;
  recurrence: RecurrenceFields;
}

export type EntityName = keyof EntityFields;

export interface SyncColumns {
  uid: string;
  version: number;
  server_seq: number | null;
  deleted_at: string | null;
  dirty: boolean;
  updated_local_at: string;
}

export type Row<E extends EntityName> = EntityFields[E] & SyncColumns;

type FieldKind = "text" | "int" | "bool" | "json";

interface EntityMeta {
  table: string;
  fields: Record<string, FieldKind>;
  /** Fields that hold the uid of another entity: [field, parent entity]. */
  refs: readonly (readonly [string, EntityName])[];
  /** ORDER BY of a list: what a person expects to see first. */
  order: string;
}

export const ENTITIES: Record<EntityName, EntityMeta> = {
  account: {
    table: "accounts",
    order: "position, name, uid",
    fields: { name: "text", kind: "text", opening_cents: "int", archived_at: "text", position: "int" },
    refs: [],
  },
  category: {
    table: "categories",
    order: "kind, name, uid",
    fields: { name: "text", kind: "text", parent_uid: "text", color: "text", icon: "text", archived_at: "text" },
    refs: [["parent_uid", "category"]],
  },
  tag: { table: "tags", order: "name, uid", fields: { name: "text", color: "text" }, refs: [] },
  transaction: {
    table: "transactions",
    order: "occurred_on DESC, uid DESC",
    fields: {
      account_uid: "text",
      category_uid: "text",
      kind: "text",
      amount_cents: "int",
      description: "text",
      occurred_on: "text",
      transfer_uid: "text",
      tag_uids: "json",
      statement_ref: "text",
      recurrence_uid: "text",
      occurrence_on: "text",
      installment_group_uid: "text",
      installment_no: "int",
      installment_total: "int",
    },
    refs: [
      ["account_uid", "account"],
      ["category_uid", "category"],
    ],
  },
  budget: {
    table: "budgets",
    order: "uid",
    fields: { category_uid: "text", limit_cents: "int", rollover: "bool" },
    refs: [["category_uid", "category"]],
  },
  credit_card: {
    table: "credit_cards",
    order: "uid",
    fields: {
      account_uid: "text",
      limit_cents: "int",
      closing_day: "int",
      due_day: "int",
      payment_account_uid: "text",
      brand: "text",
      last4: "text",
    },
    refs: [
      ["account_uid", "account"],
      ["payment_account_uid", "account"],
    ],
  },
  recurrence: {
    table: "recurrences",
    order: "start_date, uid",
    fields: {
      account_uid: "text",
      category_uid: "text",
      kind: "text",
      amount_cents: "int",
      description: "text",
      frequency: "text",
      interval: "int",
      start_date: "text",
      end_date: "text",
      max_count: "int",
      generated_through: "text",
      active: "bool",
    },
    refs: [
      ["account_uid", "account"],
      ["category_uid", "category"],
    ],
  },
};

export type { FieldKind };

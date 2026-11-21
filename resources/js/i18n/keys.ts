import type accounts from "./en/accounts";
import type ai from "./en/ai";
import type common from "./en/common";
import type components from "./en/components";
import type shell from "./en/shell";
import type transactions from "./en/transactions";

/** Every translation key. `en` is the typed source of truth; add one `typeof` line per new domain file. */
export type Key =
  | keyof typeof accounts
  | keyof typeof ai
  | keyof typeof common
  | keyof typeof components
  | keyof typeof shell
  | keyof typeof transactions;

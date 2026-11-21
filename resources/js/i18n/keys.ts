import type accounts from "./en/accounts";
import type ai from "./en/ai";
import type billing from "./en/billing";
import type cards from "./en/cards";
import type categories from "./en/categories";
import type common from "./en/common";
import type components from "./en/components";
import type importPage from "./en/import";
import type search from "./en/search";
import type shell from "./en/shell";
import type tokens from "./en/tokens";
import type transactions from "./en/transactions";

/** Every translation key. `en` is the typed source of truth; add one `typeof` line per new domain file. */
export type Key =
  | keyof typeof accounts
  | keyof typeof ai
  | keyof typeof billing
  | keyof typeof cards
  | keyof typeof categories
  | keyof typeof common
  | keyof typeof components
  | keyof typeof importPage
  | keyof typeof search
  | keyof typeof shell
  | keyof typeof tokens
  | keyof typeof transactions;

import type accounts from "./en/accounts";
import type ai from "./en/ai";
import type alerts from "./en/alerts";
import type auth from "./en/auth";
import type billing from "./en/billing";
import type budgets from "./en/budgets";
import type cards from "./en/cards";
import type categories from "./en/categories";
import type common from "./en/common";
import type components from "./en/components";
import type goals from "./en/goals";
import type help from "./en/help";
import type importPage from "./en/import";
import type mcp from "./en/mcp";
import type privacy from "./en/privacy";
import type recurrences from "./en/recurrences";
import type reports from "./en/reports";
import type search from "./en/search";
import type shell from "./en/shell";
import type syncPage from "./en/sync";
import type telegram from "./en/telegram";
import type tokens from "./en/tokens";
import type tutorial from "./en/tutorial";
import type twofactor from "./en/twofactor";
import type transactions from "./en/transactions";
import type workspace from "./en/workspace";

/** Every translation key. `en` is the typed source of truth; add one `typeof` line per new domain file. */
export type Key =
  | keyof typeof accounts
  | keyof typeof ai
  | keyof typeof alerts
  | keyof typeof auth
  | keyof typeof billing
  | keyof typeof budgets
  | keyof typeof cards
  | keyof typeof categories
  | keyof typeof common
  | keyof typeof components
  | keyof typeof goals
  | keyof typeof help
  | keyof typeof importPage
  | keyof typeof mcp
  | keyof typeof privacy
  | keyof typeof recurrences
  | keyof typeof reports
  | keyof typeof search
  | keyof typeof shell
  | keyof typeof syncPage
  | keyof typeof telegram
  | keyof typeof tokens
  | keyof typeof tutorial
  | keyof typeof twofactor
  | keyof typeof transactions
  | keyof typeof workspace;

---
title: "MCP: tools"
description: "Every tool of the vukem-finances MCP server with its operation, kind, scope, toolsets and input."
order: 0
---

> Generated from `docs/reference/mcp.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

52 tools. A tool name is the operation name with an underscore for each dot. The `MCP_TOOLSETS` setting decides which tools a server offers; [MCP over stdio](/docs/integrate/mcp-stdio/) explains how to set it.

## Toolsets

| Toolset | Offers |
| --- | --- |
| `read` | every read operation |
| `transactions` | transactions and transfers: list, show, record, change |
| `planning` | budgets, the summary, goals, recurring rules, cards and statements |
| `all` | every tool |

Serialized size of the tool list of each toolset, which a client may hold in context:

| Toolset | Tools | Bytes |
| --- | --- | --- |
| `read` | 21 | 38841 |
| `read + destructive` | 21 | 38841 |
| `transactions` | 5 | 10785 |
| `transactions + destructive` | 7 | 11954 |
| `planning` | 19 | 41253 |
| `planning + destructive` | 25 | 48022 |
| `all` | 41 | 76911 |
| `all + destructive` | 52 | 87761 |

## `accounts_archive`

Archive an account.

- Operation: `accounts.archive`, destructive, asks you to confirm every call, scope `accounts:write`.
- Toolsets: all.
- Input: `uid` (string, required).

## `accounts_create`

Create an account.

- Operation: `accounts.create`, write, scope `accounts:write`.
- Toolsets: all.
- Input: `uid` (string), `name` (string, required), `kind` (string, required), `opening_cents` (int64).

## `accounts_list`

List the accounts with their balances.

- Operation: `accounts.list`, read, scope `accounts:read`.
- Toolsets: read, all.
- Input: none.

## `accounts_show`

Read an account.

- Operation: `accounts.show`, read, scope `accounts:read`.
- Toolsets: read, all.
- Input: `uid` (string, required).

## `accounts_update`

Change the name, kind and opening balance of an account.

- Operation: `accounts.update`, write, scope `accounts:write`.
- Toolsets: all.
- Input: `uid` (string, required), `name` (string?), `kind` (string?), `opening_cents` (int64?).

## `budgets_delete`

Remove a budget.

- Operation: `budgets.delete`, destructive, asks you to confirm every call, scope `budgets:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `budgets_list`

List the budgets with what the month spent.

- Operation: `budgets.list`, read, scope `budgets:read`.
- Toolsets: read, planning, all.
- Input: `month` (string).

## `budgets_set`

Set the monthly limit of an expense category.

- Operation: `budgets.set`, write, scope `budgets:write`.
- Toolsets: planning, all.
- Input: `category_uid` (string, required), `limit_cents` (int64).

## `cards_archive`

Archive a credit card.

- Operation: `cards.archive`, destructive, asks you to confirm every call, scope `cards:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `cards_create`

Create a credit card with its account.

- Operation: `cards.create`, write, scope `cards:write`.
- Toolsets: planning, all.
- Input: `name` (string, required), `limit_cents` (int64), `closing_day` (int), `due_day` (int), `payment_account_uid` (string), `brand` (string), `last4` (string).

## `cards_list`

List the credit cards with their limits and statements.

- Operation: `cards.list`, read, scope `cards:read`.
- Toolsets: read, planning, all.
- Input: none.

## `cards_show`

Read a credit card.

- Operation: `cards.show`, read, scope `cards:read`.
- Toolsets: read, planning, all.
- Input: `uid` (string, required).

## `cards_unarchive`

Bring an archived credit card back.

- Operation: `cards.unarchive`, write, scope `cards:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `cards_update`

Change a credit card.

- Operation: `cards.update`, write, scope `cards:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `name` (string?), `limit_cents` (int64?), `closing_day` (int?), `due_day` (int?), `payment_account_uid` (string?), `brand` (string?), `last4` (string?).

## `categories_archive`

Archive a category.

- Operation: `categories.archive`, destructive, asks you to confirm every call, scope `categories:write`.
- Toolsets: all.
- Input: `uid` (string, required).

## `categories_create`

Create a category.

- Operation: `categories.create`, write, scope `categories:write`.
- Toolsets: all.
- Input: `uid` (string), `name` (string, required), `kind` (string), `parent_uid` (string), `color` (string), `icon` (string).

## `categories_list`

List the categories.

- Operation: `categories.list`, read, scope `categories:read`.
- Toolsets: read, all.
- Input: none.

## `categories_show`

Read a category.

- Operation: `categories.show`, read, scope `categories:read`.
- Toolsets: read, all.
- Input: `uid` (string, required).

## `categories_update`

Change a category.

- Operation: `categories.update`, write, scope `categories:write`.
- Toolsets: all.
- Input: `uid` (string, required), `name` (string?), `parent_uid` (string?), `color` (string?), `icon` (string?).

## `goals_archive`

Archive a goal; its contributions stay.

- Operation: `goals.archive`, destructive, asks you to confirm every call, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `goals_contributions_add`

Put an amount towards a goal.

- Operation: `goals.contributions.add`, write, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `amount_cents` (int64), `date` (string), `note` (string).

## `goals_contributions_delete`

Delete a contribution of a goal.

- Operation: `goals.contributions.delete`, destructive, asks you to confirm every call, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `contribution_uid` (string, required).

## `goals_create`

Create a savings goal.

- Operation: `goals.create`, write, scope `goals:write`.
- Toolsets: planning, all.
- Input: `name` (string, required), `target_cents` (int64), `target_date` (string).

## `goals_delete`

Delete a goal and its contributions.

- Operation: `goals.delete`, destructive, asks you to confirm every call, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `goals_list`

List the savings goals with their progress.

- Operation: `goals.list`, read, scope `goals:read`.
- Toolsets: read, planning, all.
- Input: none.

## `goals_unarchive`

Bring an archived goal back.

- Operation: `goals.unarchive`, write, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required).

## `goals_update`

Change the name, target and date of a goal.

- Operation: `goals.update`, write, scope `goals:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `name` (string?), `target_cents` (int64?), `target_date` (string?).

## `recurrences_create`

Create a recurring rule and post what is already due.

- Operation: `recurrences.create`, write, scope `recurrences:write`.
- Toolsets: planning, all.
- Input: `account_uid` (string, required), `category_uid` (string), `kind` (string, required), `amount_cents` (int64), `description` (string), `frequency` (string, required), `interval` (int), `start_date` (string, required), `end_date` (string), `max_count` (int), `backfill` (bool).

## `recurrences_delete`

Delete a recurring rule and keep what it generated.

- Operation: `recurrences.delete`, destructive, asks you to confirm every call, scope `recurrences:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `future` (bool).

## `recurrences_list`

List the recurring rules.

- Operation: `recurrences.list`, read, scope `recurrences:read`.
- Toolsets: read, planning, all.
- Input: none.

## `recurrences_update`

Change a recurring rule.

- Operation: `recurrences.update`, write, scope `recurrences:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `account_uid` (string?), `category_uid` (string?), `kind` (string?), `amount_cents` (int64?), `description` (string?), `frequency` (string?), `interval` (int?), `start_date` (string?), `end_date` (string?), `max_count` (int?), `active` (bool?).

## `reports_accounts`

Income and expense of a period by account.

- Operation: `reports.accounts`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string).

## `reports_cashflow`

Income against expense per month of a period.

- Operation: `reports.cashflow`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string).

## `reports_categories`

Spending or income of a period by category.

- Operation: `reports.categories`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string), `kind` (string).

## `reports_compare`

A period against the one right before it.

- Operation: `reports.compare`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string).

## `reports_networth`

The net worth at the end of each month of a period.

- Operation: `reports.networth`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string).

## `reports_tags`

Income and expense of a period by tag.

- Operation: `reports.tags`, read, scope `reports:read`.
- Toolsets: read, all.
- Input: `period` (string), `from` (string), `to` (string).

## `statements_list`

List the statements of a credit card, newest first.

- Operation: `statements.list`, read, scope `statements:read`.
- Toolsets: read, planning, all.
- Input: `uid` (string, required).

## `statements_pay`

Pay a statement from a bank account.

- Operation: `statements.pay`, write, scope `statements:write`.
- Toolsets: planning, all.
- Input: `uid` (string, required), `ref` (string, required), `from_account_uid` (string, required), `amount_cents` (int64), `date` (string).

## `statements_show`

Read a statement of a credit card with its rows.

- Operation: `statements.show`, read, scope `statements:read`.
- Toolsets: read, planning, all.
- Input: `uid` (string, required), `ref` (string, required).

## `summary_show`

Income, expense, net, account balances and net worth of a month.

- Operation: `summary.show`, read, scope `summary:read`.
- Toolsets: read, planning, all.
- Input: `month` (string).

## `tags_create`

Create a tag.

- Operation: `tags.create`, write, scope `tags:write`.
- Toolsets: all.
- Input: `uid` (string), `name` (string, required), `color` (string).

## `tags_delete`

Delete a tag and take it off its transactions.

- Operation: `tags.delete`, destructive, asks you to confirm every call, scope `tags:write`.
- Toolsets: all.
- Input: `uid` (string, required).

## `tags_list`

List the tags.

- Operation: `tags.list`, read, scope `tags:read`.
- Toolsets: read, all.
- Input: none.

## `tags_update`

Rename or recolor a tag.

- Operation: `tags.update`, write, scope `tags:write`.
- Toolsets: all.
- Input: `uid` (string, required), `name` (string?), `color` (string?).

## `transactions_create`

Record a transaction.

- Operation: `transactions.create`, write, scope `transactions:write`.
- Toolsets: transactions, all.
- Input: `uid` (string), `account_uid` (string, required), `category_uid` (string), `kind` (string, required), `amount_cents` (int64), `date` (string, required), `description` (string), `tags` ([]string).

## `transactions_delete`

Delete a transaction.

- Operation: `transactions.delete`, destructive, asks you to confirm every call, scope `transactions:write`.
- Toolsets: transactions, all.
- Input: `uid` (string, required).

## `transactions_list`

List transactions, newest first, with filters.

- Operation: `transactions.list`, read, scope `transactions:read`.
- Toolsets: read, transactions, all.
- Input: `from` (string), `to` (string), `account` (string), `category` (string), `tag` (string), `kind` (string), `q` (string), `min_cents` (int64?), `max_cents` (int64?), `page` (int).

## `transactions_show`

Read a transaction.

- Operation: `transactions.show`, read, scope `transactions:read`.
- Toolsets: read, transactions, all.
- Input: `uid` (string, required).

## `transactions_update`

Change a transaction.

- Operation: `transactions.update`, write, scope `transactions:write`.
- Toolsets: transactions, all.
- Input: `uid` (string, required), `account_uid` (string?), `category_uid` (string?), `kind` (string?), `amount_cents` (int64?), `date` (string?), `description` (string?), `tags` ([]string?).

## `transfers_create`

Move money between two accounts.

- Operation: `transfers.create`, write, scope `transfers:write`.
- Toolsets: transactions, all.
- Input: `from_account_uid` (string, required), `to_account_uid` (string, required), `amount_cents` (int64), `date` (string, required), `description` (string).

## `transfers_delete`

Delete a transfer and both of its legs.

- Operation: `transfers.delete`, destructive, asks you to confirm every call, scope `transfers:write`.
- Toolsets: transactions, all.
- Input: `uid` (string, required).

## Operations with no tool

Each one has a written reason.

- `export.transactions`: it streams a file, and a tool answers with JSON that fits a model context; the reads (transactions.list, reports.*) cover it
- `export.workspace`: it streams a file, and a tool answers with JSON that fits a model context; the reads (transactions.list, reports.*) cover it

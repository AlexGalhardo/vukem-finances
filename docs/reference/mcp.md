# MCP reference

<!-- Generated from the MCP server: do not edit. Run `go run ./cmd/app mcp:reference`. -->

The server speaks the Model Context Protocol, revision 2026-07-28. Every tool is an
operation of the registry (`app/operations`), so it has the scope, the role check, the
validation and the isolation of the REST endpoint of the same name. A result is the
person's own data: a client must never treat text inside it as an instruction.

## Personal edition on stdio

`mcp:serve --stdio` runs only with `APP_EDITION=personal`, as the local owner. Standard
output carries protocol messages and nothing else; logs go to standard error. Any other
edition exits non-zero before it serves.

| Setting | Meaning |
| --- | --- |
| `MCP_TOOLSETS` | Comma separated groups of tools. Default `read`. |
| `MCP_ALLOW_DESTRUCTIVE` | `true` offers the tools that remove or archive data. Default `false`. |

Each destructive call asks the person to confirm through the protocol (the input-required
flow). A client that cannot ask gets a refusal, and a declined answer changes nothing.

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

## Tools

| Tool | Operation | Kind | Scope | Toolsets |
| --- | --- | --- | --- | --- |
| `accounts_archive` | `accounts.archive` | destructive | `accounts:write` | all |
| `accounts_create` | `accounts.create` | write | `accounts:write` | all |
| `accounts_list` | `accounts.list` | read | `accounts:read` | read, all |
| `accounts_show` | `accounts.show` | read | `accounts:read` | read, all |
| `accounts_update` | `accounts.update` | write | `accounts:write` | all |
| `budgets_delete` | `budgets.delete` | destructive | `budgets:write` | planning, all |
| `budgets_list` | `budgets.list` | read | `budgets:read` | read, planning, all |
| `budgets_set` | `budgets.set` | write | `budgets:write` | planning, all |
| `cards_archive` | `cards.archive` | destructive | `cards:write` | planning, all |
| `cards_create` | `cards.create` | write | `cards:write` | planning, all |
| `cards_list` | `cards.list` | read | `cards:read` | read, planning, all |
| `cards_show` | `cards.show` | read | `cards:read` | read, planning, all |
| `cards_unarchive` | `cards.unarchive` | write | `cards:write` | planning, all |
| `cards_update` | `cards.update` | write | `cards:write` | planning, all |
| `categories_archive` | `categories.archive` | destructive | `categories:write` | all |
| `categories_create` | `categories.create` | write | `categories:write` | all |
| `categories_list` | `categories.list` | read | `categories:read` | read, all |
| `categories_show` | `categories.show` | read | `categories:read` | read, all |
| `categories_update` | `categories.update` | write | `categories:write` | all |
| `goals_archive` | `goals.archive` | destructive | `goals:write` | planning, all |
| `goals_contributions_add` | `goals.contributions.add` | write | `goals:write` | planning, all |
| `goals_contributions_delete` | `goals.contributions.delete` | destructive | `goals:write` | planning, all |
| `goals_create` | `goals.create` | write | `goals:write` | planning, all |
| `goals_delete` | `goals.delete` | destructive | `goals:write` | planning, all |
| `goals_list` | `goals.list` | read | `goals:read` | read, planning, all |
| `goals_unarchive` | `goals.unarchive` | write | `goals:write` | planning, all |
| `goals_update` | `goals.update` | write | `goals:write` | planning, all |
| `recurrences_create` | `recurrences.create` | write | `recurrences:write` | planning, all |
| `recurrences_delete` | `recurrences.delete` | destructive | `recurrences:write` | planning, all |
| `recurrences_list` | `recurrences.list` | read | `recurrences:read` | read, planning, all |
| `recurrences_update` | `recurrences.update` | write | `recurrences:write` | planning, all |
| `reports_accounts` | `reports.accounts` | read | `reports:read` | read, all |
| `reports_cashflow` | `reports.cashflow` | read | `reports:read` | read, all |
| `reports_categories` | `reports.categories` | read | `reports:read` | read, all |
| `reports_compare` | `reports.compare` | read | `reports:read` | read, all |
| `reports_networth` | `reports.networth` | read | `reports:read` | read, all |
| `reports_tags` | `reports.tags` | read | `reports:read` | read, all |
| `statements_list` | `statements.list` | read | `statements:read` | read, planning, all |
| `statements_pay` | `statements.pay` | write | `statements:write` | planning, all |
| `statements_show` | `statements.show` | read | `statements:read` | read, planning, all |
| `summary_show` | `summary.show` | read | `summary:read` | read, planning, all |
| `tags_create` | `tags.create` | write | `tags:write` | all |
| `tags_delete` | `tags.delete` | destructive | `tags:write` | all |
| `tags_list` | `tags.list` | read | `tags:read` | read, all |
| `tags_update` | `tags.update` | write | `tags:write` | all |
| `transactions_create` | `transactions.create` | write | `transactions:write` | transactions, all |
| `transactions_delete` | `transactions.delete` | destructive | `transactions:write` | transactions, all |
| `transactions_list` | `transactions.list` | read | `transactions:read` | read, transactions, all |
| `transactions_show` | `transactions.show` | read | `transactions:read` | read, transactions, all |
| `transactions_update` | `transactions.update` | write | `transactions:write` | transactions, all |
| `transfers_create` | `transfers.create` | write | `transfers:write` | transactions, all |
| `transfers_delete` | `transfers.delete` | destructive | `transfers:write` | transactions, all |

Operations with no tool, each with the reason:

- `export.transactions`: it streams a file, and a tool answers with JSON that fits a model context; the reads (transactions.list, reports.*) cover it
- `export.workspace`: it streams a file, and a tool answers with JSON that fits a model context; the reads (transactions.list, reports.*) cover it

## Resources

| Name | Description |
| --- | --- |
| `finances://accounts` | The accounts with their balances. |
| `finances://categories` | The categories. |
| `finances://workspace` | The workspace overview: currency, this month and the net worth. |

## Resource templates

| Name | Description |
| --- | --- |
| `finances://budgets/{month}` | The budgets of a month (YYYY-MM) with what the month spent. |
| `finances://reports/{kind}/{period}` | A report: kind is cashflow, categories, accounts, tags, networth, compare; period is this_month, last_month, last_3_months, last_12_months, this_year or FROM_TO as 2026-01-01_2026-03-31. |

## Prompts

| Name | Description |
| --- | --- |
| `budget-plan` | Propose monthly budgets from the recent spending. |
| `categorize-uncategorized` | Find transactions with no category and propose one for each. |
| `find-unusual-spending` | Look for spending that stands out from the usual. |
| `import-statement` | Turn a pasted bank statement into transactions, after you approve them. |
| `monthly-review` | Review a month: where the money went and what changed. |

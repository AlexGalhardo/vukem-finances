# CLI reference

<!-- Generated from the operation registry: do not edit. Run `go run ./cmd/app cli:reference`. -->

Every operation of the registry is a command `<resource> <verb> [arguments] [flags]`.
The same registry feeds the REST API (`docs/api.md`), so a command and its endpoint
take the same input and give the same answer.

## Modes

- **Local** (`--local`): runs in this process as the owner of a personal installation
  (`APP_EDITION=personal`). Any other edition exits 3 and changes nothing.
- **Remote** (`--server URL` or the stored login): calls the REST API with a token.
  `auth login --server URL` reads the token from the standard input; `FINANCES_TOKEN`
  is read for CI. A token is never a flag.

## Shared flags

| Flag | Meaning |
| --- | --- |
| `--output table, json or csv` | Output format. The default is a table on a terminal and JSON when piped. |
| `--input FILE` or `--input -` | The whole input as a JSON object, from a file or the standard input. |
| `--server URL` | Run against this server. |
| `--local` | Run in this process (personal edition). |
| `--if-match VERSION` | Optimistic concurrency, for commands that change data. |
| `--idempotency-key KEY` | Safe retries, for commands that honor a key. |
| `--yes` | Skip the confirmation of a destructive command; required without a terminal. |
| `--out FILE` | Where a download is written (required on a terminal). |

An amount flag such as `--amount` takes `12.50` or `1.234,56`; its `--amount-cents`
twin takes whole cents. JSON output keeps amounts as exact decimal text next to integer cents.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Any other failure |
| 2 | Usage error, or a destructive command without a terminal and without `--yes` |
| 3 | Authentication or permission |
| 4 | Not found |
| 5 | Conflict |
| 6 | Rate limit or plan limit |

## Other commands

- `auth login --server URL`, `auth logout`, `auth status`: manage the stored login.
- `completion bash|zsh|fish|powershell [program]`: print a shell completion script.

## accounts

### `accounts list`

List the accounts with their balances.

```sh
accounts list [flags]
```

- REST: `GET /accounts`
- Operation: `accounts.list`, scope `accounts:read`, read

### `accounts show`

Read an account.

```sh
accounts show <uid> [flags]
```

- REST: `GET /accounts/{uid}`
- Operation: `accounts.show`, scope `accounts:read`, read

### `accounts create`

Create an account.

```sh
accounts create [flags]
```

- REST: `POST /accounts`
- Operation: `accounts.create`, scope `accounts:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--uid` | text |  |
| `--name` | text | yes |
| `--kind` | text | yes |
| `--opening-cents` | whole cents |  |
| `--opening` | amount, `12.50` or `1.234,56` | |

### `accounts update`

Change the name, kind and opening balance of an account.

```sh
accounts update <uid> [flags]
```

- REST: `PATCH /accounts/{uid}`
- Operation: `accounts.update`, scope `accounts:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--kind` | text |  |
| `--opening-cents` | whole cents |  |
| `--opening` | amount, `12.50` or `1.234,56` | |

### `accounts archive`

Archive an account.

```sh
accounts archive <uid> [flags]
```

- REST: `POST /accounts/{uid}/archive`
- Operation: `accounts.archive`, scope `accounts:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## categories

### `categories list`

List the categories.

```sh
categories list [flags]
```

- REST: `GET /categories`
- Operation: `categories.list`, scope `categories:read`, read

### `categories show`

Read a category.

```sh
categories show <uid> [flags]
```

- REST: `GET /categories/{uid}`
- Operation: `categories.show`, scope `categories:read`, read

### `categories create`

Create a category.

```sh
categories create [flags]
```

- REST: `POST /categories`
- Operation: `categories.create`, scope `categories:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--uid` | text |  |
| `--name` | text | yes |
| `--kind` | text |  |
| `--parent-uid` | text |  |
| `--color` | text |  |
| `--icon` | text |  |

### `categories update`

Change a category.

```sh
categories update <uid> [flags]
```

- REST: `PATCH /categories/{uid}`
- Operation: `categories.update`, scope `categories:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--parent-uid` | text |  |
| `--color` | text |  |
| `--icon` | text |  |

### `categories archive`

Archive a category.

```sh
categories archive <uid> [flags]
```

- REST: `POST /categories/{uid}/archive`
- Operation: `categories.archive`, scope `categories:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## tags

### `tags list`

List the tags.

```sh
tags list [flags]
```

- REST: `GET /tags`
- Operation: `tags.list`, scope `tags:read`, read

### `tags create`

Create a tag.

```sh
tags create [flags]
```

- REST: `POST /tags`
- Operation: `tags.create`, scope `tags:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--uid` | text |  |
| `--name` | text | yes |
| `--color` | text |  |

### `tags update`

Rename or recolor a tag.

```sh
tags update <uid> [flags]
```

- REST: `PATCH /tags/{uid}`
- Operation: `tags.update`, scope `tags:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--color` | text |  |

### `tags delete`

Delete a tag and take it off its transactions.

```sh
tags delete <uid> [flags]
```

- REST: `DELETE /tags/{uid}`
- Operation: `tags.delete`, scope `tags:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## transactions

### `transactions list`

List transactions, newest first, with filters.

```sh
transactions list [flags]
```

- REST: `GET /transactions`
- Operation: `transactions.list`, scope `transactions:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--from` | text |  |
| `--to` | text |  |
| `--account` | text |  |
| `--category` | text |  |
| `--tag` | text |  |
| `--kind` | text |  |
| `--q` | text |  |
| `--min-cents` | whole cents |  |
| `--min` | amount, `12.50` or `1.234,56` | |
| `--max-cents` | whole cents |  |
| `--max` | amount, `12.50` or `1.234,56` | |
| `--page` | number |  |

### `transactions show`

Read a transaction.

```sh
transactions show <uid> [flags]
```

- REST: `GET /transactions/{uid}`
- Operation: `transactions.show`, scope `transactions:read`, read

### `transactions create`

Record a transaction.

```sh
transactions create [flags]
```

- REST: `POST /transactions`
- Operation: `transactions.create`, scope `transactions:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--uid` | text |  |
| `--account-uid` | text | yes |
| `--category-uid` | text |  |
| `--kind` | text | yes |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text | yes |
| `--description` | text |  |
| `--tags` | text, repeat for more than one |  |

### `transactions update`

Change a transaction.

```sh
transactions update <uid> [flags]
```

- REST: `PATCH /transactions/{uid}`
- Operation: `transactions.update`, scope `transactions:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--account-uid` | text |  |
| `--category-uid` | text |  |
| `--kind` | text |  |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text |  |
| `--description` | text |  |
| `--tags` | text, repeat for more than one |  |

### `transactions delete`

Delete a transaction.

```sh
transactions delete <uid> [flags]
```

- REST: `DELETE /transactions/{uid}`
- Operation: `transactions.delete`, scope `transactions:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## transfers

### `transfers create`

Move money between two accounts.

```sh
transfers create [flags]
```

- REST: `POST /transfers`
- Operation: `transfers.create`, scope `transfers:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--from-account-uid` | text | yes |
| `--to-account-uid` | text | yes |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text | yes |
| `--description` | text |  |

### `transfers delete`

Delete a transfer and both of its legs.

```sh
transfers delete <uid> [flags]
```

- REST: `DELETE /transfers/{uid}`
- Operation: `transfers.delete`, scope `transfers:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## budgets

### `budgets list`

List the budgets with what the month spent.

```sh
budgets list [flags]
```

- REST: `GET /budgets`
- Operation: `budgets.list`, scope `budgets:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--month` | text |  |

### `budgets set`

Set the monthly limit of an expense category.

```sh
budgets set [flags]
```

- REST: `POST /budgets`
- Operation: `budgets.set`, scope `budgets:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--category-uid` | text | yes |
| `--limit-cents` | whole cents |  |
| `--limit` | amount, `12.50` or `1.234,56` | |

### `budgets delete`

Remove a budget.

```sh
budgets delete <uid> [flags]
```

- REST: `DELETE /budgets/{uid}`
- Operation: `budgets.delete`, scope `budgets:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## summary

### `summary show`

Income, expense, net, account balances and net worth of a month.

```sh
summary show [flags]
```

- REST: `GET /summary`
- Operation: `summary.show`, scope `summary:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--month` | text |  |

## cards

### `cards list`

List the credit cards with their limits and statements.

```sh
cards list [flags]
```

- REST: `GET /cards`
- Operation: `cards.list`, scope `cards:read`, read

### `cards show`

Read a credit card.

```sh
cards show <uid> [flags]
```

- REST: `GET /cards/{uid}`
- Operation: `cards.show`, scope `cards:read`, read

### `cards create`

Create a credit card with its account.

```sh
cards create [flags]
```

- REST: `POST /cards`
- Operation: `cards.create`, scope `cards:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text | yes |
| `--limit-cents` | whole cents |  |
| `--limit` | amount, `12.50` or `1.234,56` | |
| `--closing-day` | number |  |
| `--due-day` | number |  |
| `--payment-account-uid` | text |  |
| `--brand` | text |  |
| `--last4` | text |  |

### `cards update`

Change a credit card.

```sh
cards update <uid> [flags]
```

- REST: `PATCH /cards/{uid}`
- Operation: `cards.update`, scope `cards:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--limit-cents` | whole cents |  |
| `--limit` | amount, `12.50` or `1.234,56` | |
| `--closing-day` | number |  |
| `--due-day` | number |  |
| `--payment-account-uid` | text |  |
| `--brand` | text |  |
| `--last4` | text |  |

### `cards archive`

Archive a credit card.

```sh
cards archive <uid> [flags]
```

- REST: `POST /cards/{uid}/archive`
- Operation: `cards.archive`, scope `cards:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

### `cards unarchive`

Bring an archived credit card back.

```sh
cards unarchive <uid> [flags]
```

- REST: `POST /cards/{uid}/unarchive`
- Operation: `cards.unarchive`, scope `cards:write`, write

## statements

### `statements list`

List the statements of a credit card, newest first.

```sh
statements list <uid> [flags]
```

- REST: `GET /cards/{uid}/statements`
- Operation: `statements.list`, scope `statements:read`, read

### `statements show`

Read a statement of a credit card with its rows.

```sh
statements show <uid> <ref> [flags]
```

- REST: `GET /cards/{uid}/statements/{ref}`
- Operation: `statements.show`, scope `statements:read`, read

### `statements pay`

Pay a statement from a bank account.

```sh
statements pay <uid> <ref> [flags]
```

- REST: `POST /cards/{uid}/statements/{ref}/pay`
- Operation: `statements.pay`, scope `statements:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--from-account-uid` | text | yes |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text |  |

## recurrences

### `recurrences list`

List the recurring rules.

```sh
recurrences list [flags]
```

- REST: `GET /recurrences`
- Operation: `recurrences.list`, scope `recurrences:read`, read

### `recurrences create`

Create a recurring rule and post what is already due.

```sh
recurrences create [flags]
```

- REST: `POST /recurrences`
- Operation: `recurrences.create`, scope `recurrences:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--account-uid` | text | yes |
| `--category-uid` | text |  |
| `--kind` | text | yes |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--description` | text |  |
| `--frequency` | text | yes |
| `--interval` | number |  |
| `--start-date` | text | yes |
| `--end-date` | text |  |
| `--max-count` | number |  |
| `--backfill` | true or false |  |

### `recurrences update`

Change a recurring rule.

```sh
recurrences update <uid> [flags]
```

- REST: `PATCH /recurrences/{uid}`
- Operation: `recurrences.update`, scope `recurrences:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--account-uid` | text |  |
| `--category-uid` | text |  |
| `--kind` | text |  |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--description` | text |  |
| `--frequency` | text |  |
| `--interval` | number |  |
| `--start-date` | text |  |
| `--end-date` | text |  |
| `--max-count` | number |  |
| `--active` | true or false |  |

### `recurrences delete`

Delete a recurring rule and keep what it generated.

```sh
recurrences delete <uid> [flags]
```

- REST: `DELETE /recurrences/{uid}`
- Operation: `recurrences.delete`, scope `recurrences:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

| Flag | Value | Required |
| --- | --- | --- |
| `--future` | true or false |  |

## goals

### `goals list`

List the savings goals with their progress.

```sh
goals list [flags]
```

- REST: `GET /goals`
- Operation: `goals.list`, scope `goals:read`, read

### `goals create`

Create a savings goal.

```sh
goals create [flags]
```

- REST: `POST /goals`
- Operation: `goals.create`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text | yes |
| `--target-cents` | whole cents |  |
| `--target` | amount, `12.50` or `1.234,56` | |
| `--target-date` | text |  |

### `goals update`

Change the name, target and date of a goal.

```sh
goals update <uid> [flags]
```

- REST: `PATCH /goals/{uid}`
- Operation: `goals.update`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--target-cents` | whole cents |  |
| `--target` | amount, `12.50` or `1.234,56` | |
| `--target-date` | text |  |

### `goals archive`

Archive a goal; its contributions stay.

```sh
goals archive <uid> [flags]
```

- REST: `POST /goals/{uid}/archive`
- Operation: `goals.archive`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

### `goals unarchive`

Bring an archived goal back.

```sh
goals unarchive <uid> [flags]
```

- REST: `POST /goals/{uid}/unarchive`
- Operation: `goals.unarchive`, scope `goals:write`, write

### `goals delete`

Delete a goal and its contributions.

```sh
goals delete <uid> [flags]
```

- REST: `DELETE /goals/{uid}`
- Operation: `goals.delete`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

### `goals contributions-add`

Put an amount towards a goal.

```sh
goals contributions-add <uid> [flags]
```

- REST: `POST /goals/{uid}/contributions`
- Operation: `goals.contributions.add`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text |  |
| `--note` | text |  |

### `goals contributions-delete`

Delete a contribution of a goal.

```sh
goals contributions-delete <uid> <contribution_uid> [flags]
```

- REST: `DELETE /goals/{uid}/contributions/{contribution_uid}`
- Operation: `goals.contributions.delete`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## reports

### `reports cashflow`

Income against expense per month of a period.

```sh
reports cashflow [flags]
```

- REST: `GET /reports/cash-flow`
- Operation: `reports.cashflow`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |

### `reports categories`

Spending or income of a period by category.

```sh
reports categories [flags]
```

- REST: `GET /reports/categories`
- Operation: `reports.categories`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |
| `--kind` | text |  |

### `reports accounts`

Income and expense of a period by account.

```sh
reports accounts [flags]
```

- REST: `GET /reports/accounts`
- Operation: `reports.accounts`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |

### `reports tags`

Income and expense of a period by tag.

```sh
reports tags [flags]
```

- REST: `GET /reports/tags`
- Operation: `reports.tags`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |

### `reports networth`

The net worth at the end of each month of a period.

```sh
reports networth [flags]
```

- REST: `GET /reports/net-worth`
- Operation: `reports.networth`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |

### `reports compare`

A period against the one right before it.

```sh
reports compare [flags]
```

- REST: `GET /reports/compare`
- Operation: `reports.compare`, scope `reports:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--period` | text |  |
| `--from` | text |  |
| `--to` | text |  |

## export

### `export transactions`

Download the transactions that match the filters as a CSV file.

```sh
export transactions [flags]
```

- REST: `GET /export/transactions`
- Operation: `export.transactions`, scope `export:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--from` | text |  |
| `--to` | text |  |
| `--account` | text |  |
| `--category` | text |  |
| `--tag` | text |  |
| `--kind` | text |  |
| `--q` | text |  |
| `--min-cents` | whole cents |  |
| `--min` | amount, `12.50` or `1.234,56` | |
| `--max-cents` | whole cents |  |
| `--max` | amount, `12.50` or `1.234,56` | |
| `--delimiter` | text |  |
| `--bom` | true or false |  |

### `export workspace`

Download the whole workspace as one JSON document (owner and admins).

```sh
export workspace [flags]
```

- REST: `GET /export/workspace`
- Operation: `export.workspace`, scope `export:read`, read

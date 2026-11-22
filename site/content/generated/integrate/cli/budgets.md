---
title: "CLI: budgets"
description: "The budgets commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 6
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `budgets list`

List the budgets with what the month spent.

```sh
budgets list [flags]
```

- REST: `GET /budgets`
- Operation: `budgets.list`, scope `budgets:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--month` | text |  |

## `budgets set`

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

## `budgets delete`

Remove a budget.

```sh
budgets delete <uid> [flags]
```

- REST: `DELETE /budgets/{uid}`
- Operation: `budgets.delete`, scope `budgets:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

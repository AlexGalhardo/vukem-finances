---
title: "CLI: transactions"
description: "The transactions commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 4
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `transactions list`

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

## `transactions show`

Read a transaction.

```sh
transactions show <uid> [flags]
```

- REST: `GET /transactions/{uid}`
- Operation: `transactions.show`, scope `transactions:read`, read

## `transactions create`

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

## `transactions update`

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

## `transactions delete`

Delete a transaction.

```sh
transactions delete <uid> [flags]
```

- REST: `DELETE /transactions/{uid}`
- Operation: `transactions.delete`, scope `transactions:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

---
title: "CLI: recurrences"
description: "The recurrences commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 10
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `recurrences list`

List the recurring rules.

```sh
recurrences list [flags]
```

- REST: `GET /recurrences`
- Operation: `recurrences.list`, scope `recurrences:read`, read

## `recurrences create`

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

## `recurrences update`

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

## `recurrences delete`

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

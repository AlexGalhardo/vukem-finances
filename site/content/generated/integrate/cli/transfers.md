---
title: "CLI: transfers"
description: "The transfers commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 5
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `transfers create`

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

## `transfers delete`

Delete a transfer and both of its legs.

```sh
transfers delete <uid> [flags]
```

- REST: `DELETE /transfers/{uid}`
- Operation: `transfers.delete`, scope `transfers:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

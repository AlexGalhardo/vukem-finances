---
title: "CLI: export"
description: "The export commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 13
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `export transactions`

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

## `export workspace`

Download the whole workspace as one JSON document (owner and admins).

```sh
export workspace [flags]
```

- REST: `GET /export/workspace`
- Operation: `export.workspace`, scope `export:read`, read

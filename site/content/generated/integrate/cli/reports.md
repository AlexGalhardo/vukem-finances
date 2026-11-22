---
title: "CLI: reports"
description: "The reports commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 12
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `reports cashflow`

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

## `reports categories`

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

## `reports accounts`

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

## `reports tags`

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

## `reports networth`

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

## `reports compare`

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

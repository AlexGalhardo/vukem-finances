---
title: "CLI: cards"
description: "The cards commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 8
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `cards list`

List the credit cards with their limits and statements.

```sh
cards list [flags]
```

- REST: `GET /cards`
- Operation: `cards.list`, scope `cards:read`, read

## `cards show`

Read a credit card.

```sh
cards show <uid> [flags]
```

- REST: `GET /cards/{uid}`
- Operation: `cards.show`, scope `cards:read`, read

## `cards create`

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

## `cards update`

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

## `cards archive`

Archive a credit card.

```sh
cards archive <uid> [flags]
```

- REST: `POST /cards/{uid}/archive`
- Operation: `cards.archive`, scope `cards:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## `cards unarchive`

Bring an archived credit card back.

```sh
cards unarchive <uid> [flags]
```

- REST: `POST /cards/{uid}/unarchive`
- Operation: `cards.unarchive`, scope `cards:write`, write

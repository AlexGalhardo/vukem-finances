---
title: "CLI: accounts"
description: "The accounts commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 1
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `accounts list`

List the accounts with their balances.

```sh
accounts list [flags]
```

- REST: `GET /accounts`
- Operation: `accounts.list`, scope `accounts:read`, read

## `accounts show`

Read an account.

```sh
accounts show <uid> [flags]
```

- REST: `GET /accounts/{uid}`
- Operation: `accounts.show`, scope `accounts:read`, read

## `accounts create`

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

## `accounts update`

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

## `accounts archive`

Archive an account.

```sh
accounts archive <uid> [flags]
```

- REST: `POST /accounts/{uid}/archive`
- Operation: `accounts.archive`, scope `accounts:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

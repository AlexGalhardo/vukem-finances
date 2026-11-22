---
title: "CLI: statements"
description: "The statements commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 9
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `statements list`

List the statements of a credit card, newest first.

```sh
statements list <uid> [flags]
```

- REST: `GET /cards/{uid}/statements`
- Operation: `statements.list`, scope `statements:read`, read

## `statements show`

Read a statement of a credit card with its rows.

```sh
statements show <uid> <ref> [flags]
```

- REST: `GET /cards/{uid}/statements/{ref}`
- Operation: `statements.show`, scope `statements:read`, read

## `statements pay`

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

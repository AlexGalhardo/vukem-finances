---
title: "CLI: categories"
description: "The categories commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 2
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `categories list`

List the categories.

```sh
categories list [flags]
```

- REST: `GET /categories`
- Operation: `categories.list`, scope `categories:read`, read

## `categories show`

Read a category.

```sh
categories show <uid> [flags]
```

- REST: `GET /categories/{uid}`
- Operation: `categories.show`, scope `categories:read`, read

## `categories create`

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

## `categories update`

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

## `categories archive`

Archive a category.

```sh
categories archive <uid> [flags]
```

- REST: `POST /categories/{uid}/archive`
- Operation: `categories.archive`, scope `categories:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

---
title: "CLI: tags"
description: "The tags commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 3
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `tags list`

List the tags.

```sh
tags list [flags]
```

- REST: `GET /tags`
- Operation: `tags.list`, scope `tags:read`, read

## `tags create`

Create a tag.

```sh
tags create [flags]
```

- REST: `POST /tags`
- Operation: `tags.create`, scope `tags:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--uid` | text |  |
| `--name` | text | yes |
| `--color` | text |  |

## `tags update`

Rename or recolor a tag.

```sh
tags update <uid> [flags]
```

- REST: `PATCH /tags/{uid}`
- Operation: `tags.update`, scope `tags:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--color` | text |  |

## `tags delete`

Delete a tag and take it off its transactions.

```sh
tags delete <uid> [flags]
```

- REST: `DELETE /tags/{uid}`
- Operation: `tags.delete`, scope `tags:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

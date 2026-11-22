---
title: "CLI: goals"
description: "The goals commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 11
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `goals list`

List the savings goals with their progress.

```sh
goals list [flags]
```

- REST: `GET /goals`
- Operation: `goals.list`, scope `goals:read`, read

## `goals create`

Create a savings goal.

```sh
goals create [flags]
```

- REST: `POST /goals`
- Operation: `goals.create`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text | yes |
| `--target-cents` | whole cents |  |
| `--target` | amount, `12.50` or `1.234,56` | |
| `--target-date` | text |  |

## `goals update`

Change the name, target and date of a goal.

```sh
goals update <uid> [flags]
```

- REST: `PATCH /goals/{uid}`
- Operation: `goals.update`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--name` | text |  |
| `--target-cents` | whole cents |  |
| `--target` | amount, `12.50` or `1.234,56` | |
| `--target-date` | text |  |

## `goals archive`

Archive a goal; its contributions stay.

```sh
goals archive <uid> [flags]
```

- REST: `POST /goals/{uid}/archive`
- Operation: `goals.archive`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## `goals unarchive`

Bring an archived goal back.

```sh
goals unarchive <uid> [flags]
```

- REST: `POST /goals/{uid}/unarchive`
- Operation: `goals.unarchive`, scope `goals:write`, write

## `goals delete`

Delete a goal and its contributions.

```sh
goals delete <uid> [flags]
```

- REST: `DELETE /goals/{uid}`
- Operation: `goals.delete`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

## `goals contributions-add`

Put an amount towards a goal.

```sh
goals contributions-add <uid> [flags]
```

- REST: `POST /goals/{uid}/contributions`
- Operation: `goals.contributions.add`, scope `goals:write`, write

| Flag | Value | Required |
| --- | --- | --- |
| `--amount-cents` | whole cents |  |
| `--amount` | amount, `12.50` or `1.234,56` | |
| `--date` | text |  |
| `--note` | text |  |

## `goals contributions-delete`

Delete a contribution of a goal.

```sh
goals contributions-delete <uid> <contribution_uid> [flags]
```

- REST: `DELETE /goals/{uid}/contributions/{contribution_uid}`
- Operation: `goals.contributions.delete`, scope `goals:write`, destructive
- Asks for confirmation on a terminal; needs `--yes` otherwise.

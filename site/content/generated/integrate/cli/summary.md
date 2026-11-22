---
title: "CLI: summary"
description: "The summary commands of the vukem-finances command line: arguments, flags, scopes and the REST route of each."
order: 7
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## `summary show`

Income, expense, net, account balances and net worth of a month.

```sh
summary show [flags]
```

- REST: `GET /summary`
- Operation: `summary.show`, scope `summary:read`, read

| Flag | Value | Required |
| --- | --- | --- |
| `--month` | text |  |

---
title: "CLI: modes, flags, exit codes"
description: "How the command line runs locally or against a server, the flags every command shares and its exit codes."
order: 0
---

> Generated from `docs/reference/cli.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

Every operation of the registry is a command `<resource> <verb> [arguments] [flags]`.
The same registry feeds the REST API (`docs/api.md`), so a command and its endpoint
take the same input and give the same answer.

## Modes

- **Local** (`--local`): runs in this process as the owner of a personal installation
  (`APP_EDITION=personal`). Any other edition exits 3 and changes nothing.
- **Remote** (`--server URL` or the stored login): calls the REST API with a token.
  `auth login --server URL` reads the token from the standard input; `FINANCES_TOKEN`
  is read for CI. A token is never a flag.

## Shared flags

| Flag | Meaning |
| --- | --- |
| `--output table, json or csv` | Output format. The default is a table on a terminal and JSON when piped. |
| `--input FILE` or `--input -` | The whole input as a JSON object, from a file or the standard input. |
| `--server URL` | Run against this server. |
| `--local` | Run in this process (personal edition). |
| `--if-match VERSION` | Optimistic concurrency, for commands that change data. |
| `--idempotency-key KEY` | Safe retries, for commands that honor a key. |
| `--yes` | Skip the confirmation of a destructive command; required without a terminal. |
| `--out FILE` | Where a download is written (required on a terminal). |

An amount flag such as `--amount` takes `12.50` or `1.234,56`; its `--amount-cents`
twin takes whole cents. JSON output keeps amounts as exact decimal text next to integer cents.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Any other failure |
| 2 | Usage error, or a destructive command without a terminal and without `--yes` |
| 3 | Authentication or permission |
| 4 | Not found |
| 5 | Conflict |
| 6 | Rate limit or plan limit |

## Other commands

- `auth login --server URL`, `auth logout`, `auth status`: manage the stored login.
- `completion bash|zsh|fish|powershell [program]`: print a shell completion script.

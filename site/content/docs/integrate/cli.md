---
title: "Command line"
description: "Use the vukem-finances command line locally as the owner of a personal installation, or against a server with a token, with table, JSON and CSV output."
---

The command line is the third adapter over the operation registry. Every operation offered on the CLI surface is a command `<resource> <verb>`, with flags generated from the input type, so a command and its endpoint take the same input and give the same answer.

The binary is the application itself. A resource command is a subcommand of it:

```sh
app accounts list
app transactions list --output json
```

The generated reference lists [the modes, flags and exit codes](/docs/integrate/cli/overview/) and one page per resource: [accounts](/docs/integrate/cli/accounts/), [budgets](/docs/integrate/cli/budgets/), [cards](/docs/integrate/cli/cards/), [categories](/docs/integrate/cli/categories/), [export](/docs/integrate/cli/export/), [goals](/docs/integrate/cli/goals/), [recurrences](/docs/integrate/cli/recurrences/), [reports](/docs/integrate/cli/reports/), [statements](/docs/integrate/cli/statements/), [summary](/docs/integrate/cli/summary/), [tags](/docs/integrate/cli/tags/), [transactions](/docs/integrate/cli/transactions/) and [transfers](/docs/integrate/cli/transfers/).

## Two modes

- **Local.** `--local` runs in this process as the owner of a personal installation. It needs no network and no token, and it works only with `APP_EDITION=personal`: any other edition exits with code 3 and changes nothing.
- **Remote.** `--server URL`, or a stored login, calls the REST API with a token. Sign in with `auth login --server URL`, which reads the token from the standard input, or set `FINANCES_TOKEN` in CI. A token is never a flag, so it never lands in your shell history.

```sh
app auth login --server https://finance.example.com
app auth status
app accounts list
app auth logout
```

## Output

The default is a table on a terminal and JSON when the output is piped. `--output table`, `--output json` or `--output csv` choose it. Table and CSV text is cleaned of control characters, and the CSV guards against spreadsheet formulas.

## Amounts

An amount flag such as `--amount` takes `12.50` or `1.234,56`, and has a `--amount-cents` twin that takes whole cents. Money is never a floating point number.

## Safe by default

- A destructive command asks for confirmation on a terminal and needs `--yes` without one.
- `--idempotency-key` makes a retry safe for the commands that honor a key, and `--if-match` checks the version of the row you read.
- `--input FILE` or `--input -` gives the whole input as a JSON object.

## Shell completion

```sh
app completion bash
```

`bash`, `zsh`, `fish` and `powershell` are supported. The reference is regenerated from the registry, so a command that exists is documented and one that does not is not.

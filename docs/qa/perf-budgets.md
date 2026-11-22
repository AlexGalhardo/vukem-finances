# Performance budgets

`go run ./scripts/qa/loadtest -check` builds the app, seeds one workspace of 100000 transactions and 50 workspaces of 2000, starts the server and drives it with 20 concurrent clients, then compares each p95 with the table below. A missed budget, a failed request or any `database is locked` error fails the run. Budgets are never loosened to pass a run: a miss is a finding.

The first targets come from the plan of epic 24: p95 under 300 ms for lists and reports, under 1 s for an export of 50000 rows, and no `database is locked` error under mixed load.

## How it is measured

- One personal-edition server (SQLite, the default), `APP_ENV=development` with debug off, on the machine that runs the test. The numbers belong to that machine: a slower runner needs a rerun, not a bigger budget.
- Requests per scenario: 300 over 20 clients (the export: 8, the most one workspace may download a minute, see `config.ExportLimit`). Each client has its own token and sync device, because the API limits count per token and per device; a `429` counts as a failed request.
- The big workspace holds 100000 transactions spread over 24 months, so the 12 month export holds about 50000 rows. The small workspaces feed `api_list_small`.
- `dashboard` is the signed-in web page; the others are the public API, and the `mcp_*` rows are the stdio MCP server, one process per client.
- `mixed_load` runs dashboard, list, report, sync pull, sync push and create at the same time.

## Budgets

| Scenario | What it does | p95 budget (ms) |
| --- | --- | --- |
| `dashboard` | the dashboard page of the big workspace | 300 |
| `transactions_list` | a page of the transaction list | 300 |
| `transactions_filtered` | the list with date, kind, account and text filters | 300 |
| `reports` | cash flow, categories, accounts, net worth and comparison over 12 months | 300 |
| `export_csv_50k` | the CSV export of about 50000 rows | 1000 |
| `api_list_small` | the list of a small workspace | 300 |
| `sync_pull` | a pull page of 500 changes | 300 |
| `sync_push` | a push of 20 new transactions | 300 |
| `mcp_tools_list` | `tools/list` on stdio | 300 |
| `mcp_transactions_list` | the `transactions_list` tool on stdio | 300 |
| `mixed_load` | all of the above at once | 300 |

## Measurements

Last run after the covering indexes of QA-503 (migration `20261126120000`), on a developer machine shared with other work, so absolute numbers move by tens of percent between runs. The web rate limit counts per user: the big workspace has one member per client and each client signs in as its own member, so no 429 counts as a failure on `dashboard` (no product limit is changed). `database is locked`: none, in any client response or the server log.

| Scenario | 20 clients p50 (ms) | 20 clients p95 (ms) | 1 client p50 (ms) | 1 client p95 (ms) | Budget (ms) | 20 clients |
| --- | --- | --- | --- | --- | --- | --- |
| `dashboard` | 1271 | 1626 | 376 | 448 | 300 | over |
| `transactions_list` | 299 | 679 | 23 | 39 | 300 | over |
| `transactions_filtered` | 1664 | 2954 | 60 | 108 | 300 | over |
| `reports` | 1080 | 2437 | 213 | 359 | 300 | over |
| `export_csv_50k` | 1727 | 1749 | 1453 | 1478 | 1000 | over |
| `api_list_small` | 13 | 28 | 5 | 7 | 300 | ok |
| `sync_pull` | 64 | 89 | 23 | 28 | 300 | ok |
| `sync_push` | 1025 | 3869 | 52 | 145 | 300 | over |
| `mcp_tools_list` | 3 | 9 | 1 | 1 | 300 | ok |
| `mcp_transactions_list` | 147 | 225 | 23 | 30 | 300 | ok |
| `mixed_load` | 182 | 1264 | 20 | 292 | 300 | over |

With one client the single-device limit (30 pushes a minute per device) makes `mixed_load` answer a few 429s after `sync_push` used the same device; with 20 clients each device stays under it.

Before QA-500 and QA-503 (one machine, same load): dashboard 739 ms p50 for one client, list page 296 ms, push 866 ms. The first 20-client run missed every budget by 5 to 25 times.

What is left, by the profile: with one client the dashboard (448 ms p95), the reports (359 ms) and the export (1478 ms) still miss; with 20 clients every request that scans the workspace queues on the CPU of the machine. Each of those reads aggregates all rows of a period at request time (a covering index scan of 100000 entries costs about 100 ms by itself), and the reports group by amount to keep exact sums without `SUM` overflow, so a group per row. See QA-503.

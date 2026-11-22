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

Last run after the monthly summary of QA-503 (migration `20261127100000`, the dashboard, the budgets and the reports read `transaction_month_totals`) and the plain-column CSV export, on a developer machine shared with other work, so absolute numbers move by tens of percent between runs. The web rate limit counts per user: the big workspace has one member per client and each client signs in as its own member, so no 429 counts as a failure on `dashboard` (no product limit is changed). `database is locked`: none, in any client response or the server log.

The 1 client column is `go run ./scripts/qa/loadtest -check -clients 1 -requests 15`: with one user the read limit (120 a minute, `config.ReadLimit`) answers 429 to a longer run now that the requests are fast, so it holds 15 samples per scenario (the export is always 8 downloads at once, 10 an hour is its limit).

| Scenario | 20 clients p50 (ms) | 20 clients p95 (ms) | 1 client p50 (ms) | 1 client p95 (ms) | Budget (ms) | 20 clients |
| --- | --- | --- | --- | --- | --- | --- |
| `dashboard` | 55 | 77 | 10 | 13 | 300 | ok |
| `transactions_list` | 159 | 230 | 21 | 24 | 300 | ok |
| `transactions_filtered` | 1057 | 1541 | 30 | 68 | 300 | over |
| `reports` | 8 | 13 | 2 | 4 | 300 | ok |
| `export_csv_50k` | 732 | 751 | 728 | 754 | 1000 | ok |
| `api_list_small` | 10 | 233 | 3 | 3 | 300 | ok |
| `sync_pull` | 49 | 64 | 11 | 14 | 300 | ok |
| `sync_push` | 624 | 1576 | 30 | 45 | 300 | over |
| `mcp_tools_list` | 2 | 2 | 1 | 2 | 300 | ok |
| `mcp_transactions_list` | 134 | 156 | 21 | 26 | 300 | ok |
| `mixed_load` | 42 | 243 | 9 | 23 | 300 | ok |

Before QA-500 and QA-503 (one machine, same load): dashboard 739 ms p50 for one client, list page 296 ms, push 866 ms. The first 20-client run missed every budget by 5 to 25 times. After the covering indexes of `20261126120000` the dashboard was 448 ms, the reports 359 ms and the export 1478 ms p95 with one client.

What is left, by the profile, with 20 clients: `transactions_filtered` (1541 ms p95, 5.1 times the budget) scans the date range of the workspace for its `LIKE '%text%'` filter (68 ms alone, queueing on the CPU of the machine under 20 clients) and `sync_push` (1576 ms p95, 5.3 times the budget) is 20 mutations a push written one after the other by the single SQLite writer (45 ms alone, so 20 clients wait their turn). Neither is an aggregate the summary can answer. See QA-503.

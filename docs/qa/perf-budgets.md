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

Last run, one developer machine shared with other work, 20 clients, `go run ./scripts/qa/loadtest -check` (exit 1, budgets not met):

| Scenario | p50 (ms) | p95 (ms) | Budget (ms) | Result |
| --- | --- | --- | --- | --- |
| `dashboard` | 4440 | 7493 | 300 | over; 72 of 300 requests answered 429 (the per-user web rate limit) |
| `transactions_list` | 2427 | 4557 | 300 | over |
| `transactions_filtered` | 1114 | 1555 | 300 | over |
| `reports` | 1264 | 2578 | 300 | over |
| `export_csv_50k` | 1567 | 1602 | 1000 | over |
| `api_list_small` | 35 | 61 | 300 | ok |
| `sync_pull` | 72 | 110 | 300 | ok |
| `sync_push` | 1058 | 2571 | 300 | over |
| `mcp_tools_list` | 2 | 5 | 300 | ok |
| `mcp_transactions_list` | 1984 | 3235 | 300 | over |
| `mixed_load` | 974 | 3598 | 300 | over |

No `database is locked` error appeared in any client response or in the server log.

The same run with one client (`-clients 1 -requests 30`), which shows the cost of a request with no queueing: dashboard 473 ms p50, list page 253 ms, reports 166 ms (p95 639), export 1196 ms, sync push 40 ms (866 ms before QA-500), sync pull 11 ms.

The misses are recorded as QA-503 in `rounds.md`.

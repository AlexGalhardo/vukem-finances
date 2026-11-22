# Accepted known issues

Findings the author accepted, with the reason. High and Critical findings are never listed here: they block CI. This list holds what the static and dependency checks (`golangci-lint`, `go tool govulncheck ./...`, `bun audit --audit-level=high`) report below that level, or a result that cannot be fixed in this repository.

Each row needs the author's approval before the campaign relies on it.

## Dependency advisories below High

| Where | Package | Severity | Advisory | Why it is accepted |
| --- | --- | --- | --- | --- |
| root | `katex` through `markdownlint-cli2` | low | GHSA-238p-pmpm-9mq7 | Dev tool only (the Markdown linter); it never ships and never sees untrusted input. |
| `mobile/` | `glob` through `jest` | moderate | GHSA-3g3m-2fjr-6pw4 | Test runner only; it is not in the app bundle. |
| `mobile/` | `minimatch` through `expo`, `jest` | moderate | GHSA-g3hv-75wg-m249 | Build and test tooling over the project's own file names; not in the app bundle. |

## Go

`go tool govulncheck ./...` reports no vulnerability on the day of writing.

## Framework

| Finding | Accepted for | Why | Plan |
| --- | --- | --- | --- |
| CodeQL 33, `oauth/client.go:188`: fetching a Client ID Metadata Document | Until a framework change, under the decisions delegated by the author on 2026-11-22 | The safe client blocks private addresses at connect time, follows no redirect and never returns the body. What remains is a blind GET to a public https host | Keep |
| CodeQL 34, `oauth/authorize.go:265`: an error on `/authorize` redirects to a URI from an attacker's metadata document (SaaS only) | Until the fix ships in Vukem | Low: a phishing bounce carrying only `error`, `state` and `iss`, with no code or token | VF-119 in the framework |

## Performance

| Finding | Accepted for | Why | Plan |
| --- | --- | --- | --- |
| QA-503 (what is left): with 20 concurrent clients and 100000 transactions in one workspace the filtered transaction list (1541 ms p95) and the sync push (1576 ms p95) miss their 300 ms budgets (`perf-budgets.md`); dashboard, reports and export meet theirs, and every scenario meets its budget with one client | v0.7.0, under the decisions delegated by the author on 2026-11-22 | A text filter scans the date range and a push is written by the single SQLite writer; no summary answers either, and the budgets are not loosened | Full text search for the filter and a cheaper insert path for a push |

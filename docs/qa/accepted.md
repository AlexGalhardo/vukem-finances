# Accepted known issues

Findings the author accepted, with the reason. High and Critical findings are never listed here: they block CI. This list holds what the static and dependency checks (`golangci-lint`, `go tool govulncheck ./...`, `bun audit --audit-level=high`) report below that level, or a result that cannot be fixed in this repository.

Each row needs the author's approval before the campaign relies on it.

## Dependency advisories below High

| Where | Package | Severity | Advisory | Why it is accepted |
| --- | --- | --- | --- | --- |
| root | `katex` through `markdownlint-cli2` | low | GHSA-238p-pmpm-9mq7 | Dev tool only (the Markdown linter); it never ships and never sees untrusted input. |
| `mobile/` | `glob` through `jest` | moderate | GHSA-3g3m-2fjr-6pw4 | Test runner only; it is not in the app bundle. |
| `mobile/` | `minimatch` through `expo`, `jest` | moderate | GHSA-g3hv-75wg-m249 | Build and test tooling over the project's own file names; not in the app bundle. |

`site/` does not exist in this repository yet (epics 22 and 23). The CI job audits it as soon as `site/package.json` exists; add its findings here when it does.

## Go

`go tool govulncheck ./...` reports no vulnerability on the day of writing.

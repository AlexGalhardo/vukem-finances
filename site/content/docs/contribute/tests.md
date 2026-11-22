---
title: "Running the tests"
description: "The commands that check every part of vukem-finances before a hand-off: Go, JavaScript, browser, mobile, desktop, the documentation site and the docs checks."
---

Run every command that applies to what you changed before a hand-off. Install the JavaScript dependencies with `bun install --frozen-lockfile`.

## The application

```sh
go build ./...
go vet ./...
go test ./...
bun run typecheck
bun run lint:js
bun run format:check
bun run test:js
bun run lint:md
bun run build
```

`bun run test:js` runs the unit tests of `resources/js`. The browser specs in `e2e/` run with `bunx playwright test` against the real server, which builds the app and seeds a throwaway database.

## The other projects

- The mobile app is separate: `cd mobile && bun install --frozen-lockfile && bun run typecheck && bun run test`. Use Node 22.13 or newer.
- The desktop app is its own Go module: `cd desktop && go vet ./... && go test ./...`.

## Regenerating the contracts

The OpenAPI document, the TypeScript client, the operation inventory and the CLI and MCP references are generated, and a test fails when one is stale:

```sh
UPDATE_OPENAPI=1 go test ./tests/... -run 'TestOpenAPISnapshot|TestTypeScriptClientIsFresh'
go run ./cmd/app operations:inventory
go run ./cmd/app cli:reference
go run ./cmd/app mcp:reference
```

## This site

The site is its own project in `site/`. Run these there after `bun install --frozen-lockfile`:

| Command | What it does |
| --- | --- |
| `bun run build` | builds the static site, the sitemap, the Markdown twins and `llms.txt` |
| `bun run check:mdx` | checks the front matter and the structure of every page |
| `bun run check:claims` | fails when a page names a command, setting, route, tool, file or link that does not exist |
| `bun run check:built` | scans the built pages: no form, no other origin, one `h1`, no skipped heading level, titles and descriptions, twins |
| `bun run reference` | regenerates `content/generated` from the sources |
| `bun run reference:check` | regenerates and fails when the result differs from what is committed |
| `bun run last-updated` | writes the date of each page from the history |
| `bun run check:last-updated` | fails when a page changed after its date |
| `bun run test:site` | runs the tests of the generators and the checks |

When you change `docs/openapi.json`, `docs/reference/cli.md`, `docs/reference/mcp.md`, `.env.example`, `docs/shortcuts.md` or `CHANGELOG.md`, run `bun run reference` and commit the generated pages with it.

## Rules about tests

A flaky test is a defect: find the cause and never skip it, retry it or add a sleep. A failing check whose cause is a real bug is fixed in the product, not in the test.

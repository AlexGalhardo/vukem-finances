---
title: "Architecture"
description: "How the code is organized: the Go application, the operation registry behind the REST API, CLI and MCP, and the rules that never bend."
---

vukem-finances is built with the Vukem framework: Go on the server, React with Inertia and Tailwind in the browser, and SQLite by default. This page is a map; `AGENTS.md` in the repository is the guide that every contributor and every agent reads first.

## Layout

| Path | What lives there |
| --- | --- |
| `app/` | the Go application: accounts, billing, finance, workspaces, reports, export, jobs, mail and more |
| `app/operations/` | the operation registry: every action on the finances, declared once |
| `app/api/` | the REST adapter over the registry, with scoped tokens |
| `app/cli/` | the command line over the registry |
| `app/mcp/` | the MCP server over the registry |
| `app/sync/`, `app/changes/` | the sync protocol server and the change log |
| `app/ai/`, `app/telegram/` | the AI layer and the Telegram bot |
| `routes/` | the HTTP routes |
| `database/migrations/` | Go migrations |
| `resources/js/` | the React pages, components and email templates |
| `mobile/` | the React Native app, with its own `package.json` |
| `desktop/` | the desktop app, a separate Go module |
| `site/` | this static site, with its own `package.json` |
| `docs/` | the design brief, the specs and the reference sources |

## One registry, three adapters

Every action a person or an agent can take is an operation, defined once in `app/operations/` with its name, scope, kind (`read`, `write` or `destructive`), editions and typed input and output. The REST API, the command line and the MCP server are adapters over that registry. `Invoke` is the only way to run an operation, and it checks the edition, scope, role, input, idempotency key and audit entry before the operation runs. That is why the three surfaces behave the same.

A new operation gets a REST route, a command and a tool by itself. A test fails when an operation has no route on a surface it is offered on, unless it opts out with a written reason.

## Rules that never bend

- Money is an integer number of cents. Never a float.
- Every domain table carries a workspace id, and every query goes through the workspace scope. Isolation is tested.
- Synced tables carry a `uid`, a `version` and a soft delete, and join the change log registry.
- Roles are enforced on the server.
- The personal edition makes no outbound connection unless you turn an integration on.
- The design is original: nothing is copied from another product.

## Documentation is derived

The REST, CLI and MCP references, the environment variables and the changelog of this site are generated from `docs/openapi.json`, `docs/reference/cli.md`, `docs/reference/mcp.md`, `.env.example` and `CHANGELOG.md`. See [how to run the tests](/docs/contribute/tests/) for the checks that keep the pages true.

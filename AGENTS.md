# AGENTS.md

Guide for every agent that works in this repository. `CLAUDE.md` points here.

## What the project is

vukem-finances is a personal finance application built with [Vukem](https://vukem.dev): Go on the server, React with Inertia and Tailwind in the browser, SQLite by default. It has accounts, categories, transactions, monthly budgets, a dashboard and CSV import, on top of sign-up, workspaces with roles, and Stripe billing.

The app ships in two editions of one binary (decision D-003), chosen by `APP_EDITION` (`config.LoadEdition`; unset means `personal`, any other value besides `personal` and `saas` stops the boot):

| | `personal` | `saas` |
| --- | --- | --- |
| Workspaces | one implicit workspace, made by `/setup` | one per user plus created ones |
| Sign-up | closed: `/register` is 404; the first owner uses `/setup` with `SETUP_TOKEN` or the code in the boot log, the others use `/invites/{token}/register` | public `/register`, email verification always on |
| Billing, `/stripe/webhook`, `/admin/revenue` | not registered (404), no Stripe settings read | registered |
| Plan limits | none (`workspaces.Personal`) | Free and Pro |
| `GET /` | redirects to `/dashboard`, `/login` or `/setup` | landing page |
| Social login | find-only, never creates a user | creates users |

Ask the edition in one place: `workspaces.Service.Edition`, or `config.LoadEdition(app.Source)` where there is no service. Never read `APP_EDITION` in a handler. The finance test suite builds the app as `saas` (`newSaaSApp`); use `newPersonalApp` for the other edition. A sign-up address is never marked verified in `saas` (`AUTH_REQUIRE_EMAIL_VERIFICATION` is forced on, and a production boot refuses `false`).

## Module map

| Path | What lives there |
| --- | --- |
| `app/` | Go application code: `accounts/`, `admin/`, `billing/`, `controllers/`, `finance/` (money, transactions, dashboard, import), `jobs/`, `listeners/`, `mail/`, `models/`, `pages/`, `requests/`, `workspaces/` |
| `routes/` | HTTP routes (`web.go`, `auth.go`, `finance.go`, `admin.go`, ...) registered by `registry.go` |
| `database/migrations/` | Go migrations, registered in `registry.go` |
| `resources/js/` | React pages, components, layouts and React Email templates |
| `cmd/app/` | the binary entry point |
| `config/`, `bootstrap/` | configuration and application start-up |
| `.agents/rules/` | the rules of the team: read them before editing |
| `.github/` | CI workflows and `scopes.txt`, the list of commit scopes |
| `scripts/` | the release train script |

## Commands

```sh
go build ./...
go vet ./...
go test ./...
bun run typecheck
bun run lint:js
bun run lint:md
bun run build
```

Install the JavaScript dependencies with `bun install --frozen-lockfile`. Run every command above before a hand-off.

## Rules that never bend

- Money is `int64` cents (D-005). Amounts are stored in clear so SQL can sum and filter. Never use floats for money.
- Every domain table carries `workspace_id`, and every query goes through the workspace scope (D-004). A query that skips the scope is a bug.
- Commits follow Conventional Commits with a scope from `.github/scopes.txt`; see `.agents/rules/commits.md`.
- Branches and releases follow `.agents/rules/git-and-branches.md`: never commit to `main`, never push, never edit `CHANGELOG.md` by hand.
- Everything in the repository is in English.
- Never commit secrets. `.env` is ignored; `.env.example` holds placeholders only.

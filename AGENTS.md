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
| `resources/js/shortcuts/`, `resources/js/nav.ts`, `resources/js/i18n/` | the shortcut registry, the sidebar items, the translation files (see "The application shell") |
| `docs/` | `design.md` (brief and tokens), `shortcuts.md` (key allocation) |
| `cmd/app/` | the binary entry point |
| `config/`, `bootstrap/` | configuration and application start-up |
| `.agents/rules/` | the rules of the team: read them before editing |
| `.github/` | CI workflows and `scopes.txt`, the list of commit scopes |
| `scripts/` | the release train script and the Docker smoke tests (`docker-persist-test.sh`, `docker-clientip-test.sh`; they need Docker and are not part of `go test`) |
| `Dockerfile`, `compose.yaml`, `compose.prod.yaml`, `deploy/` | the image, the local stack, the VPS stack with Caddy and the [deployment guide](deploy/README.md) |

## Commands

```sh
go build ./...
go vet ./...
go test ./...
bun run typecheck
bun run lint:js
bun run format:check
bun run test:js
bunx playwright test
bun run lint:md
bun run build
```

Install the JavaScript dependencies with `bun install --frozen-lockfile`. Run every command above before a hand-off.

## The application shell

Read `docs/design.md` and `docs/shortcuts.md` before touching the UI. The design is original (D-011, D-012): never copy code, assets or layout from another product.

- **Add a sidebar item.** Append an entry to `resources/js/nav.ts` (`id`, `labelKey`, `href`, `icon`, optional `keys`, optional `edition: "saas" | "personal"`). The sidebar, the palette and a `go-<id>` action come from it. Billing is `edition: "saas"`: the personal edition must never show billing, workspace creation or workspace switching.
- **Register an action.** Call `useAction({ id, labelKey, group, keys, run, enabled })` from `resources/js/shortcuts/useAction.ts` in the component that owns the behavior. Never add a `keydown` listener. The action appears in the command palette and in the shortcut help by itself. Pick `keys` from the free ones in `docs/shortcuts.md` and add the row there; `registerAction` throws on a duplicate id or key, on a key a longer sequence starts with, and on a browser-reserved combination (`shortcuts/reserved.ts`).
- **Shortcuts never fire while the person types** (`shortcuts/guard.ts`). Mark a custom widget that takes text with `role="textbox"`, `combobox` or `searchbox`, or `data-shortcuts="off"`.
- **Add translations.** Add `resources/js/i18n/en/<domain>.ts`, `pt-BR/<domain>.ts` and `es/<domain>.ts` (the last two use `satisfies Record<keyof typeof en, string>`), then add one `typeof` line for the new domain to `resources/js/i18n/keys.ts`. `bun test resources/js/i18n` fails when a locale misses a key.
- **Page structure.** A signed-in page renders `PageHeader` for its one `h1`; the shell moves focus there after a navigation. Show money with `MoneyText` (sign or label, never color alone) and lists with `DataTable`.
- **Tests.** `bunx playwright test` runs the browser specs in `e2e/` against the real server (`e2e/prepare.mjs` builds the app and seeds a throwaway database; set `E2E_CHANNEL=chrome` to use the installed Chrome, otherwise run `bunx playwright install chromium` once). `bun run test:js` runs the unit tests (`bun test resources/js`; the path is required). `@types/bun` is not installed: `resources/js/bun-test.d.ts` holds the few ambient types the tests use; extend it when a test needs another matcher.

## Rules that never bend

- Money is `int64` cents (D-005). Amounts are stored in clear so SQL can sum and filter. Never use floats for money.
- Every domain table carries `workspace_id`, and every query goes through the workspace scope (D-004). A query that skips the scope is a bug.
- Isolation is tested, not trusted. A new domain model joins `models.TenantModels` (with `SetTenant` and `BeforeCreate`) and its table has an index that starts with `workspace_id`; a table with no `workspace_id` goes on the allow-list of `tests/isolation_guard_test.go`, with a reason. A new workspace route that takes an id needs a case in `tests/cross_tenant_matrix_test.go` (the foreign id answers 404, never 403 or 200). Both tests fail until it is done.
- Roles are enforced on the server: `owner`, `admin`, `member`, `viewer` (`app/workspaces/policy.go` holds the table of abilities). Every write handler starts with `h.can(c, ability)`; hiding a button is never the check. The workspace has exactly one owner: only `TransferOwnership` moves it, and every change of members runs under `lockWorkspace`.
- "Today" and "this month" come from the workspace timezone (`finance.Service.Today(ctx)`, `Month(ctx)`). Month arithmetic goes through `app/timeutil` (`AddMonths`); `AddDate` with a month argument is banned elsewhere (a test enforces it).
- Workspace routes are rate limited (`config/limits.go`: reads and writes per user, imports and invites per workspace). The Stripe and Resend webhooks are not limited by IP; their signatures are the check. Failed two-factor codes are limited per account as well as per IP.
- Sensitive workspace actions (roles, ownership, members, invites, settings) write `audit_events` through `Service.record`.
- Commits follow Conventional Commits with a scope from `.github/scopes.txt`; see `.agents/rules/commits.md`.
- Branches and releases follow `.agents/rules/git-and-branches.md`: never commit to `main`, never push, never edit `CHANGELOG.md` by hand.
- Everything in the repository is in English.
- Never commit secrets. `.env` is ignored; `.env.example` holds placeholders only.

## Docker and deployment

- The database is SQLite only, one file in WAL mode (`/app/storage/database.sqlite` in the `storage` volume), and the app is one replica. There is no PostgreSQL and no database container; do not add one.
- The reverse proxy and static server is Caddy, never nginx. Every `image:` and `FROM` carries an exact version tag and a digest, never `latest`. The Caddy image is pinned in `compose.prod.yaml`.
- `compose.yaml` is the local stack (app on `127.0.0.1:3000`, Mailpit, edition `personal` unless `APP_EDITION` says otherwise). `compose.prod.yaml` is the VPS stack: no published port on `app`, `TRUSTED_PROXIES` set to the compose subnet so rate limits see the real client address.
- The version is `config.Version`, set by the Dockerfile `ARG VERSION`; the `about` command (`bootstrap/about.go`) shows it and `/up` must not.
- Backup and restore use the framework's `db:backup`; never copy the live file. `user:reset-password -email <address>` recovers a locked-out owner from the server console.
- `.github/workflows/image.yml` publishes to GHCR and Docker Hub when a release is published; it needs the secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.
- On Windows with Git Bash, prefix `docker ... exec` commands that carry `/app/...` paths with `MSYS_NO_PATHCONV=1`.

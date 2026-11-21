# AGENTS.md

Guide for every agent that works in this repository. `CLAUDE.md` points here.

## What the project is

vukem-finances is a personal finance application built with [Vukem](https://vukem.dev): Go on the server, React with Inertia and Tailwind in the browser, SQLite by default. It has accounts, categories, transactions, monthly budgets, a dashboard and CSV import, on top of sign-up, workspaces with roles, and Stripe billing.

The app ships in two editions of one binary (decision D-003), chosen by `APP_EDITION` (`config.LoadEdition`; unset means `personal`, any other value besides `personal` and `saas` stops the boot):

| | `personal` | `saas` |
| --- | --- | --- |
| Workspaces | one implicit workspace, made by `/setup` | one per user plus created ones |
| Sign-up | closed: `/register` is 404; the first owner uses `/setup` with `SETUP_TOKEN` or the code in the boot log, the others use `/invites/{token}/register` | public `/register`, email verification always on |
| Billing, `/stripe/webhook`, `/abacatepay/webhook`, `/admin/revenue` | not registered (404), no Stripe or AbacatePay settings read | registered |
| Plan limits | none (`workspaces.Personal`) | Free and Pro |
| `GET /` | redirects to `/dashboard`, `/login` or `/setup` | landing page |
| Social login | find-only, never creates a user | creates users |

Ask the edition in one place: `workspaces.Service.Edition`, or `config.LoadEdition(app.Source)` where there is no service. Never read `APP_EDITION` in a handler. The finance test suite builds the app as `saas` (`newSaaSApp`); use `newPersonalApp` for the other edition. A sign-up address is never marked verified in `saas` (`AUTH_REQUIRE_EMAIL_VERIFICATION` is forced on, and a production boot refuses `false`).

## Module map

| Path | What lives there |
| --- | --- |
| `app/` | Go application code: `accounts/`, `admin/`, `billing/`, `changes/` (sync: entity registry, change-log triggers and reader, compaction, merge), `controllers/`, `finance/` (money, transactions, dashboard, import), `jobs/`, `listeners/`, `mail/`, `models/`, `pages/`, `requests/`, `workspaces/` |
| `app/ai/` | the AI layer (bring your own key): providers, the encrypted key vault, `Redact`, the call pipeline with limits (see "The AI layer") |
| `app/api/` | the public JSON API: scoped tokens, the middleware chain, the operations table and the handlers (see "The public API") |
| `routes/` | HTTP routes (`web.go`, `auth.go`, `finance.go`, `api.go`, `admin.go`, ...) registered by `registry.go` |
| `database/migrations/` | Go migrations, registered in `registry.go` |
| `resources/js/` | React pages, components, layouts and React Email templates |
| `resources/js/shortcuts/`, `resources/js/nav.ts`, `resources/js/i18n/` | the shortcut registry, the sidebar items, the translation files (see "The application shell") |
| `docs/` | `design.md` (brief and tokens), `shortcuts.md` (key allocation), `api.md` and `openapi.json` (the public API) |
| `clients/typescript/` | the generated TypeScript client of the API (`api.ts`) |
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

## The public API

Read `docs/api.md` first. The API is `/api/v1`, bearer tokens only (the session and CSRF middleware skip the prefix, so a cookie is ignored); it exists in both editions.

- A token (`vf_<id>.<secret>`, only a SHA-256 of the secret is stored, shown once on the Tokens page) is bound to one workspace through `access_tokens.resource` and to the person who made it. Scopes are a ceiling: every request intersects the scope with the person's role now (`workspaces.RoleCan`). No `*` scope exists, and a token with one is refused. Removing a member revokes their tokens of the workspace.
- Every operation is one row of `app/api/operations.go` (name, method, path, scope); `register` reads the row for the route, the scope check and the Idempotency-Key replay, and the OpenAPI document reads it for `x-required-scope` (D-017). To add an endpoint: add the row (and the scope in `scopes.go` if new), write a thin handler that calls a service of `app/finance`, register it in `API.Routes`, regenerate `docs/openapi.json` and `clients/typescript/api.ts` with `UPDATE_OPENAPI=1 go test ./tests/... -run 'TestOpenAPISnapshot|TestTypeScriptClientIsFresh'`, and add the case to `matrixCalls` (`tests/api_scope_matrix_test.go`) and, for a `{uid}` path, to `apiCrossTenantCases` in the cross-tenant matrix.
- A handler never holds a rule. Amounts become text for `finance.ParseAmount` through `centsText`; the uid of a row is turned into its id with `a.accountByUID` and the like (workspace scoped, a foreign uid is a 404); lists reuse `TransactionPage`. Money on the wire is an integer `*_cents`; ids on the wire are uids.
- Errors are problem+json with a stable `code` (`api.CodeOf`, set by the `ProblemDecorator` in `routes/api.go`). The failure order is 401, 404 (not a member), 429, 403, then 404 for a row of another workspace.
- The app serves only `/api/v1/openapi.json`; the reference is rendered by the static docs site (D-021). Never import the framework's `apidocs` package (`TestNoAPIDocsPage`).
- Rate limits are per token (120 reads and 60 writes a minute, `config/api.go`) and per address for failed lookups (60 a minute, refused before any lookup).

## The AI layer

Read `docs/ai.md` first. AI is bring your own key (D-007): off by default, enabled by the workspace owner (`ai.manage`, owner only), never paid by the service.

- Every feature calls `ai.Client.Call(ctx, workspaceID, ai.Request{Feature: ...})`: it checks enabled, feature, key, 20 calls per minute and the monthly limit, then records `ai_usage`. Never call a provider directly and never build an `http.Client` for one (`ai.NewHTTP`: safe client, 30 s, 1 MiB, no redirects, one retry).
- The key is an `ai.Key` (prints `[redacted]`), stored sealed with `ai-key:<workspace>:<credential>` as additional data. It must not appear in a response, page prop, flash, log, audit event or export. The form field is `apiSecret` so the framework never flashes it back. Provider error text is never passed on or logged: `ai.ProviderError` keeps only the status and a plain error type.
- Pass every free-text field through `ai.Redact` before it goes into a request, and send only what the owner agreed to (description, amount, date, category names).
- A custom base URL (`openai_compatible`) goes through `ai.CheckBaseURL` and the safe client. `AI_ALLOW_PRIVATE_HOSTS` works in the personal edition only.
- Tests use `app/ai/fake` or an `httptest` server; no test calls a real provider or needs a real key. The tables `ai_credentials`, `workspace_ai_settings` and `ai_usage` stay out of sync, export and API scopes (`TestAIExcludedFromExportAndAPI`); extend that test when those routes exist.
- To add a provider, see "Add a provider" in `docs/ai.md`.

## Billing

Read `docs/billing.md` first. Billing is saas only (D-003): the personal edition registers no billing route, webhook or page and reads no Stripe or AbacatePay setting.

- Access is derived: `workspaces.Service.Entitled` is the only place that decides, and `Limits` uses it. A Stripe subscription is Pro when it is `active` or `trialing` and `access_revoked_at` is empty; a PIX entitlement when it is not revoked and `valid_until` is in the future. Never store a plan flag.
- Access changes only from a verified webhook or from a re-read of the charge at the provider, never from a redirect or a client claim. Stripe: `/stripe/webhook` (signature and 5 minutes of tolerance from `contrib/stripe`). AbacatePay: `/abacatepay/webhook` (shared secret in constant time from a header or a path segment, never the query string; HMAC signature; 64 KiB cap), and the body only names the charge: `HandlePixEvent` re-reads it.
- Refunds and disputes revoke through `access_revoked_at`, `revoked_reason` and `revoked_ref`, which `SyncSubscription` never writes. The handlers read the object from the API and write with conditional `UPDATE`s, so a replayed or out-of-order event changes nothing twice.
- PIX activation is one conditional `UPDATE ... WHERE status IN ('pending','expired')` whose affected rows are checked, in the same transaction as the entitlement change. A charge becomes `expired` only when the provider says so. Months add with `timeutil.AddMonths` from `anchor_at`, so stacked payments do not drift.
- Amounts are `int64` cents from the server: `config.Pix.Amount(months)` (`PIX_PRICE_PRO_MONTHLY_CENTS` times the months, only the months in `PIX_MONTHS_OPTIONS`). The request names months and nothing else.
- Only the owner (`billing.manage`) starts a PIX charge, polls it or opens the portal. The poll is `GET /billing/pix/{ref}` (rate limited, 404 for another workspace's reference). The billing tables (`pix_payments`, `pix_entitlements`, `billing_events`) are server-only: no `uid` column, `workspace_id` with a leading index, members of `models.TenantModels`.
- Provider error text and secrets never reach a log, response or audit entry: wrap errors with `sanitizeStripe` or `providerError`, log references and kinds only. Payment records follow D-022: after a deletion only the amount, date, provider reference and a non-reversible subject reference are kept for 5 years (no legal review of that default).
- Settings: `STRIPE_*`, `ABACATEPAY_API_KEY`, `ABACATEPAY_WEBHOOK_SECRET`, `PIX_PRICE_PRO_MONTHLY_CENTS`, `PIX_MONTHS_OPTIONS` (`.env.example`). PIX is on when the API key is set; a production saas boot then requires the other three.
- The job `billing_reconcile` (every 6 hours) re-reads PIX charges that can still change. `ABACATEPAY_API_BASE` and `STRIPE_API_BASE` point at fakes: the tests (`fakeStripe`, `fakeAbacatePay` in `tests/`) and `bunx playwright test e2e/billing.spec.ts` (a saas server on port 3918 and `e2e/fake-abacatepay.mjs`); production ignores the AbacatePay one.

## Rules that never bend

- Money is `int64` cents (D-005). Amounts are stored in clear so SQL can sum and filter. Never use floats for money.
- Every domain table carries `workspace_id`, and every query goes through the workspace scope (D-004). A query that skips the scope is a bug.
- Isolation is tested, not trusted. A new domain model joins `models.TenantModels` (with `SetTenant` and `BeforeCreate`) and its table has an index that starts with `workspace_id`; a table with no `workspace_id` goes on the allow-list of `tests/isolation_guard_test.go`, with a reason. A new workspace route that takes an id needs a case in `tests/cross_tenant_matrix_test.go` (the foreign id answers 404, never 403 or 200). Both tests fail until it is done.
- Synced tables follow the [sync data model](docs/specs/sync-data-model.md). `accounts`, `categories`, `transactions` and `budgets` carry `uid` (ULID, the id on the wire), `version` (bumped by a database trigger, so re-read the row after a write) and `deleted_at` (soft delete: never `ForceDelete`, never a physical `DELETE`; only `changes.Compact` purges). A new synced table embeds `orm.SoftDeletes`, joins `changes.Register`, gets its triggers from `changes.InstallTriggers` in the migration (after any step that rebuilds the table, because a rebuild drops triggers; rebuild with `schema.DropTriggers()` and reinstall), and its unique indexes are partial (`WHERE deleted_at IS NULL`). Money fields stay within `models.MaxCents` (2^53 - 1): `models.CheckCents`. `TestIsolationGuard` fails for a table with `uid` that is not registered or lacks triggers.
- Accounts and transfers. A balance is derived (`finance.AccountBalances`, `NetWorth`; the dashboard total is `NetWorth`), never stored or edited. A transfer is two transactions that share `transfer_uid` (an expense leg on the source, an income leg on the destination, no category), written in one database transaction under the workspace lock by `CreateTransfer`, `UpdateTransfer` and `DeleteTransfer`. Transfer legs are excluded from income, expense, category spending, budgets and the monthly quota (`WhereNull("transfer_uid")`); deleting or restoring a leg or an account takes the other leg along; an ordinary edit of a leg is refused. Archived accounts (`archived_at`) leave the plan quota and the pickers but keep their history and balance. A new column on a synced table goes in `Fields` of its `changes.Register` entry or on the server-only list of `TestSyncFieldsCoverColumns`.
- Categories, tags and import. A category nests one level (`parent_id`, same kind as the parent), carries a `color` token name and an `icon` name from `models.CategoryColors` and `models.CategoryIcons` (never free hex), and can be archived (hidden from pickers, refuses new entries, keeps its history). Deleting one is `DeleteCategoryReassigning`: what used it moves to a category of the same kind in one database transaction, and transfer legs are never touched. Tags are tenant rows (`tags`, joined by `transaction_tags`), unique per workspace in any letter case, at most 10 per transaction and 200 per workspace; a transaction save names its tags and the service creates the missing ones. The join is synced as the `tag_uids` column of the transaction (`Entity.JSON`), which the service keeps equal to the join rows. A transfer leg takes no tags.
- Credit cards. A card is an account of kind `credit_card` (made only with its card: `CreateCard`, never `CreateAccount`, and never edited as a plain account) plus a synced `credit_cards` row (limit, `closing_day`, `due_day`, default payment account, `brand` from `finance.CardBrands`, `last4`; never a number, expiry or security code). A statement has no table: `transactions.statement_ref` is its closing date, set on write for purchases on a card account (create, edit, import) and on the card leg of a payment. Dates come from the pure `closingDate`, `statementFor` and `dueDate` (`app/finance/statements.go`). Status (`open`, `closed`, `paid`, `overdue`) is derived. `PayStatement` is a transfer from a non-card account; it is never an expense. Changing the closing day moves only unpaid statements, trashed rows included (`recomputeStatements`). Deleting an account takes its card along and clears the card default that named it; restoring brings the card back. Limit used is `max(0, -balance over all dates)`.
- Statement import reads untrusted files: `StatementParser` (`CSVParser`, `OFXParser`) bounds size (2 MiB), rows (20000, refused beyond, never cut), cells and columns, and a row it cannot read is a reported failure. The OFX scanner resolves no entity and refuses a DTD, CDATA or comment. Amounts go through `ParseAmount` and nothing else (never `ParseFloat`). `import_hash` (SHA-256 of account, date, signed amount, normalized description, or the FITID) skips rows the account already has unless the import allows duplicates; `RunImport` is one database transaction, so a failed import leaves nothing and a second run does nothing. An imported row is an ordinary entry, never a transfer leg. Text is stored as typed: every exporter must write text cells through `finance.SpreadsheetSafe`.
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

# vukem-finances

A personal finance application built with [Vukem](https://vukem.dev): Go on the server, React with Inertia and Tailwind in the browser, SQLite out of the box.

It has password reset and optional two-factor authentication, workspaces with roles and invitations, email through Resend with React Email templates, and the finance itself: accounts, categories, transactions, monthly budgets, a dashboard with charts and CSV import. One binary runs in two editions, chosen by `APP_EDITION`.

## Editions

| | `personal` (the default) | `saas` |
| --- | --- | --- |
| For | a person or a family on their own machine or server | the multi-tenant service |
| Workspaces | one implicit workspace | many, one per user plus created ones |
| Sign-up | closed; `/setup` creates the first owner, invite links add the others | public `/register` with email verification |
| Billing and limits | none, every limit is unlimited | Stripe, Free and Pro plans |
| Landing page | none: `/` redirects | yes |

Any other value of `APP_EDITION` stops the boot and names the two valid ones.

### Start the personal edition

1. Leave `APP_EDITION` unset or set it to `personal`, and run `vukem migrate` and `vukem serve`.
2. Open <http://localhost:3000>. With no user yet it goes to `/setup`, which asks for a setup secret: the value of `SETUP_TOKEN` or, when that is unset, the one-time code the server prints once in its boot log (`setup code: ...`; a restart before setup prints a new one).
3. Create the owner. `/setup` answers 404 from then on.
4. To add a family member, invite them from the Workspace page. With no mail server the page shows the invite link once, with its expiry: it is a secret, send it privately. The person registers at `/invites/<token>/register`.

### Start the SaaS edition

Set `APP_EDITION=saas`, the Stripe settings (see below) and a working mail setup, then run `vukem serve`. Sign-up is public at `/register`, and every account must verify its email: a self-asserted address is never treated as verified, and production refuses `AUTH_REQUIRE_EMAIL_VERIFICATION=false`.

## Run it

```sh
bun install        # once, unless vukem new already did
bun run build:emails
vukem migrate
vukem serve        # http://localhost:3000, with reload
go test ./...
```

Development writes emails to the log (`MAIL_MAILER=log`). In `saas`, register at <http://localhost:3000/register> and open the verification link from the log.

Set `ADMIN_EMAILS=you@example.com` in `.env` to open the admin panel at `/admin` (users, workspaces, subscriptions) and the revenue page at `/admin/revenue`.

## Plans

The plan limits apply to the `saas` edition; `personal` has none.

| | Free | Pro |
| --- | --- | --- |
| Accounts | 1 | unlimited |
| Transactions per month | 100 | unlimited |
| Members | 1 | unlimited |
| CSV import | no | yes |
| Budgets and alerts | no | yes |

The workspace is what pays, not the user: only its owner can start or manage a subscription. The first subscription of a workspace gets a 14-day trial. The limits live in `app/workspaces/limits.go`.

## Stripe

Only the `saas` edition reads these settings; `personal` registers no billing route.

1. Create a sandbox with `stripe sandbox create`, and a recurring price for Pro.
2. Put a restricted key (`rk_...`) in `STRIPE_KEY` and the price id in `STRIPE_PRICE_PRO`.
3. Run `stripe listen --forward-to localhost:3000/stripe/webhook` and put the `whsec_...` it prints in `STRIPE_WEBHOOK_SECRET`.

Access is granted by webhooks only, never by the page Checkout returns to. The webhook also sends the trial-ending, payment-failed and receipt emails.

## Email

Production sends with Resend: set `MAIL_MAILER=resend`, `MAIL_FROM` (a verified domain) and `RESEND_API_KEY`. In the Resend dashboard add a webhook for `email.bounced` and `email.complained` pointing at `https://your-app/webhooks/resend`, and put its signing secret in `RESEND_WEBHOOK_SECRET`: bounced and complaining addresses stop receiving mail.

The templates are React Email components in `resources/js/emails`. Run `bun run build:emails` after you change one: it writes `resources/emails`, which the app embeds and fills with Go values when it sends.

## PostgreSQL

SQLite is the default. For PostgreSQL install the driver and set `DATABASE_URL` as in the [database docs](https://vukem.dev/docs/database/overview). Money is stored as integer cents (`BIGINT`) and each workspace has an ISO 4217 currency, so no amount is ever a float. Dates are `YYYY-MM-DD` text, which sorts and compares the same on both databases.

## How it fits together

| Path | What |
| --- | --- |
| `app/workspaces` | workspaces, members, invites, plan limits and the role policies; `tenant.go` here and in `app/models` are the only code that calls the tenancy package |
| `app/finance` | accounts, transactions, budgets, dashboard, CSV import, monthly summary |
| `app/billing` | Stripe Checkout, Customer Portal and webhooks per workspace |
| `app/mail` | the mailables and the Resend webhook |
| `app/jobs` | the queued CSV import and the scheduled monthly summary |
| `app/controllers/ledger` | the page handlers |
| `routes/finance.go` | every route, and which middleware guards it |
| `resources/js` | the React pages, charts (plain SVG) and email components |

Every table of the finance app has a `workspace_id`, and every query goes through `workspaces.Scoped`, so a workspace never reads or writes another one's rows. `tests/finance_test.go` checks it.

## Documentation

The full guide is at <https://vukem.dev/docs/finance-saas>; everything else is at <https://vukem.dev/docs>.

## Empty dashboards

Dashboard chart collections are JSON arrays even before a workspace records its first transaction. The browser shows the empty spending state instead of treating missing data as a chart.

Workspace settings use the `workspaceDetails` page prop; the shared `workspace` prop always retains the current workspace and switcher data.

The six-month income and expense series retains every month's totals, including older months, independently of slice allocation.

## Project standards

| File | What it does |
| --- | --- |
| `.editorconfig` | tabs of width 4, spaces where the format needs them (YAML, Markdown, JavaScript) |
| `.gitattributes` | LF in the repository and in every working copy, Windows included |
| `.markdownlint-cli2.jsonc` | Markdown rules; run them with `npm run lint:md` |
| `CHANGELOG.md` | [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), generated from the commits by `git cliff --config .config/cliff.toml` |
| `.githooks/` | a `commit-msg` hook that enforces [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and a `pre-commit` hook that scans the staged changes for secrets (when [Betterleaks](https://github.com/betterleaks/betterleaks) is installed), then runs `gofmt` and `go vet` |
| `.betterleaks.toml` | secret scanning rules (`APP_KEY`, `vk_` API tokens, Stripe keys) on top of Betterleaks' defaults; scan the history with `npm run secrets:scan` |
| `compose.yaml` | the local stack: the app, PostgreSQL and a mail catcher (`docker compose up --build`) |

Releases use [Semantic Versioning](https://semver.org/spec/v2.0.0.html): tag
them `vMAJOR.MINOR.PATCH` and regenerate the changelog from the commits.

## License

MIT, see [LICENSE](LICENSE). Copyright (c) 2026-Present Alex Galhardo.

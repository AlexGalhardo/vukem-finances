# Landing page brief

The brief of the commercial landing page in `site/` (epic 22) and the register of every claim it makes. The page is static, in English, Portuguese and Spanish, and part of the site described in [site.md](site.md).

## Audience and promise

The audience is people and families who want to see where their money goes and would rather keep the data on their own computer or server. The promise: an open-source personal finance app you run yourself, whose data stays yours.

## Tone

Calm, plain and specific. Short sentences, no hype, no superlatives, no pressure. It says what the software does and what it does not do yet.

## Structure

One `h1` and landmarks, a skip link first. Hero with the calls to action, how it works in three steps, features grouped by what the person wants to do, privacy and ownership, editions (personal, and the SaaS edition with its Free and Pro limits), install, FAQ with `details`, and a footer with the repository, docs, privacy, terms (only when approved) and the language switch.

## Rules

1. Every factual sentence maps to a row of the register below whose evidence is a file of this repository. `bun run check:claims` in `site/` fails on a row without evidence, a missing path, a `where` key that is not in the English catalog, or a forbidden phrase in a catalog.
2. No hosted service is offered or implied: no sign-up, login, purchase, demo, contact form or price. The page says in plain words that there is no hosted service.
3. No invented numbers (users, savings, ratings), no logos, no quotes of people, no comparison with another product.
4. Plan limits come from `site/generated/plans.json`, never from the copy.
5. The design and the text are original: nothing is copied from another product (decisions D-011 and D-012).
6. Not claimed, because the evidence is not in this branch or the state is not final: the mobile app and its sync, an MCP server, store availability of any app, signed desktop builds, deletion of an account.

## Claims register

| claim | where it appears | evidence | how checked |
| --- | --- | --- | --- |
| The software is open source under the MIT license | hero.eyebrow, privacy.points, footer.license | `LICENSE` | file exists |
| It can be run with Docker, on a machine or a VPS | how.steps, install.docker | `Dockerfile`, `compose.prod.yaml`, `deploy/README.md` | files exist |
| Container images are published on GHCR and Docker Hub under the names shown | install.dockerText | `.github/workflows/image.yml` | names match `site/lib/site.ts` (test) |
| The personal edition has one workspace, closed registration and no plan limits | editions.personal | `docs/guide.md`, `app/workspaces/limits.go`, `config/edition.go` | files exist |
| The SaaS edition has billing, workspaces and plan limits | editions.saas | `docs/billing.md`, `app/workspaces/limits.go` | files exist |
| Free and Pro limits | editions.rows | `app/workspaces/limits.go`, `site/generated/plans.json` | the page reads the JSON; the JSON mirrors the Go values |
| There is no hosted service today | hero.note, editions.notice, faq.items | `docs/landing-brief.md` | decision D-020; forbidden phrases in the catalogs |
| The default database is SQLite | privacy.points, faq.items | `config/edition.go`, `AGENTS.md` | files exist |
| The site sets no cookie, runs no analytics, loads nothing from other sites | privacy.points | `site/astro.config.mjs` | static build, no integration that adds scripts |
| Desktop app for Windows, macOS and Linux, offline, data in the user folder | features.groups, how.steps, faq.items | `docs/desktop.md`, `.github/workflows/desktop.yml` | files exist |
| Desktop builds are unsigned | install.desktopText, faq.items | `docs/desktop.md` | files exist |
| Installers are attached to releases on GitHub | install.desktopText | `.github/workflows/desktop.yml` | files exist |
| Telegram bot, optional, off until configured, messages pass through Telegram | features.groups, privacy.points | `docs/telegram.md`, `app/telegram` | paths exist |
| AI with the user's own key, off by default, personal data redacted | features.groups, privacy.points, faq.items | `docs/ai.md`, `app/ai` | paths exist |
| CSV and OFX import | features.groups, how.steps | `app/finance/importer.go`, `app/finance/ofx.go` | files exist |
| Export of transactions as CSV and of the workspace as JSON | features.groups, faq.items | `docs/export-format.md`, `app/export/csv.go`, `app/export/json.go` | files exist |
| Roles owner, admin, member and viewer, joined by invitation | features.groups, faq.items | `app/workspaces/policy.go`, `app/workspaces/invite_register.go` | files exist |
| Quick entry, keyboard shortcuts and a command palette | features.groups | `app/finance/quickadd.go`, `docs/shortcuts.md` | files exist |
| Recurring entries and installments | features.groups | `app/finance/recurrence.go`, `app/finance/installments.go` | files exist |
| Dashboard | features.groups | `app/finance/dashboard.go`, `resources/js/pages/Dashboard.tsx` | files exist |
| Reports: cash flow, breakdowns, net worth, comparison | features.groups | `app/reports/cashflow.go`, `app/reports/breakdown.go`, `app/reports/networth.go`, `app/reports/compare.go` | files exist |
| Search and filters | features.groups | `app/finance/search.go` | file exists |
| Budgets | features.groups, how.steps | `app/finance/budgets.go` | file exists |
| Goals and alerts | features.groups, how.steps | `app/finance/goals.go`, `app/alerts/alerts.go` | files exist |
| Credit cards and statements | features.groups | `app/finance/cards.go`, `app/finance/card_statements.go` | files exist |
| Light and dark themes and three languages in the web app | features.groups | `docs/design.md`, `resources/js/i18n/keys.ts` | files exist |
| Command line and REST API with scoped tokens | features.groups | `docs/reference/cli.md`, `docs/api.md`, `app/cli/run.go` | files exist |
| The source is public | privacy.points | `README.md` | file exists |

## Out of the register

Final legal text (the author approves it) and the documentation content (epic 23).

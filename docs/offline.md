# Offline-first personal edition

Decision D-027: the desktop app and the web app in local Docker (`APP_EDITION=personal`, the default)
are standalone and work with no internet. Only the saas edition needs the network. Nothing in
personal boots, migrates, renders or runs a core feature through the network, and nothing leaves the
machine unless the person turned an integration on.

This page lists every outbound path of the app and of the framework features it enables, and what
each one does in personal. `app/offline` is the one place that applies the rules; the guard tests
below keep them true.

## What works offline

Everything a person does with their money: setup, sign-in, accounts, categories, tags, transactions,
transfers, cards, recurrences, budgets, goals, reports, CSV and JSON export, CSV and OFX import,
alerts shown in the app, the command line and the local MCP server (`mcp:serve --stdio`), the API
and the sync protocol between the app and a device on the same network, backups, and the Docker
stack. The page, its scripts, its styles and its fonts come from the binary: there is no CDN, remote
font, remote image or icon set, and the Content-Security-Policy is `default-src 'self'` with
`img-src 'self' data:` and `connect-src 'self'`, so a page could not load anything else.

## Inventory of outbound paths

"Off" means the code path is not registered or does nothing, and "explicit" means it runs only after
the person sets the integration up.

| Path | Where | What it does in personal |
| --- | --- | --- |
| Telegram Bot API (`api.telegram.org`) | `app/telegram` | Off until `TELEGRAM_BOT_TOKEN` is set. Then polling asks Telegram for updates (explicit). With no network the poll logs a warning and retries every few seconds; nothing else is affected. Webhook mode needs a public https address, which a local install does not have. |
| `t.me` link of the connect button | `/settings/telegram` | A link the person clicks. Never fetched by the server. |
| Framework log forwarding to Telegram | `vukem` `TELEGRAM_CHAT_ID` | Off unless both `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` are set; then error logs go to that chat (explicit). |
| AI providers (`api.openai.com`, `api.anthropic.com`, a custom base URL) | `app/ai` | Off until the owner saves a key and turns AI on (explicit). Every call goes through `ai.Client.Call`. With no network the call fails with `ErrUnreachable`: the page shows a message in the person's language and everything else keeps working. A model server on this computer or the local network (`AI_ALLOW_PRIVATE_HOSTS=true`) needs no internet. |
| Background AI jobs (`ai_categorize`, `ai_reseal`, `assistant_prune`) | `app/jobs` | Categorize calls the provider only for a workspace that enabled it; reseal and prune are local database work. |
| Mail | `app/mail`, `app/offline` | See "Mail" below. The image default is `MAIL_MAILER=log`: nothing is sent. |
| Resend webhook (`/webhooks/resend`) | `routes/finance.go` | Inbound only, and only a signed request from Resend changes anything. Nothing is fetched. |
| Google sign-in | `app/accounts/social.go`, `/auth/google/*` | Off until both `SOCIAL_GOOGLE_*` values are set (explicit). The button is a link to Google, so it needs the internet; the sign-in page says so. Password sign-in is always available. In personal it only signs in people who already have an account. |
| Stripe, AbacatePay (`api.abacatepay.com`), `/stripe/webhook`, `/abacatepay/webhook`, `/admin/revenue`, the billing pages | `app/billing`, `routes/finance.go` | saas only: not registered in personal, no setting read. |
| Job `billing_reconcile` (every 6 hours) | `app/jobs/billing.go` | Returns at once outside saas. |
| Job `privacy_purge` (daily), the deletion mails | `app/jobs/privacy.go`, `app/privacy/` | Local database work; no Stripe call outside saas. The mails of a deletion go through the best-effort transport, so a missing network never blocks a deletion. "Erase all local data" removes the database file and its WAL and SHM files and needs no network. |
| Remote MCP, OAuth metadata fetch (`/mcp`, `/authorize`, `/token`, client metadata documents) | `routes/mcp.go`, `app/oauth` | saas only: `buildMCP` returns nothing in personal, so no client metadata is ever fetched. |
| Analytics (`{{analytics}}` slot of `app.html`: Umami, Google Analytics, Search Console and Bing verification) | `resources/views/app.html`, framework `analytics` | Never rendered in personal, even when `UMAMI_*`, `GA_*` or the verification variables are set: the slot is guarded by the `networkTags` view function and the settings are cleared, so no tracker script, no verification route and no CSP allowance exist. saas keeps them. |
| Update check, telemetry, crash reporting | none | The app has none. `golang.org/x/telemetry` is only a dependency of the `govulncheck` dev tool and is not in the binary. The desktop app has no updater: a new version is a new installer. |
| Health check `/up` | framework | Answers from the process; it pings no host. |
| Outgoing webhooks, file upload scanning, Turnstile challenge, SSR server, IndexNow, Bing | framework packages | Linked into the binary but never configured by the app: none of them runs. |
| CLI remote mode (`FINANCES_TOKEN`, `auth login <server>`) | `app/cli` | Runs only when the person runs a command against a server they name. Local mode (the default) goes through the registry with no network. |
| `ai:eval`, `telegram:set-webhook`, `telegram:info` | `bootstrap` | Manual commands that exist to talk to a provider. Never scheduled. |
| Desktop "open external link" | `desktop/gate.go` | Opens `http`, `https` and `mailto` links in the system browser after a click. |
| Documentation links | docs and the README only | The app's screens carry no link to a remote page. |

### Scheduled jobs

`vukem serve` runs the schedule in process. In personal none of its tasks opens a connection by
itself: `monthly_summary`, `alerts_evaluate` (mail through the mailer, see below),
`recurrences_generate`, `changes_compact`, `sync_ops_prune`, `ai_reseal` and `assistant_prune` are
local; `billing_reconcile` returns at once; `telegram_prune` is local database work.

### Mail

Mail is the one feature that is both local-capable (a catcher such as Mailpit, or the log) and
network-capable (a real SMTP server, Resend, Mailtrap). The rule in personal, applied in
`offline.BestEffort` around whatever transport `MAIL_MAILER` chose:

- A failure to connect (DNS, a refused connection, an unreachable network, a timeout of 10 seconds)
  never fails the request that sent the mail. On the default inline queue (`QUEUE_CONNECTION=sync`)
  nobody can retry it, so it is logged (`mail not delivered: no connection to the mail server`, the
  kind of failure only, never an address) and dropped, and the answer is the same for everyone, so a
  flow that mails only some people cannot be used to tell who has an account. On a real queue
  (`QUEUE_CONNECTION=database` or `memory`) the error is returned to the worker instead: the job is
  retried with the queue's backoff and shows as failed, and the request had already answered when it ran.
- Any other failure (bad credentials, TLS, a refused recipient, a bad message) is returned as it is:
  it is a configuration error the person must see.

What each mailable does with no mail server and the default inline queue: the reset-password link,
the verification mail, the account-exists notice, the two-factor lock notice, workspace invitations,
the monthly summary and alert mails are all dropped with the warning above and the flow continues. A
person who cannot receive a reset link recovers with `user:reset-password -email <address>` on the
machine, and an invitation link is shown to the inviter in the app, so it can be shared by hand. saas
does not get this wrapper: its mail behavior is unchanged.

## Install time versus run time

Needed once, with a network: pulling the base images and building (`docker compose up --build`
pulls `oven/bun`, `golang` and `alpine`, runs `go mod download` and `bun install`), pulling Mailpit,
or downloading the desktop installer. After that nothing needs the network.

## Docker Compose with no network

1. With a network, once: `docker compose build` and, if you want the mail catcher,
   `docker compose pull mailpit`.
2. From then on `docker compose up -d` (no `--build`, no `--pull`) starts from the local images and
   needs no network. The app migrates and serves on `127.0.0.1:3000`; `/up` and `/setup` answer.
3. Mailpit is optional. It only catches mail. `docker compose up -d --no-deps app` starts the app
   without it; the mail the app then tries to send finds no server, is logged and dropped as above,
   and nothing else changes. Leave Mailpit out if you want no mail at all.
4. Compose reads `.env`, which `vukem new` wrote and which sets `APP_ENV=development`; a plain
   `docker run` of the image is `APP_ENV=production` and refuses `SMTP_INSECURE=true` for any host
   but loopback. The image defaults to `MAIL_MAILER=log`.

## How it was checked

- `go test ./tests -run 'TestOfflineGuard|TestPersonalMakes|TestPersonalDrops|TestPersonalMail'`.
  `TestPersonalMakesNoOutboundConnection` installs a guard on `http.DefaultTransport` (every
  `http.Client` that has no transport of its own) and on `net.DefaultResolver` (every connection to
  a host name), then boots personal, migrates, walks setup, sign-in, an account, a transaction, the
  main pages, reports and both exports, and runs every task of the real schedule once. Any
  non-loopback request or lookup fails the test. `TestOfflineGuardSeesOutboundConnections` proves the
  guard is not blind. It does not see a dial to an IP literal through a custom transport or a bare
  `net.Dialer`, nor anything outside Go; the next two checks cover those.
- `bunx playwright test e2e/offline.spec.ts --project=chromium` routes every request that is not to the
  app's own origin to an abort and fails if any is attempted, over the main pages, the command palette
  and the sign-in screens. The page CSP already refuses other origins before the network is asked,
  so the spec is a second line.
- A real container on a network with no route out (checked on 2026-11-22 with Docker 29.9.2 and
  Compose 5.6.1): the image built from this tree ran with `--network none` (the only interface is
  `lo`), migrated, started the server and the scheduler, answered `/up` with `{"status":"ok"}` and
  served `/setup`, while a request to `1.1.1.1` was "Network unreachable" and a lookup of
  `api.telegram.org` failed. Compose on an `internal: true` network with the app and Mailpit was
  healthy, the app reached `mailpit:8025`, no outside host could be reached, and `/up` kept answering
  with Mailpit stopped.

## Features that cannot work offline

AI features with a hosted provider, the Telegram bot, Google sign-in and the log forwarding to
Telegram need the internet by nature. They are off until turned on and fail with a clear message
without touching the rest. Mail to a real address needs a reachable mail server.

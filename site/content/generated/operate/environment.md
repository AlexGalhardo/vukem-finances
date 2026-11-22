---
title: "Environment variables"
description: "Every setting of .env.example with its default, the edition it belongs to and what it does."
---

> Generated from `.env.example`. Do not edit this page: change the source and run `bun run reference` in `site/`.

41 variables. Edition says where a variable matters: `personal`, `saas` or `both`. Copy `.env.example` to `.env` and set what you need; the example file holds placeholders only.

| Variable | Example value | Edition | What it does |
| --- | --- | --- | --- |
| `APP_NAME` | `vukem-finances` | both | No comment in the example file. |
| `APP_ENV` | `development` | both | No comment in the example file. |
| `APP_KEY` | empty | both | No comment in the example file. |
| `APP_PREVIOUS_KEYS` | empty | both | Former APP_KEY values, comma-separated: they still decrypt after a rotation. |
| `APP_DEBUG` | `true` | both | No comment in the example file. |
| `PORT` | `3000` | both | No comment in the example file. |
| `DATABASE_URL` | `file:storage/database.sqlite` | both | No comment in the example file. |
| `QUEUE_TABLE` | `queue_jobs` | both | Database queue (QUEUE_CONNECTION=database) tables, prefixed so they never collide with an app table such as "jobs". Set before the first boot. |
| `QUEUE_FAILED_TABLE` | `queue_failed_jobs` | both | No comment in the example file. |
| `ADMIN_EMAILS` | `admin@example.test` | both | No comment in the example file. |
| `APP_EDITION` | `personal` | both | The edition of the binary: personal (the default when unset) or saas; any other value stops the boot. personal: one implicit workspace, no billing, no public sign-up, no landing page, no plan limits; the first owner is created at /setup and the other people join by invite link. saas: public sign-up, many workspaces, Stripe billing and plan limits; set it explicitly in production. |
| `SETUP_TOKEN` | empty | personal | personal only: the secret that /setup asks for. Leave empty to use the one-time code the server prints in its boot log ("setup code: ..."). |
| `STRIPE_KEY` | empty | saas | Stripe (vukem install:billing), saas only: personal never reads these. Develop in a sandbox: stripe sandbox create. A restricted key (rk_...) with only the billing permissions. |
| `STRIPE_WEBHOOK_SECRET` | empty | saas | stripe listen --forward-to localhost:3000/stripe/webhook prints whsec_... |
| `STRIPE_PRICE_PRO` | empty | saas | Recurring price ids (price_...) of the plans in config/billing.go. |
| `STRIPE_CURRENCIES` | empty | saas | Currencies a checkout may ask for (e.g. brl,usd,eur); empty uses each price's default currency. Prices need currency_options for the others. |
| `ABACATEPAY_API_KEY` | empty | saas | One-off PIX payment with AbacatePay (saas only, optional): on when the API key is set, and then the other three are required in production. The key needs TRANSPARENT:CREATE and TRANSPARENT:READ; use an abc_dev_ key in Test Mode. |
| `ABACATEPAY_WEBHOOK_SECRET` | empty | saas | The shared secret of the webhook: register `https://<host>/abacatepay/webhook/<secret>` (a path segment, never a query string) or send it in X-Webhook-Secret. |
| `PIX_PRICE_PRO_MONTHLY_CENTS` | empty | saas | Price of one month of Pro in BRL cents (at least 100). The charge is this times the months. |
| `PIX_MONTHS_OPTIONS` | `1,3,12` | saas | Months the owner may buy, comma separated (default 1,3,12). |
| `TELEGRAM_BOT_TOKEN` | empty | both | Telegram bot (docs/telegram.md), optional: on when the token is set; with no token every bot route is 404. The token comes from @BotFather and is a secret. |
| `TELEGRAM_BOT_USERNAME` | empty | both | The bot name without the at sign (needed with a token): it builds the t.me link of the connect button. |
| `TELEGRAM_MODE` | empty | both | polling (default in personal: the server asks Telegram for updates) or webhook (default in saas). |
| `TELEGRAM_WEBHOOK_SECRET` | empty | both | webhook mode: 16 to 256 characters of A-Z a-z 0-9 _ -; Telegram echoes it in a header and the webhook path derives from it. |
| `TELEGRAM_WEBHOOK_URL` | empty | both | webhook mode: the public https address of the app, without a path; "telegram:set-webhook" registers it. |
| `MAIL_MAILER` | `log` | both | Mail. Development writes mails to the log; production sends them with Resend: MAIL_MAILER=resend, a verified sending domain in MAIL_FROM, an API key and the signing secret of the webhook that reports bounces and complaints (Resend dashboard, Webhooks: POST /webhooks/resend, email.bounced and email.complained). |
| `AUTH_REQUIRE_EMAIL_VERIFICATION` | `true` | both | saas: a sign-up is a self-asserted address, so verification is always on and production refuses any value but true (or unset). personal: accounts come from /setup and invite links and are created verified; the flag has no effect. |
| `MAIL_FROM` | `Finance <no-reply@example.com>` | both | No comment in the example file. |
| `RESEND_API_KEY` | empty | both | No comment in the example file. |
| `RESEND_WEBHOOK_SECRET` | empty | both | No comment in the example file. |
| `AI_ALLOW_PRIVATE_HOSTS` | `false` | personal | AI (bring your own key, docs/ai.md): the owner saves a provider key in the app, encrypted with APP_KEY. personal only: true lets the OpenAI-compatible provider reach a model server on this machine or a private network (http is then accepted for those hosts). saas ignores it. |
| `SOCIAL_GOOGLE_CLIENT_ID` | empty | both | Optional Google sign-in. Leave both empty to use password login only. Set both together, and register SOCIAL callback `http://localhost:3000/auth/google/callback`. |
| `SOCIAL_GOOGLE_CLIENT_SECRET` | empty | both | No comment in the example file. |
| `FINANCE_ADMIN_EMAIL` | `admin@example.test` | both | Local administrator created by vukem db:seed (development and test only). Leave the password empty to generate one into storage/finance-admin-\*.txt. |
| `FINANCE_ADMIN_PASSWORD` | empty | both | No comment in the example file. |
| `APP_DOMAIN` | empty | both | compose.prod.yaml (the VPS stack with Caddy): the public host name Caddy gets a certificate for, whose DNS record must already point at the server. Compose reads it from here or from the shell. See deploy/README.md. |
| `TOMBSTONE_RETENTION_DAYS` | `90` | both | Optional, commented out in the example. How many days a deleted row (a tombstone) and its change-log entries are kept before the nightly compaction purges them. 30 or more (the restore window). |
| `MCP_TOOLSETS` | `read` | both | Optional, commented out in the example. The MCP server (docs/mcp.md), personal edition on stdio: `app mcp:serve --stdio`. Groups of tools offered: read (the default), transactions, planning, all. |
| `MCP_ALLOW_DESTRUCTIVE` | `false` | both | Optional, commented out in the example. true offers the tools that remove or archive data; each call still asks you to confirm. |
| `MCP_OAUTH_CLIENTS` | `[{"id":"my-client","name":"My client","redirect_uris":["https://client.example/callback"]}]` | saas | Optional, commented out in the example. Remote MCP (saas only) needs APP_URL to be https. Clients known ahead of time, as a JSON list of public clients; any other client is identified by a metadata document. |
| `PAYMENT_RECORD_RETENTION_YEARS` | `5` | both | Optional, commented out in the example. Years the minimal payment record (amount, date, provider reference, a subject hash) of a deleted workspace is kept, 1 to 30 (decision D-022; the default has had no legal review). |

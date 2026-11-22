---
title: "Offline use"
description: "The personal edition, on the desktop and in local Docker, works with no internet. Only the SaaS edition needs the network."
---

The desktop app and the web app in local Docker run the personal edition (`APP_EDITION=personal`, the default). They are standalone applications and work with no internet. Only the SaaS edition needs the network. Nothing in the personal edition boots, migrates, renders or runs a core feature through the network, and nothing leaves the machine unless you turned an integration on.

## What works offline

Everything you do with your money: setup, sign-in, accounts, categories, tags, transactions, transfers, cards, recurrences, budgets, goals, reports, CSV and JSON export, CSV and OFX import, alerts shown in the app, the command line, the local MCP server, the REST API and the sync protocol between the app and a device on the same network, backups, and the Docker stack.

The pages, scripts, styles and fonts come from the binary. There is no CDN, remote font, remote image or icon set, and the Content-Security-Policy is `default-src 'self'`, with `img-src 'self' data:` and `connect-src 'self'`, so a page could not load anything else. The app has no analytics, update check, telemetry or crash reporting.

## What needs the internet

These are off until you set them up, and without a network they fail with a clear message while the rest of the app keeps working:

| Feature | Why | How to turn it on |
| --- | --- | --- |
| AI with a hosted provider | the provider is a remote service | the owner saves a key and turns AI on; see [AI with your own key](/docs/use/ai/) |
| Telegram bot | it talks to the Telegram Bot API | set `TELEGRAM_BOT_TOKEN`; see [the Telegram bot](/docs/use/telegram/) |
| Google sign-in | the button leads to Google | set both `SOCIAL_GOOGLE_*` values; password sign-in is always available |
| Mail to a real address | it needs a reachable mail server | set `MAIL_MAILER`; the default only logs |
| CLI against a remote server | it calls that server | run a command with `--server` |

A model server on your own computer or local network needs no internet: set `AI_ALLOW_PRIVATE_HOSTS=true` in the personal edition.

## What only the SaaS edition does

Billing, the Stripe and AbacatePay webhooks, remote MCP with OAuth, and analytics tags exist only in the SaaS edition. In the personal edition they are not registered and no setting for them is read.

## Docker with no network

With a network, once, build the images. From then on start the stack without building or pulling:

```sh
docker compose build
docker compose up -d
```

The app migrates and serves on `127.0.0.1:3000`. Mailpit only catches mail; `docker compose up -d --no-deps app` starts the app without it, and the mail the app then tries to send is logged and dropped.

The full inventory of outbound paths, with the tests that keep it true, is `docs/offline.md` in the repository.

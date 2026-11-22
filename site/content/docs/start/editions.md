---
title: "Editions"
description: "The personal edition is a standalone offline app; the SaaS edition is the same binary set up for many workspaces, billing and public sign-up."
---

One binary runs in two editions, chosen by the `APP_EDITION` setting. When it is unset, the edition is `personal`; any value other than `personal` or `saas` stops the boot.

## Personal edition

The personal edition is for one person, a family or a small team on their own machine. It is the default, and it is a standalone application:

- **Offline.** The desktop app and Docker on your own computer run with no internet. See [offline use](/docs/run/offline/).
- **One implicit workspace**, created by `/setup`. There is no billing and no plan limit.
- **No public sign-up.** The first owner uses `/setup` with `SETUP_TOKEN` or with the code printed in the boot log; other people join through an invite link.
- **Everything else is available**: the API, the command line, the local MCP server, the sync protocol, the Telegram bot and AI with your own key, the last two only when you turn them on.

## SaaS edition

The SaaS edition is the same code configured for many people:

- Public sign-up at `/register`, with email verification always on.
- One workspace per person plus the ones they create.
- Billing with Stripe and PIX, a Free and a Pro plan, and plan limits.
- A landing page at `/` and a remote MCP endpoint with OAuth.

No hosted service is currently offered. The SaaS edition is software that an operator runs on their own server, with its public address set by `APP_URL`; the [VPS guide](/docs/run/vps/) shows how. It is the only edition that needs the internet.

## Side by side

| | Personal | SaaS |
| --- | --- | --- |
| Workspaces | one implicit | one per person plus created ones |
| Sign-up | closed, invite links | public, email verification |
| Billing | not registered | Stripe and PIX |
| Plan limits | none | Free and Pro |
| Remote MCP | not offered | offered over HTTPS |
| Needs internet | no | yes |

## Which one to run

Choose personal unless you want to offer the app to people you do not know. The [environment variables](/docs/operate/environment/) page says which settings belong to which edition.

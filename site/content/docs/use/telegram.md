---
title: "Telegram bot"
description: "Record expenses with one short message, check balances and undo mistakes from a private Telegram chat, with the bot running inside your own installation."
---

A person can link a Telegram chat to their account, record an expense or income with one short message, check balances, the month and the latest entries, and undo a mistake, without opening the app. The bot lives inside the application and talks to the Telegram Bot API directly.

The bot is off until `TELEGRAM_BOT_TOKEN` is set. While it is off, every bot route is 404 and no worker starts. Telegram is an online service, so the bot needs the internet and is not part of the offline core; see [offline use](/docs/run/offline/).

## Privacy

Messages pass through Telegram's servers and are not end-to-end encrypted. The `/balance` command shows balances in the chat, so use it where you are comfortable with that. Balances are only sent when asked for. To stop, send `/unlink` or press Disconnect in Settings, Telegram. Only private chats are served: a group, supergroup or channel gets no reply and no data.

## Set it up

1. In Telegram, talk to the BotFather bot, send `/newbot` and follow the steps. It gives you a token and a bot name.
2. Set the variables below in `.env` or the container environment and restart the app.
3. In webhook mode run `telegram:set-webhook` once. In polling mode there is nothing else to do.
4. Run `telegram:info` to check the token: it prints the bot name and the mode, never the token.

| Variable | Meaning |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | the token from BotFather; a secret; empty means the bot is off |
| `TELEGRAM_BOT_USERNAME` | the bot name without the at sign, used for the connect link |
| `TELEGRAM_MODE` | `polling` (the personal default) or `webhook` (the SaaS default) |
| `TELEGRAM_WEBHOOK_SECRET` | webhook mode: 16 to 256 characters of letters, digits, `_` and `-` |
| `TELEGRAM_WEBHOOK_URL` | webhook mode: the public https address of the app, without a path |

## Link a chat

In Settings, Telegram, press Connect. The page shows a link and a code that work once, for 10 minutes. Open the link in Telegram and press Start. Linking proves that the person in the chat is the signed-in owner of the account; there is no link by email or password. A person can link at most 5 chats.

## Commands

| You send | What happens |
| --- | --- |
| `-12,50 lunch`, `12.50 lunch`, `+3000 salary` | a draft with the kind, amount, description, date and account, and the most recent categories to tap; nothing is saved until a tap |
| `1.234,56 rent yesterday` | a date word at the end sets the date: `today`, `yesterday`, `hoje`, `ontem`, `hoy`, `ayer`, `dd/mm`, `dd/mm/yyyy` or an ISO date |
| `/balance` | the balance of each account and the net worth |
| `/month` | income, expense and net of the current month |
| `/last` | the last 5 entries |
| `/budgets` | the progress of each budget this month |
| `/account` | choose the default account |
| `/workspace` | switch among your workspaces |
| `/lang` | English, Portuguese or Spanish |
| `/unlink` | disconnect this chat |

A tap on a category saves the entry once, even on a double tap. The reply has an Undo button that deletes the entry for 60 seconds. A draft is good for 15 minutes.

## Permissions

The bot reads your role at every message. It holds a fixed set of scopes (reading accounts, categories, transactions, budgets and the summary, and writing transactions), and a viewer reads but never records. It never changes members, billing, settings, tokens or AI keys.

More detail is in `docs/telegram.md` in the repository.

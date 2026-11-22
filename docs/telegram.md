# Telegram

A person can link a Telegram chat to their account, record an expense or an
income with one short message, check balances, the month and the last
entries, and undo a mistake, without opening the app. The bot lives inside the
application and talks to the Telegram Bot API directly with `net/http` (no
library, decision D-010). The code is in `app/telegram/`.

The bot is off until `TELEGRAM_BOT_TOKEN` is set. While it is off every bot
route (the webhook and the settings page) is 404 and no worker starts.

## Privacy notice

Messages pass through Telegram's servers and are not end-to-end encrypted. The
`/balance` command shows balances in the chat, so use it where you are
comfortable with that. Balances are only sent when asked for. To stop, send
`/unlink` or press Disconnect in Settings, Telegram. To remove what was
already said, delete the chat history in Telegram (the app cannot delete
messages in your chat). Only private chats are served: a group, a supergroup
or a channel gets no reply and no data.

## Setup

1. In Telegram, talk to [@BotFather](https://t.me/BotFather), send `/newbot`
   and follow the steps. It gives the bot token and the bot name.
2. Set the variables below (in `.env` or the container environment) and
   restart the app.
3. In webhook mode, run `telegram:set-webhook` once (it needs the public
   address). In polling mode there is nothing else to do.
4. Run `telegram:info` to check the token: it prints the bot name and the
   mode, and never the token.

| Variable | Meaning |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | The token from BotFather. A secret: it prints as `[redacted]` and is never logged. Empty means the bot is off. |
| `TELEGRAM_BOT_USERNAME` | The bot name without the at sign, used to build the `https://t.me/<bot>?start=<code>` link. |
| `TELEGRAM_MODE` | `polling` or `webhook`. The default is `polling` in the personal edition and `webhook` in saas. |
| `TELEGRAM_WEBHOOK_SECRET` | Webhook mode: 16 to 256 characters of `A-Z a-z 0-9 _ -`. Telegram echoes it in the header `X-Telegram-Bot-Api-Secret-Token`. |
| `TELEGRAM_WEBHOOK_URL` | Webhook mode: the public `https://` address of the app, without a path. Used by `telegram:set-webhook`. |
| `TELEGRAM_API_BASE` | Tests and development only: points the client at a fake Bot API. Ignored in production. |

## Modes

**Polling** (personal default). A worker inside the server process calls
`getUpdates` with a 30 second long poll, for `message` and `callback_query`
only. After each update it stores the next offset in `telegram_state`, so a
restart resumes at the first update not handled, with no loss and no
duplicate. Only the process that holds the lease row (`lease_until` in
`telegram_state`) polls: a second instance waits until the lease of the first
runs out (two minutes without renewal). SQLite means one instance anyway. The
worker removes the webhook when it starts, because Telegram refuses
`getUpdates` while one is set.

**Webhook** (saas default). `POST /telegram/webhook/{secret}`:

- The path segment is derived from `TELEGRAM_WEBHOOK_SECRET` (an HMAC, so the
  URL does not hold the secret) and is declared with `SecretParams`, so it
  stays out of the logs. A wrong segment is a 404.
- The header `X-Telegram-Bot-Api-Secret-Token` must equal the secret, compared
  in constant time (over SHA-256 of both values). Missing or wrong is a 401.
- The body is capped at 64 KiB (413) and decoded into typed fields, one JSON
  object (400 otherwise). Fields Telegram adds later are ignored.
- The route is outside the session and CSRF middleware (the header is its
  credential), so it works without a cookie or a CSRF token.
- A handled or repeated update is a 200. An internal failure is a 500, so
  Telegram delivers it again.

`telegram:set-webhook` registers the URL with `secret_token`. It prints the
host only, because the path is a secret.

## Linking a chat

Linking proves that the person at the chat is the signed-in owner of the
account:

1. In Settings, Telegram, press Connect. The app makes a code with 128 bits
   of randomness (22 characters of base64url, inside the 64 that `/start`
   accepts) and stores only its SHA-256, with the person and the workspace.
2. The page shows the deep link `https://t.me/<bot>?start=<code>` and the
   code once, with a countdown. It works for 10 minutes and once. Making a new
   code spends the earlier ones.
3. In Telegram the person opens the link and presses Start. The bot runs one
   conditional `UPDATE ... WHERE code_hash = ? AND used_at IS NULL AND
   expires_at > now` and links only when exactly one row changed. The link row
   is written in the same database transaction. Of 50 chats racing for one
   code, one links.

There is no link by account id, email or password, no form field takes a chat
id, and the web page names a link only by its own id (a revoke of another
person's link is a 404). A person holds at most 5 chats. A chat already linked
to someone else cannot be taken: it must send `/unlink` first.

## Authorization on every message

Nothing about access is remembered in the chat. Each message and each tap
resolves the link, then reads the membership of the person in the workspace now
and takes the role from it. A removed member or a viewer loses writing at the
next message, and a tap on a draft made before the change is refused too.

The bot holds a fixed set of scopes, the same names and the same role ceiling
as an API token: `accounts:read`, `categories:read`, `transactions:read`,
`transactions:write`, `budgets:read` and `summary:read`. A scope only counts
while the role of the membership holds the ability behind it, so a viewer reads
but never records. The bot never changes members, billing, settings, tokens or
AI keys. Its only delete is the undo of the entry it just saved.

## Commands

| Input | Result |
| --- | --- |
| `-12,50 lunch`, `12.50 lunch`, `+3000 salary` | A draft: kind, amount, description, date and account, with the five most recent categories of that kind, "No category" and "Cancel". Nothing is saved until a tap. |
| `1.234,56 rent yesterday` | The amount is read by `finance.ParseAmount` (123456 cents). A date word at the end sets the date: `today`, `yesterday`, `hoje`, `ontem`, `hoy`, `ayer`, `dd/mm`, `dd/mm/yyyy` or an ISO date. |
| `/balance` | The balance of each account and the net worth. |
| `/month` | Income, expense and net of the current month in the workspace timezone. |
| `/last` | The last 5 entries. |
| `/budgets` | The progress of each budget this month. |
| `/account` | Choose the default account. |
| `/workspace` | Switch among the workspaces you belong to. |
| `/lang` | English, Portuguese (Brazil) or Spanish. |
| `/unlink` | Disconnect this chat. |
| `/help`, anything else | The help text, never an error trace. |

A message is read as an entry when it starts with an optional `+` (income) or
`-` (expense, the default) and an amount, then the description, then an optional
date word. Input is capped at 500 characters. A zero, negative or absurd amount
(beyond 2^53 - 1 cents), a date that does not exist and a message with no
amount are refused with a short explanation.

A tap on a category consumes the draft atomically (a conditional update, so a
double tap or two instances make one entry) and creates the transaction through
the normal service path, with the draft reference as its uid. The reply shows
the saved entry with an Undo button that soft-deletes it for 60 seconds. A draft
is good for 15 minutes.

## Idempotency and limits

`telegram_updates` holds every processed `update_id` under a unique index: handling an update
begins by inserting it, and proceeds only when a row was inserted, so webhook
retries and poll overlaps do nothing twice. A failure of the server itself
releases the claim so the redelivery is handled. Rows older than 7 days are
pruned by the daily `telegram_prune` job, which also drops expired codes and
drafts and finished rate windows.

Rate limits, per chat, counted in the database (the table
`telegram_counters`), so they survive a restart and do not depend on
`RATE_LIMIT_STORE`:

- 20 messages a minute and 200 a day;
- 5 `/start` link attempts a minute;
- 60 button taps a minute.

An excess message gets one polite reply a minute and is otherwise dropped. A
failing counter store drops the message (the limit fails closed).

## Alerts

With the switch in Settings, Telegram (or from the Notifications page), the
alerts of the workspace (budgets, cards, goals, failed recurring entries) are
also sent to the person's linked private chats. It is off until turned on and
is a column of `notification_preferences`. The channel is `telegram.Channel`, an
`alerts.Channel`: a failure is logged and never fails the alert job, the claim
is released and the next run tries again for 48 hours, and the same alert is
never sent twice.

## Threat notes

| Finding of the old app | Here | Test |
| --- | --- | --- |
| Linking a chat by account id, or without proof of ownership (critical) | Only a single-use code made for the signed-in person links | `TestSecurity_NoLinkByAccountID`, `TestSecurity_LinkCodeSingleUse` |
| An endpoint that let a user set any chat id (critical) | No route takes a chat id; revoke is by own link id | `TestNoRouteTakesAChatID`, `TestRevokeOnlyOwnLinks`, `TestTelegramSettingsAreOwnOnly` |
| Password login inside the chat bypassing the HTTP rate limiter | There is no login in the chat, and every entry point is limited | `TestSecurity_TelegramRateLimit`, `TestSecurity_TelegramStartRateLimit`, `TestSecurity_TelegramCallbackRateLimit` |
| Find-then-delete of single-use link tokens (race) | One conditional `UPDATE` and the link in one transaction | `TestSecurity_LinkConcurrent`, `TestSecurity_LinkConcurrentSameChat` |
| User text interpolated into Markdown messages | HTML mode, every user or database string escaped | `TestUserTextIsEscaped`, `TestEscapeHTML` |
| Unbounded inputs | 500 characters, 64 KiB webhook body, 64 byte callback data, bounded lists | `TestParseLimits`, `TestSecurity_TelegramWebhook` |
| Bot state only in memory | Offset, lease, drafts, update ids, counters and links are in the database | `TestPollerResumesFromOffset`, `TestRateLimitSurvivesRestart` |

More guarantees and their tests:

- The webhook secret and the path segment are compared in constant time:
  `TestSecurity_TelegramWebhook`.
- A removed member, a viewer and a demoted member lose access at once:
  `TestSecurity_RemovedMemberLosesBot`, `TestViewerCannotRecord`.
- The bot cannot delete anything but the entry it just saved:
  `TestBotCannotDeleteOthers`.
- Group chats are ignored in silence: `TestSecurity_GroupChatIgnored`.
- The token, the secrets, a link code and the text of a message are in no log
  line and no response: `TestSecurity_TokenNeverInLogsOrResponses`,
  `TestTokenNeverInErrors`. The client prints as `[redacted]`, errors carry the
  method, the status and Telegram's description with the token scrubbed, and
  the transport error drops the URL (the token is in the path of every call).
- A leaked bot token: it lives only in the environment. Rotate it with
  `/revoke` in BotFather, set the new value and restart.
- Telegram being down, or a user blocking the bot, never fails an action that
  already happened: the reply is logged as failed (without its text) and the
  update stays handled.

All the tests run against a fake Bot API (`tests/fakes/telegram.go`); none
calls the real API or needs a real token.

## Known limits

- An update claimed and then lost to a crash between the claim and the handling
  is not handled again (at most once for that update). The offset is stored per
  update, so a restart between two updates loses nothing.
- Polling and SQLite mean a single instance. With several instances use
  webhook mode, where the update id claim keeps handling to once.
- Telegram is not end-to-end encrypted (see the privacy notice).

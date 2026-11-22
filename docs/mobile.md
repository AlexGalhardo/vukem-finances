# The mobile app

`mobile/` is the React Native app of vukem-finances (Expo, TypeScript). It works with no account and no server: everything lives in a SQLite database on the phone, follows the same rules as the server, and syncs both ways with a personal server or the SaaS when the person connects it. Sync is optional: until the person pairs the phone, and again after they disconnect, the app makes no network request at all.

## Setup

The app is its own project, not a workspace of the root: its `package.json` and `bun.lock` are in `mobile/`, and the web Docker build ignores the folder (`.dockerignore`).

```sh
cd mobile
bun install --frozen-lockfile
bun run typecheck
bun run test
```

- Bun is the package manager. Every dependency is pinned to an exact stable version (Expo SDK 58); upgrading the SDK is a separate change.
- The tests run on Node. The repository tests use Node's built-in SQLite (`node:sqlite`), which needs Node 22.13 or newer; CI pins Node 24.21.0.
- `bun install` may report one blocked postinstall (`@parcel/watcher`). Nothing here needs it: the tests, the typecheck and `expo export` all pass without it.
- Lint and format come from the root Biome config (`bun run lint:js`, `bun run format:check` at the repository root), which covers `mobile/src`, `mobile/app` and `mobile/__tests__`.

## Running it

```sh
cd mobile
bunx expo start
```

Open it in an emulator or on a phone with Expo Go or a development build. To check that both platform bundles build without a device:

```sh
bunx expo export --platform android --output-dir "$TEMP/vf-android"
bunx expo export --platform ios --output-dir "$TEMP/vf-ios"
```

Routes are in `mobile/app` (expo-router). Do not create `mobile/src/app`: expo-router reads it as the routes root when it exists (`mobile/__tests__/routes.test.ts` guards this).

## Layout

| Path | What lives there |
| --- | --- |
| `mobile/app/` | The routes: the five tabs (Home, Transactions, Add, Accounts, More) and one route per screen the More list opens |
| `mobile/src/db/` | The `Database` interface, the expo-sqlite adapter, the migrations, the entities and the repositories with the outbox |
| `mobile/src/domain/` | The rules, ported from the server: money, ids, dates, balances and transfers, categories, budgets, reports, recurrence and installments, card statements, cards |
| `mobile/src/core/` | The app state, the settings, the start-up (`Bootstrap`) |
| `mobile/src/screens/`, `mobile/src/components/` | The screens and the shared components |
| `mobile/src/i18n/` | The catalogs: `en` is the source, `pt-BR` and `es` must hold the same keys and placeholders |
| `mobile/src/theme/tokens.ts` | The palette and sizes, generated from `resources/css/tokens.css` |
| `mobile/src/backup/`, `mobile/src/lock/` | The local backup and the app lock |
| `mobile/src/sync/` | The sync client: transport (the only file that uses `fetch`), outbox, pull apply, push results, integrity pass, resync, merge choice, connect, engine and controller |
| `mobile/__tests__/` | The tests: Jest with the `jest-expo` preset and Testing Library, querying by role and label |

Regenerate the tokens after changing the CSS with `bun run scripts/export-tokens.ts` at the root; a test and a CI step fail when the generated file is stale.

## Shared vectors

The money parser, the recurrence engine and the card statement rules exist twice, in Go and in TypeScript, and both are tested against the same files in `testdata/`:

- `amount-vectors.json`: `finance.ParseAmount` and `parseAmount`.
- `recurrence-vectors.json`: the recurrence engine (`docs/specs/recurrence.md`).
- `statement-vectors.json`: closing date, statement, due date and status. It is written by the Go test `TestStatementVectorsExport`; rewrite it with `UPDATE_STATEMENT_VECTORS=1 go test ./app/finance/ -run TestStatementVectorsExport`.

Money is an integer number of cents everywhere (never above 9007199254740991), and the amount parser is the only way text becomes money.

## Data model

One table per synced entity (`accounts`, `categories`, `tags`, `transactions`, `budgets`, `credit_cards`, `recurrences`), keyed by `uid` (a ULID the app makes itself), with the field names of the sync protocol (`docs/specs/sync-data-model.md`): references are `<entity>_uid`, the tags of a transaction are `tag_uids`. Every row also has `version`, `server_seq` (empty until synced), `deleted_at`, `dirty` and `updated_local_at`. There are no foreign keys, because the protocol delivers references out of order; `checkIntegrity` checks them after writes.

Other tables: `outbox` (the queued mutations), `meta` (settings and the schema version) and `transaction_tags`. Migrations are numbered, applied in order inside a transaction, and tested from every earlier version.

Every write goes through a repository, which in one SQLite transaction changes the row, bumps `version`, marks it `dirty` and appends the mutation to the outbox with only the changed fields (`upsert`, `delete` or `restore`, with `base_seq`). With no server the queue just keeps the history. It is soft-capped at 5000 entries: past that the app asks before dropping the oldest, and Settings can turn the queue off after a confirmation.

Generated recurrence occurrences use the natural key `(recurrence_uid, occurrence_on)`, so a deleted occurrence is never made again. Dates are ISO strings, and the workspace time zone decides what today is.

## Sync

The client of `docs/specs/sync-protocol.md`. It is in `mobile/src/sync/` and depends only on a `Database`, a `Transport`, a `Clock` and a `Random`, so every behavior is tested without a device or a network. `transport.ts` is the only place that uses `fetch`, and a test says so.

### Pairing

On the web, open Devices and press Add device: the page shows an 8 character code (valid once, for 10 minutes) and the server address. On the phone open More, Sync, Connect, type the address, the code and a name for the phone. The server answers with a token for that workspace (only the `sync` scope, 365 days, listed on the Tokens page as "Phone" and the device name) and registers the device. The token is kept only in the secure store of the phone (`expo-secure-store`); it is never in SQLite, never in a backup and never in a log or an error message. The code is cleared from the screen after use. Revoking the device or the token on the web stops the phone at once; it keeps its data and offers Pair again or Disconnect. When the owner deletes the workspace, sync answers `410 workspace_deleted` (during the 30 day grace period and after the purge): the phone stops for good, says so on the Sync screen and keeps its data until the person presses "Erase this workspace from the phone", which erases the local rows and the queue and disconnects. A phone that was offline for the whole grace period gets the same answer at its next sync.

### Server address rules

`https` always. `http` only for `localhost`, private network addresses (10.x, 172.16 to 31.x, 192.168.x, 169.254.x, `fc00::/7`, `fe80::/10`) and `.local` names, after a confirmation screen that says the traffic is not encrypted. `http` to a public host is refused. For a server on your own network, prefer the Caddy HTTPS setup of the deployment docs.

### What a run does

Push (up to 200 queued changes in creation order, each with a fixed `op_id`, so a retry after a lost response never writes twice), pull from the saved cursor page by page (each page and its cursor in one SQLite transaction), then an integrity pass. It runs on "Sync now", when the app comes to the foreground, 3 seconds after a local write and every 5 minutes while the app is open. A failure backs off 5 s, 15 s, 60 s, then 5 minutes, with jitter; there is no network-state package, so "offline" means a request failed. A `410 resync_required` downloads a snapshot into staging tables and replaces every row that has no pending change.

### Conflicts and problems

The rules are those of `docs/specs/sync-data-model.md`: a pending local edit survives a pull of the same row (the server wins only on fields that have no pending change), a delete wins over a concurrent edit (the edit is dropped and recorded), the same field changed on both sides goes to the later arrival and the loser is recorded. Sync, Review conflicts shows each lost value with both sides and lets the person apply the replaced value again as a normal edit, or dismiss it. A mutation the server refuses for good (a validation error, `forbidden`, and so on) is removed from the queue and listed under Review problems with Try again and Discard; a discard of a row the server never had removes it, and for any other row the next run downloads a fresh snapshot. A reference that still names nothing after a sync is listed in the data check and never deleted by the app. When the plan quota refuses a create, only that entity pauses and the status screen shows the server's message.

### First connection

A phone with no data connects without a prompt. A phone with data asks: Merge (recommended: keep both sides; an account, category or tag with the same name and kind as a server row takes the server's uid, every reference follows, and the screen says how many were merged), Replace this phone's data with the server's (after a confirmation that says local data will be deleted, with an export offered first) or Cancel. Disconnect keeps the data and, at the person's choice, the waiting changes.

### Limits

- Sync runs only while the app is open: there is no background sync.
- One workspace per phone.
- Goals, notifications and AI data are not in the app and are not synced.
- No end-to-end encryption; no QR scanning of the pairing code.
- A push is rate limited by the server (30 a minute per device); a pull 60 a minute.

### Tests

`bunx jest` runs the unit tests, the shared scenarios of `testdata/sync/` through two client engines over a reference model of the server, and a 50 seed convergence run on that model. `bash scripts/mobile-sync-e2e.sh` runs the same scenarios an owner of a Pro workspace can run, a lost-response retry and 50 seeded convergence runs against the real Go server over HTTP (it needs Go, Node 22.13 or newer, Bun and `cd mobile && bun install --frozen-lockfile`). The Free plan and viewer scenarios run in the Go suite (`tests/sync_scenarios_test.go`).

## The app lock and its limits

The lock is optional (Settings). When on, the app asks for the biometrics or the device passcode when it opens and after 60 seconds in the background, and shows no data while locked.

It is an access gate, not disk encryption. The SQLite database is not encrypted: someone with the unlocked phone or a copy of the app data can read it. The setting itself lives in `expo-secure-store`, never in the database or in plain storage. Disk encryption of the database is out of scope for this version.

## Backup

Settings, Backup, shares a JSON file of the whole local database through the system share sheet. The format is the workspace export of `docs/export-format.md` (`docs/export-format.schema.json`); a test validates it against the schema. The device has no members and no goals yet, so those arrays are empty.

## Accessibility

Every touchable has a role and a label and is at least 48 by 48 dp; layouts use flex and no fixed sizes on anything that holds text, so text can grow to 200 percent of the system size; income and expense carry a sign and a spoken label ("Expense minus 12 dollars 50 cents"), never only a color; every action is a visible button (there is no swipe-only action); nothing animates, so reduced motion is respected. `mobile/__tests__/a11y` audits every screen for these rules.

## Manual checklist (not automated)

Run on a device or emulator before a release. None of this is covered by the unit tests.

- [ ] First run on a clean install: language, currency, time zone, a first account and the default categories.
- [ ] Airplane mode: record income, expense and a transfer; check Home, Budgets and Reports.
- [ ] Turn the app lock on, send the app to the background for more than 60 seconds, return: the lock appears and unlocks with biometrics and with the device passcode. A device with no screen lock refuses to turn it on.
- [ ] Settings, Backup: the share sheet opens and the file imports into a JSON viewer.
- [ ] TalkBack (Android) and VoiceOver (iOS): every control is read with a role and a name; amounts are read in words; reading order follows the screen.
- [ ] System font size at 200 percent (Android) and the largest Dynamic Type size (iOS): no text is cut off on any screen.
- [ ] Reduce motion on: nothing moves.
- [ ] The app opens after the phone restarts and after a few hours in the background, with the data intact.
- [ ] Pairing: a code from the web Devices page connects the phone over https; a wrong or reused code shows an error; the token appears on the web Tokens page and the phone shows the workspace name.
- [ ] Plain http to a LAN address shows the not-encrypted confirmation; http to a public host is refused.
- [ ] Airplane mode: edit on the phone and on the web, turn the network back on, press Sync now: both show the same data and nothing is lost or doubled.
- [ ] Kill the app in the middle of a sync (push, then pull): reopen it and sync again, with no duplicate and no loss.
- [ ] Two phones on one workspace: edit the same transaction on both while offline and sync both; the conflict list shows the lost value and Apply the replaced value again works.
- [ ] Revoke the device on the web: the phone shows Sign-in needed, keeps its data and offers Pair again and Disconnect.
- [ ] Delete the workspace on the web: the phone shows Workspace deleted, keeps its data until Erase this workspace from the phone is pressed, then is disconnected and empty.
- [ ] First connection of a phone that has data: Merge reports the merged count; Replace shows the warning and offers the export first.
- [ ] The status line is read by TalkBack and VoiceOver when it changes ("Synced", "Offline, 3 changes waiting", "Sign-in needed").
- [ ] A device where Hermes lacks `Intl` data for the chosen time zone or currency still shows dates and amounts.

## Not in this version

Background sync, goals, notifications, AI, the Telegram bot and billing in the app, store publication and signing, push notifications, widgets, tablet layouts and disk encryption of the database.

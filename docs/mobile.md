# The mobile app

`mobile/` is the React Native app of vukem-finances (Expo, TypeScript). It works with no account and no server: everything lives in a SQLite database on the phone, follows the same rules as the server, and is shaped so that the two-way sync (a later epic) needs no rewrite. In this version the app makes no network request at all.

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
- [ ] A device where Hermes lacks `Intl` data for the chosen time zone or currency still shows dates and amounts.

## Not in this version

Sync with a server, goals, notifications, AI, the Telegram bot and billing in the app, store publication and signing, push notifications, widgets, tablet layouts and disk encryption of the database.

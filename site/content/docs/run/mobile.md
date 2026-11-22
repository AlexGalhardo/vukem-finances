---
title: "Mobile app"
description: "The React Native app keeps your data in a SQLite database on the phone, works with no account and no server, and syncs only after you pair it."
---

`mobile/` is the React Native app of vukem-finances, built with Expo and TypeScript. It works with no account and no server: everything lives in a SQLite database on the phone and follows the same rules as the server. Sync is optional. Until you pair the phone, and again after you disconnect, the app makes no network request at all.

The app is built from the repository. It is not published in an app store yet.

## Run it from the repository

The app is its own project, not a workspace of the root, with its own `package.json` and `bun.lock`:

```sh
cd mobile
bun install --frozen-lockfile
bunx expo start
```

Open it in an emulator or on a phone with Expo Go or a development build. The unit tests run on Node 22.13 or newer (the repository tests use `node:sqlite`):

```sh
cd mobile
bun run typecheck
bun run test
```

## What is in it

Five tabs: Home, Transactions, Add, Accounts and More. It has accounts, categories, tags, transactions, transfers, budgets, reports, recurrences and installments, and credit cards, with the same money, date and balance rules as the server.

## Privacy and lock

An optional app lock asks for biometrics or the device passcode when the app opens and after 60 seconds in the background. It is an access gate, not disk encryption: the database itself is not encrypted. The pairing token is kept only in the secure store of the phone, never in the database, a backup or a log.

## Backup

Settings, Backup shares a JSON file of the whole local database through the system share sheet. The format is the workspace export described in [the export format](/docs/integrate/export-format/).

## Not in this version

Background sync, goals, notifications, AI, the Telegram bot and billing inside the app, store publication and signing, push notifications, widgets, tablet layouts and disk encryption of the database.

To keep the phone and a server in step, read [mobile sync and pairing](/docs/use/mobile-sync/).

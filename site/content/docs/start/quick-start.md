---
title: "Quick start"
description: "Get vukem-finances running on your own computer in a few minutes, with the desktop app or with Docker, and create your first account."
---

Pick one way. Both run the personal edition and work offline.

## Option 1: the desktop app

1. Download the installer for your system from [GitHub Releases](https://github.com/AlexGalhardo/vukem-finances/releases).
2. Install and open it. The installers are not code-signed, so your system shows a warning; [the desktop guide](/docs/run/desktop/) says how to continue.
3. On the first run the app shows `/setup`. Create the owner account.

## Option 2: Docker on localhost

You need Docker with the Compose plugin and a copy of the repository.

```sh
git clone https://github.com/AlexGalhardo/vukem-finances.git
cd vukem-finances
cp .env.example .env
```

Set `APP_KEY` in `.env` (for example with `openssl rand -base64 32`), then start the stack:

```sh
docker compose up -d --build --wait
docker compose logs app | grep "setup code"
```

Open `http://localhost:3000/setup`, enter the code from the log (or the `SETUP_TOKEN` you set), and create the owner. Mail the app sends appears in the local inbox at `http://localhost:8025`. The [Docker guide](/docs/run/docker/) has the details.

## Your first ten minutes

1. Create an account, for example a checking account with its opening balance.
2. Record a few transactions, or import a CSV or OFX statement; see [transactions and import](/docs/use/transactions-import/).
3. Set a monthly budget for a category; see [budgets and goals](/docs/use/budgets-goals/).
4. Open the reports to see where the money went.

## Keep your key

`APP_KEY` encrypts sensitive fields such as an AI provider key. Back it up with the database; see [backup and restore](/docs/run/backup-restore/).

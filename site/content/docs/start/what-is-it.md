---
title: "What vukem-finances is"
description: "An open-source personal finance application built with Vukem: what it does, what it does not do and how your data is treated."
---

vukem-finances is a personal finance application. It is written in Go on the server, with a React interface served by the same binary, and it keeps its data in a single SQLite file. It is open source under the MIT license, and the code is in [the repository](https://github.com/AlexGalhardo/vukem-finances).

## What it does

- **Accounts and transfers.** Checking, savings, cash, investment and wallet accounts, with balances derived from their transactions, and transfers between them.
- **Transactions.** Income and expenses with categories and tags, search and filters, and CSV and OFX statement import with duplicate detection.
- **Credit cards.** Cards with limits, closing and due days, statements and payments.
- **Recurrences and installments.** Rules that create transactions on their own, and purchases split into installments.
- **Budgets and goals.** Monthly budgets per category with optional rollover, and savings goals with a projected date.
- **Reports and exports.** Cash flow, spending by category, account and tag, net worth over time and a comparison with the previous period; CSV and JSON exports.
- **Integrations.** A REST API with scoped tokens, a command line, an MCP server for AI clients, a Telegram bot, and a mobile app that syncs.
- **AI with your own key.** Optional and off by default; see [AI with your own key](/docs/use/ai/).

## What it does not do

- It does not connect to your bank. Data comes from what you type, import or send through the API.
- It does not offer a hosted service. You run it yourself; see [the editions](/docs/start/editions/).
- It never stores money as a decimal number: every amount is an integer number of cents.

## Your data

Everything lives in one SQLite file that you can back up with a single command and export at any time as CSV or JSON. See [backup and restore](/docs/run/backup-restore/) and [reports and exports](/docs/use/insights-exports/). In the personal edition nothing leaves the machine unless you turn an integration on.

## Where to go next

The [editions](/docs/start/editions/) page explains the two ways the same binary can run, and the [quick start](/docs/start/quick-start/) gets you to a working installation.

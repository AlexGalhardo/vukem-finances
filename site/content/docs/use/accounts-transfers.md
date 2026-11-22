---
title: "Accounts and transfers"
description: "Create accounts, understand how balances are derived, move money between accounts with transfers and organize entries with categories and tags."
---

## Accounts

An account is a place where money sits: checking, savings, cash, investment or wallet. Credit cards are accounts too, made together with their card; see [credit cards](/docs/use/cards/).

An account has a name, a kind and an opening balance. Its balance is never typed in or stored: it is the opening balance plus every transaction of the account, so it can never drift from the entries behind it. The net worth on the dashboard is the sum of the balances of the accounts that are not archived, with card debt counting as a negative balance.

Archiving an account hides it from pickers and from the net worth, and refuses new entries, but keeps its history. In the SaaS edition archived accounts also leave the plan quota.

## Transfers

A transfer moves money from one account to another. It is stored as two linked transactions that share a transfer id: an expense leg on the source and an income leg on the destination, written together or not at all.

- A transfer is never income or expense. Both legs are left out of income and expense totals, category spending, budgets and the monthly transaction quota.
- Deleting or restoring one leg takes the other along. You cannot edit one leg on its own; edit the transfer.
- A transfer needs the `transfers:write` scope when you use the API.

## Categories

A category is either income or expense, and can have one level of subcategories of the same kind. It has a color and an icon from fixed lists, never a free color code. You can archive a category, which hides it from pickers and refuses new entries while keeping its history. Deleting a category moves what used it to another category of the same kind.

## Tags

Tags are free labels. A transaction can carry up to 10 of them and a workspace can hold up to 200; names are unique in any letter case. Saving a transaction with a tag that does not exist creates it. Reports can break spending down by tag, and a transaction with several tags counts under each one.

## Money rules

Amounts are integers in cents in the currency of the workspace. When you type an amount, `12.50` and `1.234,56` are both understood. The app never uses decimal numbers for money.

## From the command line and the API

The same operations exist outside the app: see the [accounts commands](/docs/integrate/cli/accounts/) and the [accounts endpoints](/docs/integrate/rest/accounts/).

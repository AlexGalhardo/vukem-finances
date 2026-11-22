---
title: "Credit cards"
description: "Track credit cards as accounts with a limit, closing and due days, statements that are derived from purchases, and payments from a bank account."
---

A credit card is an account of the kind `credit_card`, created together with its card details: a limit, a closing day, a due day, a default account to pay from, an optional brand and the last four digits. The app never stores a card number, an expiry date or a security code.

## Statements

A statement has no record of its own. Each purchase on a card is tied to the statement it falls in, computed from the closing day, and a statement's status is derived from the dates and payments:

| Status | Meaning |
| --- | --- |
| open | the closing date has not come yet |
| closed | the closing date passed and it is not paid |
| paid | what was owed has been paid |
| overdue | it is closed, unpaid and past its due date |

Changing the closing day moves only the statements that are not paid yet.

## Paying a statement

Paying is a transfer from a bank account to the card, never an expense, so paying a statement does not count twice in your spending. You pay the whole statement or a part. Through the API, an `amount_cents` of 0 pays what is left.

## The limit

The limit used is the debt of the card: the balance when it is negative, and zero otherwise. Archiving a card hides it; deleting an account takes its card along.

## Elsewhere

- Installment purchases on a card place each part on the statement of its month; see [recurrences and installments](/docs/use/recurrences-installments/).
- Commands: [cards](/docs/integrate/cli/cards/) and [statements](/docs/integrate/cli/statements/). Endpoints: [cards](/docs/integrate/rest/cards/) and [statements](/docs/integrate/rest/statements/).

---
title: "Transactions and import"
description: "Record, search and edit transactions, and import bank statements from CSV or OFX files with duplicate detection and safe limits."
---

## Recording a transaction

A transaction has an account, a kind (income or expense), an amount, a date, an optional category, an optional description and up to 10 tags. The quickest ways to add one:

- the Add button, or the `n e` and `n i` keyboard shortcuts for a new expense or income (see [keyboard shortcuts](/docs/use/shortcuts/));
- a short message to the [Telegram bot](/docs/use/telegram/), such as `-12,50 lunch`;
- the [command line](/docs/integrate/cli/transactions/), the [REST API](/docs/integrate/rest/transactions/) or an [MCP](/docs/integrate/mcp-stdio/) client.

Dates are calendar dates in the time zone of the workspace; "today" and "this month" follow that zone.

## Searching and filtering

The transaction list filters by period, account, category, tag, kind, an amount range and a text search of the description. Press `/` to jump to the search field. The same filters are used by the CSV export, so what you see is what you export.

## Deleting and the trash

A deleted transaction is a soft delete: the row stays marked as deleted so that sync can tell other devices about it. Deleting one leg of a transfer deletes both.

## Importing a statement

Import reads a CSV or OFX file from your bank. The importer treats the file as untrusted:

- the file is at most 2 MiB and at most 20000 rows; a larger file is refused, never cut;
- a row it cannot read is reported as a failure and nothing is guessed;
- amounts are parsed as exact cents, never as floating point numbers;
- the OFX reader resolves no entity and refuses a DTD, CDATA or comment.

You choose the account and map the columns of a CSV (date, description, amount and optionally category). Every row gets a fingerprint made from the account, date, signed amount and normalized description, or the bank's own transaction id for OFX. A row the account already has is skipped unless you allow duplicates, so importing the same file twice does nothing. The whole import is one database transaction: a failed import leaves nothing behind.

Imported rows are ordinary entries, never transfer legs. In the SaaS edition, CSV import is part of the Pro plan.

## The CSV you export

The CSV that the app exports can be imported again. Two things do not survive the round trip by design: a description that was protected against spreadsheet formulas keeps its protective quote, and a transfer comes back as an ordinary expense and income. See [the export format](/docs/integrate/export-format/).

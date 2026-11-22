---
title: "Insights and exports"
description: "The five report views, how periods work, and how to take your data out as a transactions CSV, a workspace JSON file or a printable page."
---

## Reports

The reports page shows five views for one period. The period is a preset (`this_month`, `last_month`, `last_3_months`, `last_12_months` or `this_year`) or a custom range of dates.

1. **Cash flow**: income, expense and net per month.
2. **Categories**: spending and income by category, with subcategories under their parent and the uncategorized entries apart.
3. **Accounts and tags**: the same totals by account and by tag. A transaction with several tags counts under each one.
4. **Net worth**: the balance at the end of each month, with card debt as a negative balance.
5. **Comparison** with the previous period of the same length. The change is computed in basis points with integers and shows `n/a` when the previous value is zero.

Every view leaves transfer legs and deleted rows out, counts income and expense on the transaction date, and includes future-dated rows only when the period reaches them. Every chart has a table with the same numbers: the "Show as table" button swaps it in, and screen readers get the table by default. Printing shows the tables and hides the shell.

## Exports

You can take your data out at any time.

- **Transactions CSV.** Download it from the transactions page, with the same filters as the list. It opens in a spreadsheet and imports back. It has at most 200000 rows; a larger result is refused before the first byte, never cut. Options add a byte order mark for old versions of Excel, or a semicolon delimiter for locales that use a decimal comma. Any text cell that could run as a spreadsheet formula is written with a leading quote.
- **Workspace JSON.** One document with accounts, categories, tags, transactions, budgets, goals, recurrences and cards. Only the owner and the admins may take it. It never contains password hashes, tokens, API keys, two-factor data, billing identifiers, the AI key or any email address.
- **Printable pages**, from the browser's print dialog.

Both files have a documented, versioned contract: see [the export format](/docs/integrate/export-format/). The same exports are available through the API as `GET /api/v1/export/transactions` with the `export:read` scope; see the [export endpoints](/docs/integrate/rest/export/).

## Limits

The CSV and the JSON together are limited to 10 downloads per workspace per hour.

Commands: [the report commands](/docs/integrate/cli/reports/) and [the export commands](/docs/integrate/cli/export/).

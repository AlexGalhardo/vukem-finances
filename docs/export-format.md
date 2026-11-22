# Export format

A person can take their data out at any time: a CSV of transactions that opens in a spreadsheet and imports back, a JSON export of the whole workspace, or a printable page of the reports. This page is the contract of the first two. The reports are in `app/reports/`, the exporters in `app/export/`.

## Transactions CSV

`GET /transactions/export.csv` takes the same filters as the transaction list (`from`, `to`, `category`, `account`, `kind`, `q`, `tag`, `min`, `max`) and streams the matching rows, oldest first. Any member of the workspace may download it, a viewer included.

- UTF-8, `\r\n` line ends, RFC 4180 quoting, one header row.
- Options: `bom=1` writes a UTF-8 byte order mark (old versions of Excel need it to read UTF-8), `delimiter=semicolon` separates cells with `;` (for locales whose decimal separator is a comma). `comma` is the default; any other delimiter is a 422.
- At most 200000 rows. A larger result is refused with a 413 before the first byte, never cut: narrow the period or the filters.
- The file is named `vukem-finances-transactions-YYYY-MM-DD.csv` with the date of the workspace, and is sent with `Content-Disposition: attachment` and `X-Content-Type-Options: nosniff`.
- 10 downloads (CSV and JSON together) per workspace per hour; the 429 carries `Retry-After`.

| Column | Meaning |
| --- | --- |
| `uid` | The id of the transaction (a ULID). |
| `date` | `YYYY-MM-DD`. |
| `account` | The account name. |
| `category` | The category name, empty when there is none. |
| `tags` | The tag names joined with `;`. |
| `kind` | `income` or `expense`. |
| `amount` | Signed decimal with a dot and two decimals: `-12.34` for an expense, `100.00` for income. Built from integer cents, never a float. |
| `description` | The text of the transaction. |
| `transfer_uid` | Set on both legs of a transfer, the same value on each; empty otherwise. |

### Formula guard

A spreadsheet evaluates a cell that starts with `=`, `+`, `-` or `@` (and some skip a leading tab or carriage return). Every text cell that starts with one of those characters is written with a leading single quote (`'=HYPERLINK(...)`), so it opens as plain text. The `amount` column is the one exception: it is built from an integer and its minus sign is data.

### Importing it back

The CSV is accepted by the statement importer with this mapping: date `date` (format `2006-01-02`), description `description`, amount `amount`, category `category`. Both the BOM and the semicolon variants are read. Two things do not survive the round trip, by design:

- A description that was guarded keeps its quote (`'=1+1` comes back as `'=1+1`): the importer stores text as typed.
- The importer never makes transfer legs, so a transfer comes back as an ordinary expense and an ordinary income. Balances agree; income and expense totals include them.

## Workspace JSON

`GET /workspace/export.json` streams one JSON document of the whole workspace. Only the owner and the admins may take it (a member or a viewer gets a 403). It is described by [`export-format.schema.json`](export-format.schema.json) (JSON Schema, `additionalProperties: false` everywhere); a test validates the real output against it.

```json
{
  "format": "vukem-finances-export",
  "version": 1,
  "exported_at": "2026-11-21T22:00:00Z",
  "workspace": { "name": "...", "currency": "USD", "timezone": "UTC", "members": [{ "name": "Ana", "role": "owner" }] },
  "accounts": [], "categories": [], "tags": [], "transactions": [],
  "budgets": [], "goals": [], "recurrences": [], "cards": []
}
```

- Money is an integer in cents (`amount_cents` is positive; `kind` says the direction, as in the database). References between rows are uids (`account_uid`, `category_uid`, `parent_uid`, `tag_uids`, `transfer_uid`). Dates are `YYYY-MM-DD`, instants are RFC 3339 in UTC.
- Only live rows are exported: nothing from the trash.
- Goals carry their `contributions`. A card carries its `account_uid` (its account is in `accounts`).
- At most 500000 transactions; more is a 413.
- Never in the export: password hashes, tokens and API keys, two-factor data, Stripe or PIX identifiers, sessions, the AI key and its settings, and any email address. Members appear by display name and role only.
- `version` changes when a field is removed or changes meaning; adding a field is a new minor of the schema and keeps `version` as it is only if readers can ignore it. The schema forbids unknown fields on purpose, so a column added to a model later reaches the export only when it is added to the exporter and to the schema together.

Importing this file back is not supported yet.

## Reports

`GET /reports` shows five reports for one period (`?period=` is `this_month`, `last_month`, `last_3_months`, `last_12_months`, `this_year` or `custom` with `from` and `to`):

1. Cash flow: income, expense and net per month.
2. Categories: spending and income by category, parents rolled up with their children beneath, the uncategorized row apart.
3. Accounts and tags. A transaction with several tags counts under each one.
4. Net worth at the end of each month: the opening balances of the accounts that are not archived plus every transaction up to that day, with card debt as a negative balance.
5. Comparison with the previous period (same length; for whole months, the same number of months before). The change is in basis points, computed with integers, and is `n/a` when the previous value is zero.

All of them leave transfer legs and deleted rows out, count income and expense on the transaction date, and include future-dated rows only when the period reaches them. Every chart has a table with the same numbers (the "Show as table" button; screen readers get the table by default, and printing shows it), so "Print or save as PDF" in the Export menu needs no library.

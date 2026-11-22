---
title: "Export format"
description: "The contract of the transactions CSV and the workspace JSON export: columns, limits, the spreadsheet formula guard, privacy rules and versioning."
---

You can take your data out at any time: a CSV of transactions that opens in a spreadsheet and imports back, or a JSON export of the whole workspace. This page is the contract of both. For the screens, see [insights and exports](/docs/use/insights-exports/).

## Transactions CSV

The CSV takes the same filters as the transaction list (`from`, `to`, `category`, `account`, `kind`, `q`, `tag`, `min` and `max`) and streams the matching rows, oldest first. Any member of the workspace may download it, a viewer included. Through the API it is `GET /api/v1/export/transactions`.

- UTF-8, `\r\n` line ends, RFC 4180 quoting and one header row.
- `bom=1` writes a UTF-8 byte order mark, which old versions of Excel need. `delimiter=semicolon` separates cells with `;` for locales whose decimal separator is a comma. `comma` is the default, and any other delimiter is a 422.
- At most 200000 rows. A larger result is refused with a 413 before the first byte, never cut: narrow the period or the filters.
- The file is named `vukem-finances-transactions-YYYY-MM-DD.csv` and sent as an attachment with `X-Content-Type-Options: nosniff`.
- 10 downloads, CSV and JSON together, per workspace per hour.

| Column | Meaning |
| --- | --- |
| `uid` | the id of the transaction, a ULID |
| `date` | `YYYY-MM-DD` |
| `account` | the account name |
| `category` | the category name, empty when there is none |
| `tags` | the tag names joined with `;` |
| `kind` | `income` or `expense` |
| `amount` | signed decimal with a dot and two decimals, such as `-12.34` for an expense; built from integer cents, never a float |
| `description` | the text of the transaction |
| `transfer_uid` | set on both legs of a transfer, the same value on each; empty otherwise |

### The formula guard

A spreadsheet evaluates a cell that starts with `=`, `+`, `-` or `@`. Every text cell whose first visible character is one of those, or its full-width form, is written with a leading single quote, so it opens as plain text. White space and invisible characters before it do not hide it. The same guard applies to each piece of text after a comma, semicolon, tab or line break, because Excel in a locale with another list separator splits the row there. The `amount` column is the one exception.

### Importing it back

The importer accepts the CSV with the mapping date, description, amount and category. Both the byte order mark and the semicolon variants are read. Two things do not survive the round trip, by design: a guarded description keeps its quote, and a transfer comes back as an ordinary expense and an ordinary income. Balances agree; income and expense totals include the legs.

## Workspace JSON

`GET /workspace/export.json` streams one JSON document of the whole workspace. Only the owner and the admins may take it. It is described by `docs/export-format.schema.json` in the repository, a JSON Schema with `additionalProperties: false` everywhere, and a test validates the real output against it.

```json
{
  "format": "vukem-finances-export",
  "version": 1,
  "exported_at": "2026-11-21T22:00:00Z",
  "workspace": { "name": "...", "currency": "USD", "timezone": "UTC", "members": [{ "name": "Ana", "role": "owner" }] },
  "accounts": [],
  "categories": [],
  "tags": [],
  "transactions": [],
  "budgets": [],
  "goals": [],
  "recurrences": [],
  "cards": []
}
```

- Money is an integer in cents. `amount_cents` is positive and `kind` says the direction. References between rows are uids, dates are `YYYY-MM-DD` and instants are RFC 3339 in UTC.
- Only live rows are exported: nothing from the trash.
- Goals carry their contributions. A card carries its `account_uid`.
- At most 500000 transactions; more is a 413.
- **Never in the export:** password hashes, tokens and API keys, two-factor data, billing identifiers, sessions, the AI key and its settings, and any email address. Members appear by display name and role only.

## Versioning

`version` changes when a field is removed or changes meaning. Adding a field is a new minor of the schema and keeps `version` when readers can ignore it. The schema forbids unknown fields on purpose, so a column added to a model later reaches the export only when it is added to the exporter and to the schema. Importing the JSON file back is not supported yet.

---
title: "REST: Export"
description: "The Export endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

2 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /export/transactions`

Download the transactions that match the filters as a CSV file.

| Field | Value |
| --- | --- |
| Operation | `export.transactions` |
| Scope | `export:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `from` | query | `string` |  | First date, YYYY-MM-DD |
| `to` | query | `string` |  | Last date, YYYY-MM-DD |
| `account` | query | `string` |  | Only this account (uid) |
| `category` | query | `string` |  | Only this category and its subcategories (uid) |
| `tag` | query | `string` |  | Only transactions with this tag (uid) |
| `kind` | query | `string` |  | income or expense |
| `q` | query | `string` |  | Text of the description, case-insensitive |
| `min_cents` | query | `integer (int64)` |  | Smallest absolute amount, in cents |
| `max_cents` | query | `integer (int64)` |  | Largest absolute amount, in cents |
| `delimiter` | query | `string` |  | comma (the default) or semicolon, for locales whose decimal separator is a comma |
| `bom` | query | `boolean` |  | Start the file with a byte order mark, for old versions of Excel |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK |  |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/export/transactions"
```

## `GET /export/workspace`

Download the whole workspace as one JSON document (owner and admins).

| Field | Value |
| --- | --- |
| Operation | `export.workspace` |
| Scope | `export:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `object` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/export/workspace"
```

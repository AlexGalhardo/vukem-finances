---
title: "REST: Transactions"
description: "The Transactions endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

5 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /transactions`

List transactions, newest first, with filters.

| Field | Value |
| --- | --- |
| Operation | `transactions.list` |
| Scope | `transactions:read` |
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
| `page` | query | `integer` |  | Page number, from 1; a page holds 50 rows |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<TransactionOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TransactionOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `date` | `string` | yes |  |
| `description` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `tags` | `array of TagRefOut` | yes |  |
| `transfer_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/transactions"
```

## `POST /transactions`

Record a transaction.

| Field | Value |
| --- | --- |
| Operation | `transactions.create` |
| Scope | `transactions:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes | At most 26 characters. |
| `amount_cents` | `integer (int64)` |  | A positive whole number of cents; the kind gives the sign |
| `category_uid` | `string` |  | A category of the same kind; leave out for none At most 26 characters. |
| `date` | `string` | yes | At most 10 characters. |
| `description` | `string` |  | At most 200 characters. |
| `kind` | `string` | yes | income or expense At most 20 characters. |
| `tags` | `array of string` |  | Tag names, created when new; at most 10 per transaction |
| `uid` | `string` |  | Optional uid chosen by the client (a ULID); repeating the call returns the row already made At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `TransactionOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TransactionOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `date` | `string` | yes |  |
| `description` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `tags` | `array of TagRefOut` | yes |  |
| `transfer_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"account_uid":"","kind":"expense","date":"2026-11-21"}' \
  "$BASE/api/v1/transactions"
```

## `GET /transactions/{uid}`

Read a transaction.

| Field | Value |
| --- | --- |
| Operation | `transactions.show` |
| Scope | `transactions:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the row |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `TransactionOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TransactionOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `date` | `string` | yes |  |
| `description` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `tags` | `array of TagRefOut` | yes |  |
| `transfer_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/transactions/<uid>"
```

## `PATCH /transactions/{uid}`

Change a transaction.

| Field | Value |
| --- | --- |
| Operation | `transactions.update` |
| Scope | `transactions:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes |  |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` |  | At most 26 characters. |
| `amount_cents` | `integer (int64)` |  |  |
| `category_uid` | `string` |  | At most 26 characters. |
| `date` | `string` |  | At most 10 characters. |
| `description` | `string` |  | At most 200 characters. |
| `kind` | `string` |  | At most 20 characters. |
| `tags` | `array of string` |  |  |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `TransactionOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TransactionOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `date` | `string` | yes |  |
| `description` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `tags` | `array of TagRefOut` | yes |  |
| `transfer_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"account_uid":"","amount_cents":0,"category_uid":""}' \
  "$BASE/api/v1/transactions/<uid>"
```

## `DELETE /transactions/{uid}`

Delete a transaction.

| Field | Value |
| --- | --- |
| Operation | `transactions.delete` |
| Scope | `transactions:write` |
| Kind | destructive |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the row |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 204 | No Content |  |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Example

```sh
curl -X DELETE -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/transactions/<uid>"
```

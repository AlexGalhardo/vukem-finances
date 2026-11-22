---
title: "REST: Cards"
description: "The Cards endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

6 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /cards`

List the credit cards with their limits and statements.

| Field | Value |
| --- | --- |
| Operation | `cards.list` |
| Scope | `cards:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<CardOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/cards"
```

## `POST /cards`

Create a credit card with its account.

| Field | Value |
| --- | --- |
| Operation | `cards.create` |
| Scope | `cards:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `brand` | `string` |  | visa, mastercard, amex, elo, hipercard, diners or other At most 20 characters. |
| `closing_day` | `integer` |  | Day of the month the statement closes |
| `due_day` | `integer` |  | Day of the month the statement is due |
| `last4` | `string` |  | The last 4 digits, nothing more At most 4 characters. |
| `limit_cents` | `integer (int64)` |  | The credit limit, a positive whole number of cents |
| `name` | `string` | yes | At most 80 characters. |
| `payment_account_uid` | `string` |  | The bank account statements are paid from by default; leave out for none At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `CardOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Visa"}' \
  "$BASE/api/v1/cards"
```

## `GET /cards/{uid}`

Read a credit card.

| Field | Value |
| --- | --- |
| Operation | `cards.show` |
| Scope | `cards:read` |
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
| 200 | OK | `CardOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/cards/<uid>"
```

## `PATCH /cards/{uid}`

Change a credit card.

| Field | Value |
| --- | --- |
| Operation | `cards.update` |
| Scope | `cards:write` |
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
| `brand` | `string` |  | At most 20 characters. |
| `closing_day` | `integer` |  |  |
| `due_day` | `integer` |  |  |
| `last4` | `string` |  | At most 4 characters. |
| `limit_cents` | `integer (int64)` |  |  |
| `name` | `string` |  | At most 80 characters. |
| `payment_account_uid` | `string` |  | At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CardOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"brand":"","closing_day":0,"due_day":0}' \
  "$BASE/api/v1/cards/<uid>"
```

## `POST /cards/{uid}/archive`

Archive a credit card.

| Field | Value |
| --- | --- |
| Operation | `cards.archive` |
| Scope | `cards:write` |
| Kind | destructive |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the row |

### Request body

A JSON value of type object.

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CardOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/cards/<uid>/archive"
```

## `POST /cards/{uid}/unarchive`

Bring an archived credit card back.

| Field | Value |
| --- | --- |
| Operation | `cards.unarchive` |
| Scope | `cards:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the row |

### Request body

A JSON value of type object.

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CardOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CardOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `archived` | `boolean` | yes |  |
| `available_cents` | `integer (int64)` | yes |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `brand` | `string` | yes |  |
| `closing_day` | `integer` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `current` | `object` | yes | The statement open today |
| `due` | `StatementOut` |  | The oldest closed statement still unpaid, when there is one |
| `due_day` | `integer` | yes |  |
| `last4` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `name` | `string` | yes |  |
| `over_limit` | `boolean` | yes |  |
| `payment_account_uid` | `string` |  | The account statements are paid from by default |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `used_cents` | `integer (int64)` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/cards/<uid>/unarchive"
```

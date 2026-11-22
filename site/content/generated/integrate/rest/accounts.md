---
title: "REST: Accounts"
description: "The Accounts endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

5 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /accounts`

List the accounts with their balances.

| Field | Value |
| --- | --- |
| Operation | `accounts.list` |
| Scope | `accounts:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<AccountOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `archived_at` | `string` |  |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `opening_cents` | `integer (int64)` | yes |  |
| `position` | `integer` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/accounts"
```

## `POST /accounts`

Create an account.

| Field | Value |
| --- | --- |
| Operation | `accounts.create` |
| Scope | `accounts:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kind` | `string` | yes | checking, savings, cash, investment or wallet At most 20 characters. |
| `name` | `string` | yes | At most 80 characters. |
| `opening_cents` | `integer (int64)` |  | Opening balance in cents |
| `uid` | `string` |  | Optional uid chosen by the client (a ULID); repeating the call returns the row already made At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `AccountOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `archived_at` | `string` |  |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `opening_cents` | `integer (int64)` | yes |  |
| `position` | `integer` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Checking","kind":"checking"}' \
  "$BASE/api/v1/accounts"
```

## `GET /accounts/{uid}`

Read an account.

| Field | Value |
| --- | --- |
| Operation | `accounts.show` |
| Scope | `accounts:read` |
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
| 200 | OK | `AccountOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `archived_at` | `string` |  |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `opening_cents` | `integer (int64)` | yes |  |
| `position` | `integer` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/accounts/<uid>"
```

## `PATCH /accounts/{uid}`

Change the name, kind and opening balance of an account.

| Field | Value |
| --- | --- |
| Operation | `accounts.update` |
| Scope | `accounts:write` |
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
| `kind` | `string` |  | At most 20 characters. |
| `name` | `string` |  | At most 80 characters. |
| `opening_cents` | `integer (int64)` |  |  |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `AccountOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `archived_at` | `string` |  |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `opening_cents` | `integer (int64)` | yes |  |
| `position` | `integer` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"kind":"","name":"","opening_cents":0}' \
  "$BASE/api/v1/accounts/<uid>"
```

## `POST /accounts/{uid}/archive`

Archive an account.

| Field | Value |
| --- | --- |
| Operation | `accounts.archive` |
| Scope | `accounts:write` |
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
| 200 | OK | `AccountOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `archived_at` | `string` |  |  |
| `balance_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `opening_cents` | `integer (int64)` | yes |  |
| `position` | `integer` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/accounts/<uid>/archive"
```

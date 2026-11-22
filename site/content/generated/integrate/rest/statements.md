---
title: "REST: Statements"
description: "The Statements endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

3 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /cards/{uid}/statements`

List the statements of a credit card, newest first.

| Field | Value |
| --- | --- |
| Operation | `statements.list` |
| Scope | `statements:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the card |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<StatementOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `StatementOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `closing_on` | `string` | yes |  |
| `credits_cents` | `integer (int64)` | yes |  |
| `due_on` | `string` | yes |  |
| `paid_cents` | `integer (int64)` | yes |  |
| `purchases_cents` | `integer (int64)` | yes |  |
| `ref` | `string` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `status` | `string` | yes | open, closed, paid or overdue |
| `total_cents` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/cards/<uid>/statements"
```

## `GET /cards/{uid}/statements/{ref}`

Read a statement of a credit card with its rows.

| Field | Value |
| --- | --- |
| Operation | `statements.show` |
| Scope | `statements:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the card |
| `ref` | path | `string` | yes | The closing date of the statement, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `StatementDetailOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `StatementDetailOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `rows` | `array of TransactionOut` | yes |  |
| `statement` | `StatementOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/cards/<uid>/statements/<ref>"
```

## `POST /cards/{uid}/statements/{ref}/pay`

Pay a statement from a bank account.

| Field | Value |
| --- | --- |
| Operation | `statements.pay` |
| Scope | `statements:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the card |
| `ref` | path | `string` | yes | The closing date of the statement, YYYY-MM-DD |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `amount_cents` | `integer (int64)` |  | What to pay, in cents; 0 pays what is left of the statement |
| `date` | `string` |  | The payment date; today when left out At most 10 characters. |
| `from_account_uid` | `string` | yes | The bank account the payment leaves At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `TransferOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TransferOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `from` | `TransactionOut` | yes |  |
| `to` | `TransactionOut` | yes |  |
| `uid` | `string` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"from_account_uid":""}' \
  "$BASE/api/v1/cards/<uid>/statements/<ref>/pay"
```

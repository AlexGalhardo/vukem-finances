---
title: "REST: Transfers"
description: "The Transfers endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

2 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `POST /transfers`

Move money between two accounts.

| Field | Value |
| --- | --- |
| Operation | `transfers.create` |
| Scope | `transfers:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `amount_cents` | `integer (int64)` |  | A positive whole number of cents |
| `date` | `string` | yes | At most 10 characters. |
| `description` | `string` |  | At most 200 characters. |
| `from_account_uid` | `string` | yes | At most 26 characters. |
| `to_account_uid` | `string` | yes | At most 26 characters. |

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
  -d '{"from_account_uid":"","to_account_uid":"","date":"2026-11-21"}' \
  "$BASE/api/v1/transfers"
```

## `DELETE /transfers/{uid}`

Delete a transfer and both of its legs.

| Field | Value |
| --- | --- |
| Operation | `transfers.delete` |
| Scope | `transfers:write` |
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
  "$BASE/api/v1/transfers/<uid>"
```

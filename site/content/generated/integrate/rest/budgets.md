---
title: "REST: Budgets"
description: "The Budgets endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

3 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /budgets`

List the budgets with what the month spent.

| Field | Value |
| --- | --- |
| Operation | `budgets.list` |
| Scope | `budgets:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `month` | query | `string` |  | YYYY-MM; this month in the workspace timezone when left out |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<BudgetOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `BudgetOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `category_name` | `string` | yes |  |
| `category_uid` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `month` | `string` | yes |  |
| `percent` | `integer` | yes |  |
| `spent_cents` | `integer (int64)` | yes |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/budgets"
```

## `POST /budgets`

Set the monthly limit of an expense category.

| Field | Value |
| --- | --- |
| Operation | `budgets.set` |
| Scope | `budgets:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `category_uid` | `string` | yes | An expense category At most 26 characters. |
| `limit_cents` | `integer (int64)` |  | The monthly limit, a positive whole number of cents |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `BudgetOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `BudgetOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `category_name` | `string` | yes |  |
| `category_uid` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `limit_cents` | `integer (int64)` | yes |  |
| `month` | `string` | yes |  |
| `percent` | `integer` | yes |  |
| `spent_cents` | `integer (int64)` | yes |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"category_uid":""}' \
  "$BASE/api/v1/budgets"
```

## `DELETE /budgets/{uid}`

Remove a budget.

| Field | Value |
| --- | --- |
| Operation | `budgets.delete` |
| Scope | `budgets:write` |
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
  "$BASE/api/v1/budgets/<uid>"
```

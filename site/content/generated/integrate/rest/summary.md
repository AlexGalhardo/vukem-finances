---
title: "REST: Summary"
description: "The Summary endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

1 operation. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /summary`

Income, expense, net, account balances and net worth of a month.

| Field | Value |
| --- | --- |
| Operation | `summary.show` |
| Scope | `summary:read` |
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
| 200 | OK | `SummaryOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `SummaryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `accounts` | `array of SummaryAccountOut` | yes |  |
| `currency` | `string` | yes |  |
| `expense_cents` | `integer (int64)` | yes |  |
| `income_cents` | `integer (int64)` | yes |  |
| `month` | `string` | yes |  |
| `net_cents` | `integer (int64)` | yes |  |
| `net_worth_cents` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/summary"
```

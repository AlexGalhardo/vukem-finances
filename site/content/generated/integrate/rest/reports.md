---
title: "REST: Reports"
description: "The Reports endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

6 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /reports/accounts`

Income and expense of a period by account.

| Field | Value |
| --- | --- |
| Operation | `reports.accounts` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `AccountReportOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `AccountReportOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `rows` | `array of AccountRowOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/accounts"
```

## `GET /reports/cash-flow`

Income against expense per month of a period.

| Field | Value |
| --- | --- |
| Operation | `reports.cashflow` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CashFlowOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CashFlowOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `months` | `array of MonthFlowOut` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `totals` | `SumOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/cash-flow"
```

## `GET /reports/categories`

Spending or income of a period by category.

| Field | Value |
| --- | --- |
| Operation | `reports.categories` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |
| `kind` | query | `string` |  | expense (the default) or income |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CategoryReportOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryReportOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `rows` | `array of CategoryRowOut` | yes |  |
| `total_cents` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/categories"
```

## `GET /reports/compare`

A period against the one right before it.

| Field | Value |
| --- | --- |
| Operation | `reports.compare` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CompareReportOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CompareReportOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `previous` | `PeriodOut` | yes |  |
| `rows` | `array of CompareRowOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/compare"
```

## `GET /reports/net-worth`

The net worth at the end of each month of a period.

| Field | Value |
| --- | --- |
| Operation | `reports.networth` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `NetWorthReportOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `NetWorthReportOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `points` | `array of NetWorthPointOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/net-worth"
```

## `GET /reports/tags`

Income and expense of a period by tag.

| Field | Value |
| --- | --- |
| Operation | `reports.tags` |
| Scope | `reports:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `period` | query | `string` |  | this_month (the default), last_month, last_3_months, last_12_months, this_year or custom |
| `from` | query | `string` |  | First date of a custom period, YYYY-MM-DD |
| `to` | query | `string` |  | Last date of a custom period, YYYY-MM-DD |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `TagReportOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TagReportOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `currency` | `string` | yes |  |
| `period` | `PeriodOut` | yes |  |
| `rows` | `array of TagRowOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/reports/tags"
```

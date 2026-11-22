---
title: "REST: Recurrences"
description: "The Recurrences endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

4 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /recurrences`

List the recurring rules.

| Field | Value |
| --- | --- |
| Operation | `recurrences.list` |
| Scope | `recurrences:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<RecurrenceOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `RecurrenceOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `active` | `boolean` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `description` | `string` | yes |  |
| `end_date` | `string` |  |  |
| `frequency` | `string` | yes | daily, weekly, monthly or yearly |
| `generated_through` | `string` |  | The last date the rule has been run up to |
| `interval` | `integer` | yes | Repeat every this many days, weeks, months or years |
| `kind` | `string` | yes |  |
| `max_count` | `integer (int64)` |  |  |
| `start_date` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/recurrences"
```

## `POST /recurrences`

Create a recurring rule and post what is already due.

| Field | Value |
| --- | --- |
| Operation | `recurrences.create` |
| Scope | `recurrences:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes | At most 26 characters. |
| `amount_cents` | `integer (int64)` |  | A positive whole number of cents; the kind gives the sign |
| `backfill` | `boolean` |  | Post the occurrences that fell before today, at most 36 |
| `category_uid` | `string` |  | A category of the same kind; leave out for none At most 26 characters. |
| `description` | `string` |  | At most 200 characters. |
| `end_date` | `string` |  | The last date the rule may run; leave out for no end At most 10 characters. |
| `frequency` | `string` | yes | daily, weekly, monthly or yearly At most 20 characters. |
| `interval` | `integer` |  | Repeat every this many days, weeks, months or years |
| `kind` | `string` | yes | income or expense At most 20 characters. |
| `max_count` | `integer` |  | At most this many occurrences; 0 for no limit |
| `start_date` | `string` | yes | At most 10 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `RecurrenceOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `RecurrenceOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `active` | `boolean` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `description` | `string` | yes |  |
| `end_date` | `string` |  |  |
| `frequency` | `string` | yes | daily, weekly, monthly or yearly |
| `generated_through` | `string` |  | The last date the rule has been run up to |
| `interval` | `integer` | yes | Repeat every this many days, weeks, months or years |
| `kind` | `string` | yes |  |
| `max_count` | `integer (int64)` |  |  |
| `start_date` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"account_uid":"","kind":"expense","frequency":"monthly","start_date":"2026-12-01"}' \
  "$BASE/api/v1/recurrences"
```

## `PATCH /recurrences/{uid}`

Change a recurring rule.

| Field | Value |
| --- | --- |
| Operation | `recurrences.update` |
| Scope | `recurrences:write` |
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
| `active` | `boolean` |  |  |
| `amount_cents` | `integer (int64)` |  |  |
| `category_uid` | `string` |  | At most 26 characters. |
| `description` | `string` |  | At most 200 characters. |
| `end_date` | `string` |  | At most 10 characters. |
| `frequency` | `string` |  | At most 20 characters. |
| `interval` | `integer` |  |  |
| `kind` | `string` |  | At most 20 characters. |
| `max_count` | `integer` |  |  |
| `start_date` | `string` |  | At most 10 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `RecurrenceOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `RecurrenceOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `account_uid` | `string` | yes |  |
| `active` | `boolean` | yes |  |
| `amount_cents` | `integer (int64)` | yes |  |
| `category_uid` | `string` |  |  |
| `created_at` | `string` | yes |  |
| `currency` | `string` | yes |  |
| `description` | `string` | yes |  |
| `end_date` | `string` |  |  |
| `frequency` | `string` | yes | daily, weekly, monthly or yearly |
| `generated_through` | `string` |  | The last date the rule has been run up to |
| `interval` | `integer` | yes | Repeat every this many days, weeks, months or years |
| `kind` | `string` | yes |  |
| `max_count` | `integer (int64)` |  |  |
| `start_date` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"account_uid":"","active":false,"amount_cents":0}' \
  "$BASE/api/v1/recurrences/<uid>"
```

## `DELETE /recurrences/{uid}`

Delete a recurring rule and keep what it generated.

| Field | Value |
| --- | --- |
| Operation | `recurrences.delete` |
| Scope | `recurrences:write` |
| Kind | destructive |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `future` | query | `boolean` |  | Also delete the generated transactions dated after today |
| `uid` | path | `string` | yes |  |

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
  "$BASE/api/v1/recurrences/<uid>"
```

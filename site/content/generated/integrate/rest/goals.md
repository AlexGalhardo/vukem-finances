---
title: "REST: Goals"
description: "The Goals endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

8 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /goals`

List the savings goals with their progress.

| Field | Value |
| --- | --- |
| Operation | `goals.list` |
| Scope | `goals:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<GoalOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `GoalOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `contributions` | `integer` | yes |  |
| `currency` | `string` | yes |  |
| `months_left` | `integer` | yes | Months to the target date, for a goal with a future date that is not reached |
| `name` | `string` | yes |  |
| `overdue` | `boolean` | yes | The target date has passed and the goal is not reached |
| `percent` | `integer` | yes |  |
| `projected_date` | `string` |  | When the pace of the last months would reach the target |
| `reached` | `boolean` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `required_monthly_cents` | `integer (int64)` | yes | What to put aside each month to reach the target by its date |
| `saved_cents` | `integer (int64)` | yes |  |
| `target_cents` | `integer (int64)` | yes |  |
| `target_date` | `string` |  |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/goals"
```

## `POST /goals`

Create a savings goal.

| Field | Value |
| --- | --- |
| Operation | `goals.create` |
| Scope | `goals:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | `string` | yes | At most 120 characters. |
| `target_cents` | `integer (int64)` |  | The amount to reach, a positive whole number of cents |
| `target_date` | `string` |  | When to reach it, YYYY-MM-DD; leave out for no date At most 10 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `GoalOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `GoalOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `contributions` | `integer` | yes |  |
| `currency` | `string` | yes |  |
| `months_left` | `integer` | yes | Months to the target date, for a goal with a future date that is not reached |
| `name` | `string` | yes |  |
| `overdue` | `boolean` | yes | The target date has passed and the goal is not reached |
| `percent` | `integer` | yes |  |
| `projected_date` | `string` |  | When the pace of the last months would reach the target |
| `reached` | `boolean` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `required_monthly_cents` | `integer (int64)` | yes | What to put aside each month to reach the target by its date |
| `saved_cents` | `integer (int64)` | yes |  |
| `target_cents` | `integer (int64)` | yes |  |
| `target_date` | `string` |  |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Trip"}' \
  "$BASE/api/v1/goals"
```

## `PATCH /goals/{uid}`

Change the name, target and date of a goal.

| Field | Value |
| --- | --- |
| Operation | `goals.update` |
| Scope | `goals:write` |
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
| `name` | `string` |  | At most 120 characters. |
| `target_cents` | `integer (int64)` |  |  |
| `target_date` | `string` |  | At most 10 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `GoalOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `GoalOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `contributions` | `integer` | yes |  |
| `currency` | `string` | yes |  |
| `months_left` | `integer` | yes | Months to the target date, for a goal with a future date that is not reached |
| `name` | `string` | yes |  |
| `overdue` | `boolean` | yes | The target date has passed and the goal is not reached |
| `percent` | `integer` | yes |  |
| `projected_date` | `string` |  | When the pace of the last months would reach the target |
| `reached` | `boolean` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `required_monthly_cents` | `integer (int64)` | yes | What to put aside each month to reach the target by its date |
| `saved_cents` | `integer (int64)` | yes |  |
| `target_cents` | `integer (int64)` | yes |  |
| `target_date` | `string` |  |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"","target_cents":0,"target_date":""}' \
  "$BASE/api/v1/goals/<uid>"
```

## `DELETE /goals/{uid}`

Delete a goal and its contributions.

| Field | Value |
| --- | --- |
| Operation | `goals.delete` |
| Scope | `goals:write` |
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
  "$BASE/api/v1/goals/<uid>"
```

## `POST /goals/{uid}/archive`

Archive a goal; its contributions stay.

| Field | Value |
| --- | --- |
| Operation | `goals.archive` |
| Scope | `goals:write` |
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
| 200 | OK | `GoalOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `GoalOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `contributions` | `integer` | yes |  |
| `currency` | `string` | yes |  |
| `months_left` | `integer` | yes | Months to the target date, for a goal with a future date that is not reached |
| `name` | `string` | yes |  |
| `overdue` | `boolean` | yes | The target date has passed and the goal is not reached |
| `percent` | `integer` | yes |  |
| `projected_date` | `string` |  | When the pace of the last months would reach the target |
| `reached` | `boolean` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `required_monthly_cents` | `integer (int64)` | yes | What to put aside each month to reach the target by its date |
| `saved_cents` | `integer (int64)` | yes |  |
| `target_cents` | `integer (int64)` | yes |  |
| `target_date` | `string` |  |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/goals/<uid>/archive"
```

## `POST /goals/{uid}/contributions`

Put an amount towards a goal.

| Field | Value |
| --- | --- |
| Operation | `goals.contributions.add` |
| Scope | `goals:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the goal |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `amount_cents` | `integer (int64)` |  | A non-zero whole number of cents; negative withdraws |
| `date` | `string` |  | The date of the contribution; today when left out At most 10 characters. |
| `note` | `string` |  | At most 200 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `ContributionOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `ContributionOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `amount_cents` | `integer (int64)` | yes |  |
| `created_at` | `string` | yes |  |
| `date` | `string` | yes |  |
| `goal_uid` | `string` | yes |  |
| `note` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"amount_cents":50000,"date":"","note":""}' \
  "$BASE/api/v1/goals/<uid>/contributions"
```

## `DELETE /goals/{uid}/contributions/{contribution_uid}`

Delete a contribution of a goal.

| Field | Value |
| --- | --- |
| Operation | `goals.contributions.delete` |
| Scope | `goals:write` |
| Kind | destructive |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `uid` | path | `string` | yes | The uid of the goal |
| `contribution_uid` | path | `string` | yes | The uid of the contribution |

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
  "$BASE/api/v1/goals/<uid>/contributions/<contribution_uid>"
```

## `POST /goals/{uid}/unarchive`

Bring an archived goal back.

| Field | Value |
| --- | --- |
| Operation | `goals.unarchive` |
| Scope | `goals:write` |
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
| 200 | OK | `GoalOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `GoalOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `contributions` | `integer` | yes |  |
| `currency` | `string` | yes |  |
| `months_left` | `integer` | yes | Months to the target date, for a goal with a future date that is not reached |
| `name` | `string` | yes |  |
| `overdue` | `boolean` | yes | The target date has passed and the goal is not reached |
| `percent` | `integer` | yes |  |
| `projected_date` | `string` |  | When the pace of the last months would reach the target |
| `reached` | `boolean` | yes |  |
| `remaining_cents` | `integer (int64)` | yes |  |
| `required_monthly_cents` | `integer (int64)` | yes | What to put aside each month to reach the target by its date |
| `saved_cents` | `integer (int64)` | yes |  |
| `target_cents` | `integer (int64)` | yes |  |
| `target_date` | `string` |  |  |
| `uid` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/goals/<uid>/unarchive"
```

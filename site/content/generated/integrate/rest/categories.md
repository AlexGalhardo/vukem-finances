---
title: "REST: Categories"
description: "The Categories endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

5 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /categories`

List the categories.

| Field | Value |
| --- | --- |
| Operation | `categories.list` |
| Scope | `categories:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<CategoryOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `icon` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `parent_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/categories"
```

## `POST /categories`

Create a category.

| Field | Value |
| --- | --- |
| Operation | `categories.create` |
| Scope | `categories:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `color` | `string` |  | A color name: slate, blue, teal, green, olive, amber, orange, red, pink or violet At most 20 characters. |
| `icon` | `string` |  | An icon name such as cart, home or wallet At most 20 characters. |
| `kind` | `string` |  | income or expense; a subcategory takes the kind of its parent At most 20 characters. |
| `name` | `string` | yes | At most 60 characters. |
| `parent_uid` | `string` |  | The parent category; a category nests one level At most 26 characters. |
| `uid` | `string` |  | Optional uid chosen by the client (a ULID); repeating the call returns the row already made At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `CategoryOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `icon` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `parent_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Groceries"}' \
  "$BASE/api/v1/categories"
```

## `GET /categories/{uid}`

Read a category.

| Field | Value |
| --- | --- |
| Operation | `categories.show` |
| Scope | `categories:read` |
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
| 200 | OK | `CategoryOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `icon` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `parent_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/categories/<uid>"
```

## `PATCH /categories/{uid}`

Change a category.

| Field | Value |
| --- | --- |
| Operation | `categories.update` |
| Scope | `categories:write` |
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
| `color` | `string` |  | At most 20 characters. |
| `icon` | `string` |  | At most 20 characters. |
| `name` | `string` |  | At most 60 characters. |
| `parent_uid` | `string` |  | At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `CategoryOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `icon` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `parent_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"color":"","icon":"","name":""}' \
  "$BASE/api/v1/categories/<uid>"
```

## `POST /categories/{uid}/archive`

Archive a category.

| Field | Value |
| --- | --- |
| Operation | `categories.archive` |
| Scope | `categories:write` |
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
| 200 | OK | `CategoryOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `CategoryOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `archived` | `boolean` | yes |  |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `icon` | `string` | yes |  |
| `kind` | `string` | yes |  |
| `name` | `string` | yes |  |
| `parent_uid` | `string` |  |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{}' \
  "$BASE/api/v1/categories/<uid>/archive"
```

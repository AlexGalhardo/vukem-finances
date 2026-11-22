---
title: "REST: Tags"
description: "The Tags endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

4 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /tags`

List the tags.

| Field | Value |
| --- | --- |
| Operation | `tags.list` |
| Scope | `tags:read` |
| Kind | read only |
| Idempotency-Key | not used |
| Editions | personal, saas |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<TagOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TagOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `name` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/tags"
```

## `POST /tags`

Create a tag.

| Field | Value |
| --- | --- |
| Operation | `tags.create` |
| Scope | `tags:write` |
| Kind | write |
| Idempotency-Key | honored |
| Editions | personal, saas |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `color` | `string` |  | At most 20 characters. |
| `name` | `string` | yes | At most 120 characters. |
| `uid` | `string` |  | Optional uid chosen by the client (a ULID); repeating the call returns the row already made At most 26 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `TagOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TagOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `name` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"work"}' \
  "$BASE/api/v1/tags"
```

## `PATCH /tags/{uid}`

Rename or recolor a tag.

| Field | Value |
| --- | --- |
| Operation | `tags.update` |
| Scope | `tags:write` |
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
| `name` | `string` |  | At most 120 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `TagOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `TagOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `color` | `string` | yes |  |
| `created_at` | `string` | yes |  |
| `name` | `string` | yes |  |
| `uid` | `string` | yes |  |
| `updated_at` | `string` | yes |  |
| `version` | `integer (int64)` | yes |  |

### Example

```sh
curl -X PATCH -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"color":"","name":""}' \
  "$BASE/api/v1/tags/<uid>"
```

## `DELETE /tags/{uid}`

Delete a tag and take it off its transactions.

| Field | Value |
| --- | --- |
| Operation | `tags.delete` |
| Scope | `tags:write` |
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
  "$BASE/api/v1/tags/<uid>"
```

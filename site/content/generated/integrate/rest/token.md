---
title: "REST: Token"
description: "The Token endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

1 operation. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /me`

The token, its scopes and the workspace it works in.

| Field | Value |
| --- | --- |
| Operation | `me.show` |
| Scope | none, any valid token |
| Kind | REST only |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `MeOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `MeOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `expires_at` | `string` |  |  |
| `role` | `string` | yes |  |
| `scopes` | `array of string` | yes |  |
| `token_name` | `string` | yes |  |
| `workspace` | `WorkspaceOut` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/me"
```

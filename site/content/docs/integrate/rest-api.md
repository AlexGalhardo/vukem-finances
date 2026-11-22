---
title: "REST API"
description: "How to call the vukem-finances JSON API: tokens, scopes, errors, pagination, idempotency, version checks and rate limits, with a reference for every endpoint."
---

A person or a program can read and write the data of one workspace over a JSON API, with a token that can do only what its scopes allow, for one workspace, until it expires or is revoked. The API exists in both editions and lives under `/api/v1`. The running app serves its OpenAPI description at `/api/v1/openapi.json`; the pages of this site are generated from the same document.

## Conventions

- Every request and response is JSON. Errors are `application/problem+json`.
- Money is an integer `*_cents` field in the currency of the workspace, never a decimal string. A float is refused.
- A date is `YYYY-MM-DD` in the time zone of the workspace. A timestamp is RFC 3339 in UTC.
- The id of a row on the wire is its `uid`, a ULID. A row carries a `version` that grows on every change.
- Field names are `snake_case`. Unknown fields in a request body are ignored.

## Tokens

Create a token in the app, on the Tokens page, for the workspace you are in. Choose a name, the scopes and an expiry of 30, 90 or 365 days. A token belongs to one workspace and to the person who created it.

- The secret looks like `vf_<id>.<secret>`. It is shown once, when the token is created, and only a hash is stored: if you lose it, create a new one and revoke the old one.
- Send it as `Authorization: Bearer <token>`. Nothing else authenticates a request to `/api/v1`: cookies are ignored.
- A person holds at most 20 live tokens per workspace.
- Revoking is immediate. A token also stops working when its person leaves or is removed from the workspace, and when the person resets the password.

## Scopes

A scope is a ceiling, never a grant. What a request may do is the scope of the token intersected with the role of the person in the workspace now, not when the token was made: a viewer's `transactions:write` token still cannot write.

| Scope | Allows | Role needed |
| --- | --- | --- |
| `accounts:read` | list and read accounts with their balances | any |
| `accounts:write` | create, edit and archive accounts | owner, admin |
| `categories:read` | list and read categories | any |
| `categories:write` | create, edit and archive categories | owner, admin |
| `tags:read` | list tags | any |
| `tags:write` | create, rename and delete tags | owner, admin |
| `transactions:read` | list and read transactions | any |
| `transactions:write` | create, edit and delete transactions | owner, admin, member |
| `transfers:write` | create and delete transfers | owner, admin, member |
| `budgets:read` | list budgets with what the month spent | any |
| `budgets:write` | set and remove budgets | owner, admin |
| `summary:read` | the monthly summary | any |
| `cards:read` | list and read credit cards | any |
| `cards:write` | create, edit, archive and unarchive cards | owner, admin |
| `statements:read` | list and read the statements of a card | any |
| `statements:write` | pay a statement from a bank account | owner, admin, member |
| `recurrences:read` | list recurring rules | any |
| `recurrences:write` | create, edit and delete recurring rules | owner, admin, member |
| `goals:read` | list savings goals with their progress | any |
| `goals:write` | manage goals and their contributions | owner, admin; contributions also member |
| `reports:read` | the report views | any |
| `export:read` | download transactions as CSV, or the whole workspace as JSON | any for the CSV; owner, admin for the JSON |
| `sync` | the [sync protocol](/docs/integrate/sync-protocol/) | any to read; each pushed row needs the role of its entity |

Members, invitations, billing, workspace settings and the AI layer have no scope and are not in the API. `GET /api/v1/me` needs a valid token and no scope.

## Errors

The order of the checks never leaks what exists: 401 for a missing, unknown, expired or revoked token; 404 for a person who is no longer a member; 429 over a rate limit; 403 for a missing scope or a role that cannot do it; then 404 for a row of another workspace, the same answer as a row that does not exist.

```json
{
  "type": "https://httpstatuses.com/404",
  "title": "Not Found",
  "status": 404,
  "detail": "Not found.",
  "code": "not_found",
  "request_id": "..."
}
```

`code` is stable and `detail` is for people. A refused body is a 422 with an `errors` map of field to messages.

| Status | `code` | When |
| --- | --- | --- |
| 400 | `bad_request` | a malformed request or an unknown query value |
| 401 | `unauthorized` | the token is missing, unknown, expired or revoked |
| 402 | `plan_limit` | the Free plan stops here |
| 403 | `missing_scope`, `forbidden` | the token lacks the scope, or the role cannot do it |
| 404 | `not_found` | no such row in this workspace |
| 409 | `conflict`, `version_conflict` | a rule, or a stale `If-Match` |
| 422 | `validation_failed` | the body breaks a rule |
| 429 | `rate_limited` | over a rate limit |
| 503 | `unavailable` | retry later |

## Pagination and filters

`GET /api/v1/transactions` pages with `?page=N`, from 1. A page holds 50 rows, newest first, and the answer carries `meta` (`page`, `per_page`, `total`, `last_page`) and `links`. The other lists are short and come whole in `{"data": [...]}`. The transaction list takes `from`, `to`, `account`, `category`, `tag`, `kind`, `q`, `min_cents` and `max_cents`.

## Safe retries

- **Your own uid.** `POST` on accounts, categories, tags and transactions accepts an optional `uid` (a ULID). Repeating the call with the same `uid` and the same row returns the row already made, with status 200, and creates nothing.
- **`Idempotency-Key`.** `POST` and `PATCH` accept `Idempotency-Key: <1 to 255 printable characters>`. The first request runs and its answer is kept for 24 hours; the same key with the same request replays that answer with `Idempotent-Replayed: true`. Keys are per token.
- **`If-Match`.** `PATCH` and `DELETE` accept `If-Match: <version>`. A stale version is a 409 with `version_conflict` and the current row in `current`; nothing changes.

## Rate limits

Per token: 120 reads and 60 writes a minute, counted separately. Over the limit the answer is 429 with `Retry-After` and the `RateLimit-Limit`, `RateLimit-Remaining` and `RateLimit-Reset` headers. Failed token lookups are limited per address, 60 a minute.

## Versioning

The version is in the path. Within `/api/v1` a change never removes or renames a field and never changes what a field means; new optional fields, operations and scopes can appear, so a client ignores what it does not know. A breaking change ships as `/api/v2`. Something that will go away is marked deprecated in the document and answers with `Deprecation` and `Sunset` headers at least six months before it is removed. CORS is off: call the API from a server, a script or the mobile app.

## Examples

```sh
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/accounts"
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/transactions?from=2026-11-01&kind=expense"
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: 6f1c0d1e-0001" \
  -d '{"account_uid":"01J...","kind":"expense","amount_cents":1250,"date":"2026-11-21","description":"Lunch"}' \
  "$BASE/api/v1/transactions"
```

## Reference by area

[Accounts](/docs/integrate/rest/accounts/), [Budgets](/docs/integrate/rest/budgets/), [Cards](/docs/integrate/rest/cards/), [Categories](/docs/integrate/rest/categories/), [Export](/docs/integrate/rest/export/), [Goals](/docs/integrate/rest/goals/), [Recurrences](/docs/integrate/rest/recurrences/), [Reports](/docs/integrate/rest/reports/), [Statements](/docs/integrate/rest/statements/), [Summary](/docs/integrate/rest/summary/), [Sync](/docs/integrate/rest/sync/), [Tags](/docs/integrate/rest/tags/), [Token](/docs/integrate/rest/token/), [Transactions](/docs/integrate/rest/transactions/) and [Transfers](/docs/integrate/rest/transfers/).

The same operations are available as [commands](/docs/integrate/cli/) and as [MCP tools](/docs/integrate/mcp/tools/).

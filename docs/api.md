# Public API

A person or a program can read and write the data of one workspace over a JSON
API with a token that can do only what its scopes allow, for one workspace,
until it expires or is revoked. The API is described by
[`openapi.json`](openapi.json), which is checked in and served by the running
app at `/api/v1/openapi.json`. The app has no reference page: the static docs
site renders the reference from that file (decisions D-019 and D-021).

The API exists in both editions (D-003). In `saas` a Free workspace has the
same API as a Pro one, with the same limits as the web app.

The code is in `app/api/`. The handlers are thin: they bind the input, call a
service of `app/finance` or `app/workspaces` and return the output, so a rule
(amount parsing, archived accounts, plan limits, tags, transfers) is the same
on the web and on the API. Each operation is declared once, with its scope, in
`app/api/operations.go` (decision D-017).

## Conventions

- Base path `/api/v1`. Every request and response is JSON
  (`Content-Type: application/json`). Errors are `application/problem+json`.
- Money is an integer `*_cents` field in the currency of the workspace
  (`GET /api/v1/me` shows it), never a decimal string. A float is refused.
- A date is `YYYY-MM-DD` in the timezone of the workspace. A timestamp is RFC
  3339 in UTC.
- The id of a row on the wire is its `uid` (a ULID). The numeric ids of the web
  app never appear.
- A row carries a `version` that grows on every change.
- Field names are `snake_case`. Unknown fields in a request body are ignored.

## Tokens

Create a token in the app, on the Tokens page, for the workspace you are in.
Choose a name, the scopes and an expiry of 30, 90 or 365 days (no token lives
longer than 365 days). A token is for one workspace and for the person who
created it.

- The secret looks like `vf_<id>.<secret>`. The `vf_` prefix lets secret
  scanners find a leaked token. It is shown once, when the token is created.
  Only a SHA-256 hash is stored, so nobody can read it again: create a new
  token and revoke the old one.
- Send it as `Authorization: Bearer <token>`. Nothing else authenticates a
  request to `/api/v1`: cookies are ignored and there is no CSRF check. A bearer
  token is not accepted on the web routes.
- A person holds at most 20 live tokens per workspace.
- Revoking is immediate. A token also stops working when its person leaves or
  is removed from the workspace, and when the person resets the password.

## Scopes

A scope is a ceiling, never a grant. What a request may do is the scope of the
token intersected with the role of the person in the workspace now (not when the
token was made): a viewer's `transactions:write` token still cannot write, and a
member demoted to viewer loses writing at once. Members, invitations, billing,
workspace settings and the AI layer have no scope and are not in the API.

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
| `transfers:write` | create and delete transfers between accounts | owner, admin, member |
| `budgets:read` | list budgets with what the month spent | any |
| `budgets:write` | set and remove budgets | owner, admin |
| `summary:read` | the monthly summary | any |

`GET /api/v1/me` needs a valid token and no scope. The scope of every operation
is in the document as `x-required-scope`.

## Requests and answers

The order of the checks never leaks what exists:

1. `401` for a missing, unknown, expired or revoked token.
2. `404` for everything of another workspace, and for a token whose person is no
   longer in its workspace.
3. `429` over a rate limit.
4. `403` for a scope the token lacks (`code: missing_scope`) or a role that
   cannot do it (`code: forbidden`).
5. `404` for a row of another workspace: the same answer as a row that does not
   exist.

An error looks like this (RFC 9457):

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

`code` is stable, `detail` is for people. A refused body is `422` with `errors`
(field to messages) and `code: validation_failed`.

| Status | `code` | When |
| --- | --- | --- |
| 400 | `bad_request` | a malformed request, an unknown query value |
| 401 | `unauthorized` | the token is missing, unknown, expired or revoked |
| 402 | `plan_limit` | the Free plan stops here (accounts, transactions a month, budgets) |
| 403 | `missing_scope`, `forbidden` | the token lacks the scope, or the role cannot do it |
| 404 | `not_found` | no such row in this workspace |
| 409 | `conflict`, `version_conflict` | a rule (a name taken, an archived parent), or a stale `If-Match` |
| 422 | `validation_failed` | the body breaks a rule |
| 429 | `rate_limited` | over a rate limit |
| 503 | `unavailable` | the server cannot answer now; retry |

### Pagination

`GET /api/v1/transactions` pages with `?page=N` (1-based). A page holds 50
rows, newest first, in a stable order, and the answer carries `meta`
(`page`, `per_page`, `total`, `last_page`) and `links` (`self`, `next`, `prev`).
The other lists are short and come whole in `{"data": [...]}`.

### Filters

`GET /api/v1/transactions` takes `from`, `to` (dates), `account`, `category`
and `tag` (uids), `kind` (`income` or `expense`), `q` (text of the description,
at most 200 characters), `min_cents` and `max_cents` (the absolute amount).

### Creating with your own uid

`POST` on accounts, categories, tags and transactions accepts an optional `uid`
(26 upper-case Crockford characters, a ULID, at most a day ahead of the server
clock). Repeating the call with the same `uid` and the same row returns the row
already made (`200`, not `201`) and creates nothing, so a client can retry
safely. A `uid` that belongs to a deleted row, or to a row that differs, is
`409`.

### Idempotency-Key

`POST` and `PATCH` accept `Idempotency-Key: <1 to 255 printable characters>`.
The first request runs and its answer is kept for 24 hours; the same key with the
same request replays that answer with `Idempotent-Replayed: true`. The same key
while the first request still runs is `409`. The same key with a different
request is `422`. Keys are per token.

### If-Match

`PATCH` and `DELETE` accept `If-Match: <version>` (the `version` of the row, with
or without quotes). A stale version is `409` with `code: version_conflict` and
the current row in `current`; nothing changes. Without the header the change
applies to whatever the row is now.

## Rate limits

Per token: 120 reads (`GET`) and 60 writes a minute, counted separately. Over
it, the answer is `429` with `Retry-After` (seconds) and `RateLimit-Limit`,
`RateLimit-Remaining` and `RateLimit-Reset`. Per IP address, before the token
is checked: 60 failed token lookups a minute, then `429`.

## Versioning and deprecation

The version is in the path. Within `/api/v1` a change never removes or renames a
field, never makes an optional field required and never changes what a field
means; new optional fields, new operations and new scopes can appear at any
time, so a client ignores what it does not know. A breaking change ships as
`/api/v2`. An operation or field that will go away is marked `deprecated` in the
document and answers with the `Deprecation` and `Sunset` headers at least six
months before it is removed.

## CORS

Off. A browser app on another origin cannot call the API in v1; call it from a
server, a script or a mobile app.

## The document and the client

- `docs/openapi.json` is generated from the routes (`UPDATE_OPENAPI=1 go test
  ./tests/... -run TestOpenAPISnapshot` rewrites it; a test fails when it is
  stale). It uses the `bearer` security scheme and `x-required-scope`.
- `clients/typescript/api.ts` is generated from it (`UPDATE_OPENAPI=1 go test
  ./tests/... -run TestTypeScriptClientIsFresh`; a test fails when it is stale).

## Examples

Replace `$TOKEN` with the token and `$BASE` with the address of the server.

```sh
# accounts:read
curl -H "Authorization: Bearer $TOKEN" $BASE/api/v1/accounts

# transactions:read
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/transactions?from=2026-11-01&kind=expense"

# transactions:write
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: 6f1c0d1e-0001" \
  -d '{"account_uid":"01J...","kind":"expense","amount_cents":1250,"date":"2026-11-21","description":"Lunch","tags":["work"]}' \
  $BASE/api/v1/transactions

# transactions:write, with a version check
curl -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -H "If-Match: 3" \
  -d '{"account_uid":"01J...","kind":"expense","amount_cents":1300,"date":"2026-11-21"}' \
  $BASE/api/v1/transactions/01J...

# transfers:write
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"from_account_uid":"01J...","to_account_uid":"01J...","amount_cents":50000,"date":"2026-11-21"}' \
  $BASE/api/v1/transfers

# budgets:read
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/budgets?month=2026-11"

# summary:read
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/summary?month=2026-11"
```

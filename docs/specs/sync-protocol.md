# Sync protocol

This is the language-neutral specification of the sync protocol between a vukem-finances server (personal or SaaS) and a phone (decisions D-004, D-005, D-008). It builds on the [sync data model](sync-data-model.md): stable `uid`s, `version`, soft deletes, the `changes` log with `seq`, the entity registry, retention and `compacted_through_seq`, and the conflict rules of `changes.Merge`. This document does not change them.

The server is `app/sync`; the routes are in `app/api/sync.go`. The contract that both the server and the mobile client engine pass is the set of scenario files in `testdata/sync/` (see [Scenarios](#scenarios)).

## Overview

A phone keeps a full local copy of its workspace. It does four things:

1. **Registers** a device once (`POST /devices`).
2. **Pulls** what changed on the server since its cursor (`GET /pull`), in pages.
3. **Pushes** what it did offline (`POST /push`), in batches, and learns for each mutation whether it was applied, merged, a duplicate or rejected.
4. **Resyncs** from scratch (`GET /snapshot`, then `GET /pull`) when the server says its cursor is too old.

Everything is state based: the log says which row changed, and a pull sends the row as it is now. A client that applies the same page twice ends in the same state.

## Common rules

All routes are under `/api/v1/sync` and need:

- a bearer token with the scope `sync` (the scope is a ceiling: the person behind the token gets only the permissions of their role, checked per row on push, see [Roles](#roles));
- the header `X-Sync-Protocol: 1`. A missing header is `400 bad_request`; a version this server does not speak is `426 protocol_unsupported` with the supported range in `min` and `max`;
- except for `/info` and `/devices`, the header `X-Device-Id: <device_id>`. An unknown device, a revoked one, one registered by another token or by a person of another workspace is the same answer: `403 device_revoked`. The client stops syncing and tells the person.

Errors are `application/problem+json` with a stable `code` (see [Errors](#errors)). Money is an integer number of cents (`*_cents`), a date is `YYYY-MM-DD`, a timestamp is RFC 3339 in UTC, and a row is named by its `uid`; the integer ids of the database never leave the server. The server owns `created_at`, `updated_at`, `version` and `workspace_id`: a client never sends them (see `unknown_field`), and the client clock is never used to order or resolve anything.

## Info

`GET /info` answers what this server speaks and knows:

```json
{
  "protocol": { "min": 1, "max": 1 },
  "server_time": "2026-11-21T12:00:00Z",
  "workspace": { "uid": "7", "currency": "USD", "timezone": "America/Sao_Paulo" },
  "entities": [
    {
      "name": "account",
      "fields": [{ "name": "name", "type": "text", "writable": true, "required": true, "min": 1, "max": 80 }],
      "refs": []
    }
  ],
  "limits": { "max_mutations": 200, "max_body_bytes": 1048576, "pull_max_limit": 1000 }
}
```

`entities` follows the registry of the server, parents first. Each field has a `type`: `text`, `enum` (with `enum`, and `blank` when the empty string is allowed), `cents`, `int`, `bool`, `date`, `timestamp`, `uid`, `uid_list` or `ref` (with `ref`, the entity it points at; references travel as `<name>_uid`, for example `account_uid`, `parent_uid` and `payment_account_uid`). `nullable` and `required` say what a create may leave out. A client **ignores** entities and fields it does not know (a newer server); a field a client sends that the server does not know is rejected for that mutation as `unknown_field`.

## Devices

`POST /devices` with `{ "name", "platform" }` (`platform` is `ios`, `android`, `web` or `other`; the default is `other`) registers a device for the calling token and answers `201` with `device_id` (a ULID), `name`, `platform`, `created_at`, `last_seen_at`, `last_pull_seq` and `revoked_at`. `GET /devices` lists the devices the caller registered in the workspace.

A person revokes a device on the web page Devices (it keeps its local data until the person clears it on the device). Revoking an API token revokes the devices it registered. A revoked device gets `403 device_revoked` on every call.

## Pairing

A phone gets its token with a short code instead of a typed token. On the web, the person opens the Devices page and presses "Add device": the server stores the hash of an 8 character code (alphabet `23456789ABCDEFGHJKLMNPQRSTUVWXYZ`, about 40 bits, shown as `XXXX-XXXX`) for the person and the workspace, valid for 10 minutes and for one use. A person holds at most 5 live codes. The page shows the code and the server address once.

`POST /api/v1/sync/pair` takes no token and no `X-Device-Id`; the code is the credential. It needs `X-Sync-Protocol: 1`.

```json
{ "code": "7K2M-9QXA", "device_name": "Ana's phone", "platform": "android" }
```

The code is case-insensitive and a space or a dash is ignored. The server consumes the code with one conditional update (`used_at` still null, not expired), so of any number of concurrent attempts exactly one pairs. It then issues a token with only the scope `sync`, bound to the workspace of the code, living 365 days and named "Phone" followed by the device name, registers the device for that token and answers `201`:

```json
{ "token": "...", "device_id": "...", "workspace": { "uid": "1", "name": "Ana household", "currency": "USD", "timezone": "UTC" }, "server_time": "2026-11-24T12:00:00Z" }
```

The token has the role ceiling of the person who made the code: the role is read at every request. The token shows on the person's token list and is revoked there or with the device. A code that is unknown, used, expired or whose person left the workspace gets the same `422 invalid_pairing_code`. The limits are 5 attempts a minute per address (`429 rate_limited`) and 200 failed attempts an hour for the whole server. The code and the token are never logged, and `Cache-Control: no-store` is set.

## Pull

`GET /pull?cursor=<seq>&limit=<n>` (the cursor starts at `0`; `limit` defaults to 500 and is at most 1000) answers:

```json
{
  "changes": [
    { "seq": 41, "entity": "account", "uid": "...", "op": "upsert", "version": 3, "row": { "uid": "...", "version": 3, "name": "Wallet" } },
    { "seq": 44, "entity": "transaction", "uid": "...", "op": "delete", "version": 2, "row": null }
  ],
  "next_cursor": 44,
  "has_more": false,
  "server_time": "2026-11-21T12:00:00Z"
}
```

- The server reads the log after `cursor` for the workspace in `seq` order, up to `limit` entries, **groups them by `(entity, uid)`** and sends the current state of each row once, with the `seq` of its newest change in the page. A row that is deleted (a tombstone) has `op: "delete"` and `row: null`; a row that compaction purged is also sent as a delete.
- Within a page, items are ordered by entity dependency (accounts, categories, tags, goals first, then the entities that point at them), then by `seq`. A reference may still point to a row that arrives in a later page: the client **must not enforce foreign keys while applying a pull** and checks references when the pull is finished.
- `next_cursor` is the `seq` of the last change read, never a clock or a count. The client stores it as is and sends it back. When there is nothing new it equals the cursor sent and `has_more` is `false`.
- A page never splits the two legs of a transfer: when one leg is in the page and the other one changed after `next_cursor`, the other leg is sent in the same page (and again in a later one; applying it twice is harmless). Its `seq` can then be higher than `next_cursor`.
- If `cursor` is below the `compacted_through_seq` of the workspace, the answer is `410 resync_required`: download a snapshot.

**A pull never skips a late commit.** The log and the rows are read in one read-only database transaction, a single consistent snapshot. SQLite hands out `seq` inside the one write transaction and there is one writer at a time, so `seq` order is commit order: nothing commits "behind" a cursor, and a writer that is still in its transaction when a pull runs is simply on the next page. `TestSyncPullNeverSkipsALateCommit` and `TestSyncPullUnderConcurrentWriters` prove it.

## Snapshot

`GET /snapshot?entity=<name>&after=<uid>&limit=<n>&snapshot_id=<id>` answers the live rows of one entity in `uid` order (limit defaults to 500, at most 1000):

```json
{ "snapshot_id": "s1.7.1042", "snapshot_cursor": 1042, "entity": "account", "rows": [], "next_after": "", "has_more": false }
```

The first call has no `snapshot_id`; the server fixes `snapshot_cursor` (the newest `seq`, read in the same transaction as the rows) and returns the id that carries it. The client sends the id back on every later call, for every entity, and `after` is the `next_after` of the previous page. When every entity is downloaded the client pulls from `snapshot_cursor`: a change made while the snapshot ran is applied twice at worst, because a pull is idempotent. Deleted rows are not in a snapshot. An unknown entity is `400 unknown_entity`; an id the server did not make for this workspace is `400 invalid_snapshot`.

## Push

`POST /push` with at most **200 mutations** and **1 MiB** (`413` above either):

```json
{
  "device_id": "optional, must equal X-Device-Id",
  "mutations": [
    { "op_id": "<client ULID>", "entity": "transaction", "uid": "<client ULID>", "op": "upsert", "base_seq": 41, "fields": { "description": "dinner" } }
  ]
}
```

- `op` is `upsert` (create or edit), `delete` or `restore`. `base_seq` is the newest `seq` the client saw for that row (0 for a row it made). `fields` holds only the fields that changed, and nothing for a `delete`.
- The batch runs in **one database transaction**, mutations ordered by entity dependency (parents first, deletes last), **each inside a savepoint**: a rejected mutation changes nothing and does not stop the others. The answer is `{ "results": [...] }` with one result per mutation **in request order**.

For each mutation the server, in this order: checks `op_id`, `entity`, `uid` (a ULID at most a day in the future) and `op`; checks the **role** for the entity; validates **every field against the registry** (the fields it knows, their types, lengths, enum values, ISO dates, the money cap); resolves each `*_uid` **in the token's workspace only**; applies the rules between fields; checks the **plan quota** inside the transaction; and calls `changes.Merge`.

### Result

```json
{
  "op_id": "...", "status": "applied", "entity": "transaction", "uid": "...",
  "row": { }, "version": 4, "seq": 57,
  "conflicts": [{ "field": "amount_cents", "kind": "field", "lost_value": 700, "applied_value": 800 }],
  "effects": [{ "entity": "transaction", "uid": "...", "op": "delete", "seq": 58 }],
  "replaced_by_uid": "", "error": { "code": "", "message": "", "field": "" }
}
```

| `status` | Meaning |
| --- | --- |
| `applied` | the mutation was written as sent |
| `merged` | written, and either another device changed the row after `base_seq` (other fields merged) or a field it changed was changed on both sides (see `conflicts`) |
| `duplicate` | nothing to write: the row already holds these values, a delete or restore was already done, or the create is a recurrence occurrence that exists under another uid (`replaced_by_uid`) |
| `rejected` | refused; `error.code` says why and nothing was written |

`row` is the row as it is now (`null` for a deleted row), `version` its version, `seq` the newest `seq` of the row. `conflicts` lists the values that lost a merge (kept 30 days in `sync_conflicts`); a client tells the person. `effects` lists other rows this mutation changed (the other leg of a transfer, the transactions of a deleted account), at most 100, in `seq` order; they also come through pull.

### Exactly once

`op_id` is a client ULID. The result of every mutation, rejections included, is stored with the `op_id` for **30 days** (`sync_ops`, unique per workspace). A retried `op_id` returns the **stored result unchanged and changes nothing**, so a client that lost a response resends the whole batch. The same `op_id` twice in one batch runs once. After 30 days the result is pruned by the job `sync_ops_prune`; a retry that late is a new mutation, and an upsert with equal values is a `duplicate`.

### Merging

The seven rules of [the data model](sync-data-model.md#conflict-policy) apply unchanged: a mutation names its base and only its changed fields; fields no later change touched apply cleanly; the same field changed on both sides goes to the later arrival and the loser is told; **delete wins over an edit** (the edit is `rejected` with `edited_deleted_row` and a conflict of that kind; only `restore` brings the row back); creating an existing `uid` with equal fields is a retry; a missing or deleted parent is `unknown_reference` or `parent_deleted`. An edit of a row the server has never seen (no `base_seq`) is a create when the `uid` does not exist, and an edit with `base_seq = 0` (every field in the conflict window) when it does.

`recurrences.generated_through` is a watermark: the merged value is the **later valid date**, and it is never a conflict. A value that is not an ISO date is `invalid_date`; a date after today in the workspace time zone is ignored, so a far-future value cannot stop a rule and a stale device cannot move it back.

### Special cases

1. **Transfers.** A transaction with a `transfer_uid` has a partner. At the end of the batch exactly two live legs must hold the `transfer_uid` (the ones stored plus the ones the batch creates, edits or restores): otherwise those mutations are all rejected `transfer_incomplete`. The legs must be one expense and one income of the same amount and date on two accounts, with no category: otherwise `transfer_invalid`. Deleting one leg deletes the other (both changes come back, one in `effects`); restoring one brings the pair back.
2. **Natural key of a recurrence occurrence.** A create whose `(recurrence_uid, occurrence_on)` already belongs to another row of the workspace, deleted rows included, is answered `duplicate` with `replaced_by_uid`: the client adopts that uid and drops its own. An occurrence the person deleted is never made again.
3. **Account delete.** Deleting an account soft-deletes its transactions, card and rules in the same transaction, each change logged; they reach the other devices through pull. Restoring the account within 30 days brings back what was deleted with it.
4. **Restore window.** A `restore` is accepted for 30 days after the delete; later it is `restore_window_expired`. A restore of a row whose parent is still deleted is `parent_deleted`; over a live row with the same name `duplicate_name`.
5. **Plan quota.** On the Free plan a create that would pass the monthly transaction quota, the one live account, or budgets (a Pro feature) is `quota_exceeded`, checked after the workspace lock inside the batch transaction so two pushes cannot both pass. It rejects that mutation only.
6. **Names.** Two devices that create a category or a tag with the same name offline: the second is `duplicate_name`. The client renames or adopts the existing row after its next pull.

### Roles

The role of the person behind the token, read at the request, decides per row: `transaction`, `recurrence` and `goal_contribution` need the ability to record (owner, admin, member); everything else (accounts, categories, tags, budgets, cards, goals) needs data management (owner, admin). A viewer syncs read-only: pull and snapshot work, every mutation is `rejected` with `forbidden` and nothing is written.

## Errors

Problem codes of a whole request: `bad_request` (400), `unauthorized` (401), `missing_scope` and `forbidden` (403), `device_revoked` (403), `unknown_entity`, `invalid_snapshot` (400), `batch_too_large` (413), `resync_required` (410), `workspace_deleted` (410: the workspace waits to be purged or was purged; the client tells the person and erases that workspace locally), `protocol_unsupported` (426), `rate_limited` (429, with `Retry-After`), `unavailable` (503).

Codes of one mutation (`error.code` of a `rejected` result):

| Code | Meaning |
| --- | --- |
| `unknown_entity`, `invalid_op`, `invalid_uid` | the entity, op, `op_id`, `uid` or a uid field is not valid |
| `unknown_field` | a field the registry does not have, including every server-generated one (`version`, `workspace_id`, `created_at`, `updated_at`, `deleted_at`, `id`) |
| `missing_field`, `invalid_field` | a required field is missing, or a value has the wrong type, length, enum value or null |
| `invalid_amount` | money is not a whole number of cents, is beyond +-9007199254740991, is not read back unchanged by `finance.ParseAmount`, or breaks the minimum of the field (an expense is at least 1 cent) |
| `invalid_date` | not a strict `YYYY-MM-DD` date |
| `unknown_reference`, `parent_deleted` | a `*_uid` or a tag uid names nothing in the workspace (another workspace's uid reads the same), or a deleted row |
| `invalid_state` | a rule between fields: a category of another kind, an archived account or category, a budget on an income category, a nested parent, a card on a plain account |
| `forbidden` | the role cannot change that entity |
| `quota_exceeded` | the plan quota |
| `transfer_incomplete`, `transfer_invalid` | the legs of a transfer |
| `duplicate_name`, `duplicate_key` | a live row has the same name (category, tag) or unique key |
| `edited_deleted_row` | the row was deleted: the edit was dropped (with a conflict) |
| `restore_window_expired`, `not_restorable` | the restore is too late, or the row is not in the trash |
| `unknown_row` | a delete or restore of a uid the workspace does not have |

## Limits and abuse

- Pull and snapshot: 60 per minute per device. Push: 30 per minute per device. A device may not push more than 5000 mutations per hour. Each answers `429 rate_limited` with `Retry-After`.
- Body 1 MiB, 200 mutations per push, text lengths from the registry (`/info` lists them).
- The API limits of the token (120 reads and 60 writes a minute) apply as well.
- Each pull and push logs the workspace, device, counts and duration, never row contents or tokens.
- Compaction (see the data model) keeps the log bounded; `sync_ops` and `sync_conflicts` are pruned daily.

## Scenarios

`testdata/sync/*.json` are the conformance scenarios. Each file is one scenario run against a fresh workspace:

```json
{
  "name": "merge-different-fields",
  "description": "...",
  "workspace": { "plan": "pro", "devices": { "a": "owner", "b": "owner" } },
  "steps": [
    { "device": "a", "action": "push", "input": { "mutations": [] }, "expect": { "status": 200, "results": [] } },
    { "device": "b", "action": "pull", "input": { "from": "start" }, "expect": { "contains": [] } }
  ],
  "final": { "devices_equal": ["a", "b"], "server": { "account": { "<uid>": { "name": "Wallet" } } }, "absent": ["<uid>"] }
}
```

- `workspace.plan` is `pro` or `free`; `devices` maps a device name to the role of its person (`owner`, `admin`, `member`, `viewer`).
- A step has a `device`, an `action` (`push`, `pull`, `snapshot` or `admin`), an `input` and an `expect`. `push` input is `{ "mutations": [...] }`; a `base_seq` of `"$seen"` means the newest `seq` that device has seen of that row (from its own results and its pulls). `pull` input is `{ "limit", "from": "start" }`: it reads every page from the device cursor (or from 0) and applies it to the device replica. `snapshot` rebuilds the replica and the cursor. `admin` is something only the server can do: `revoke_device`, `compact` (the log is compacted through its newest change) and `age_tombstones` (every delete is 31 days old).
- `expect` is matched as a subset: objects by the keys listed, arrays by length and position, `null` matches null or a missing key. `status` (default 200) and `code` check the HTTP answer; `results` is the list of push results; for a pull, `changes` is the ordered list, `contains` items are found by `uid` in any order, `absent_uids` must not be there and `parents_first` checks the order; `replica_contains` lists uids a snapshot must hold.
- `final.devices_equal` names devices that sync to the end (resyncing when told) and must then hold exactly the live rows of the server; `final.server` lists rows (a subset) the server must hold and `final.absent` uids it must not.
- Every uid and `op_id` in a file is fixed, so a run is repeatable and a client can resend an `op_id`.

The scenarios are: first sync, two devices editing different fields, the same field (last writer wins), delete versus edit, the same category name on two devices, a retry of a lost response, resync after compaction, transfer legs, a recurrence duplicate, quota exceeded, a viewer's push, a revoked device and the tombstone round trip. The Go runner is `tests/sync_scenarios_test.go`; epic 20 runs the same files against the client engine.

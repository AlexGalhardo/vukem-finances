---
title: "Sync protocol"
description: "How a device keeps a full local copy of a workspace in step with a server: registration, pairing, pull, snapshot, push, merging and errors."
---

This is a readable guide to the protocol between a vukem-finances server and a phone. The complete, language-neutral specification is `docs/specs/sync-protocol.md` in the repository, and the scenario files in `testdata/sync/` are the contract that the server and the mobile client both pass. For the person using it, see [mobile sync and pairing](/docs/use/mobile-sync/).

## Overview

A phone keeps a full local copy of its workspace. It does four things:

1. **Registers** a device once.
2. **Pulls** what changed on the server since its cursor, in pages.
3. **Pushes** what it did offline, in batches, and learns for each mutation whether it was applied, merged, a duplicate or rejected.
4. **Resyncs** from scratch with a snapshot when the server says its cursor is too old.

Everything is state based: the change log says which row changed, and a pull sends the row as it is now. A client that applies the same page twice ends in the same state.

## Common rules

All routes are under `/api/v1/sync/...` and need:

- a bearer token with the scope `sync`. The scope is a ceiling: the permissions of the person's role are checked for each pushed row;
- the header `X-Sync-Protocol: 1`. A missing header is a 400, and a version the server does not speak is a 426 with the supported range;
- except for `/info` and `/devices`, the header `X-Device-Id`. An unknown or revoked device is a 403 `device_revoked`, and the client stops syncing and tells the person.

Money is an integer number of cents, a date is `YYYY-MM-DD`, a timestamp is RFC 3339 in UTC, and a row is named by its `uid`. The server owns `created_at`, `updated_at`, `version` and `workspace_id`: a client never sends them, and the client clock is never used to resolve anything. Sync is REST only: there is no command or MCP tool for it. The routes are in the [sync endpoints](/docs/integrate/rest/sync/).

## Info and devices

`GET /info` answers the protocol range the server speaks, the server time, the workspace, the entities with their fields and types, and the limits. A client ignores entities and fields it does not know. `POST /devices` registers a device with a name and a platform (`ios`, `android`, `web` or `other`) and answers a `device_id`. A person revokes a device on the Devices page; revoking an API token revokes the devices it registered.

## Pairing

On the web, the Devices page makes an 8 character code, shown as `XXXX-XXXX`, valid for 10 minutes and one use. A person holds at most 5 live codes. `POST /api/v1/sync/pair` takes no token: the code is the credential. The server consumes it with one conditional update, so of any number of concurrent attempts exactly one pairs. It then issues a token with only the `sync` scope, living 365 days, registers the device and answers with the token, the device id and the workspace. A code that is unknown, used or expired gets the same 422 `invalid_pairing_code`. Attempts are limited to 5 a minute per address.

## Pull

`GET /pull?cursor=<seq>&limit=<n>` starts at cursor 0, with a default limit of 500 and a maximum of 1000. The server reads the change log after the cursor, groups the changes by entity and uid, and sends the current state of each row once. A deleted row has `op: "delete"` and no row.

- Items are ordered so parents come first, but a reference may point at a row in a later page: the client must not enforce foreign keys while applying a pull and checks references when the pull is finished.
- `next_cursor` is the sequence number of the last change read. The client stores it as is. A page never splits the two legs of a transfer.
- A cursor older than the compaction point of the workspace is a 410 `resync_required`.
- A pull never skips a late commit: it reads one consistent snapshot, and the log order is the commit order.

## Snapshot

`GET /snapshot` returns the live rows of one entity at a time, in uid order, with a `snapshot_id` that fixes the cursor of the whole download. When every entity is downloaded, the client pulls from the snapshot cursor, and a change made while the snapshot ran is applied twice at worst, which is harmless.

## Push

`POST /push` takes at most 200 mutations and 1 MiB. Each mutation has an `op_id` (a client ULID), the entity, the uid, an operation (`upsert`, `delete` or `restore`), the `base_seq` the client saw and only the fields that changed.

The batch runs in one database transaction with a savepoint for each mutation, so a rejected mutation changes nothing and does not stop the others. The answer has one result per mutation in request order, with a status:

| Status | Meaning |
| --- | --- |
| `applied` | written as sent |
| `merged` | written, and another device changed the row after `base_seq`, or a field changed on both sides |
| `duplicate` | nothing to write: the row already holds these values |
| `rejected` | refused with an error code, and nothing was written |

A result also carries the row as it is now, its version, the conflicts that lost a merge and the effects on other rows.

**Exactly once.** The result of every mutation, rejections included, is stored with its `op_id` for 30 days. A retried `op_id` returns the stored result and changes nothing, so a client that lost a response resends the whole batch.

**Merging.** A mutation names its base and only its changed fields. Fields no later change touched apply cleanly. The same field changed on both sides goes to the later arrival and the loser is told. A delete wins over an edit; only a `restore` brings the row back, and only within 30 days.

**Roles.** Transactions, recurrences and goal contributions need the ability to record (owner, admin, member). Everything else needs data management (owner, admin). A viewer pulls but every mutation is rejected `forbidden`.

## Errors and limits

Whole-request errors are problem documents with stable codes such as `bad_request`, `unauthorized`, `missing_scope`, `forbidden`, `device_revoked`, `batch_too_large`, `resync_required`, `protocol_unsupported`, `rate_limited` and `unavailable`. A rejected mutation carries a code of its own, for example `unknown_field`, `invalid_amount`, `invalid_date`, `unknown_reference`, `quota_exceeded`, `transfer_incomplete` or `duplicate_name`.

Pull and snapshot are limited to 60 a minute per device, and push to 30 a minute per device, with at most 5000 mutations an hour. Each answers 429 with `Retry-After`.

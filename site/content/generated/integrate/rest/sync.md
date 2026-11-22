---
title: "REST: Sync"
description: "The Sync endpoints of the vukem-finances REST API: scopes, parameters, bodies, responses and examples."
---

> Generated from `docs/openapi.json`. Do not edit this page: change the source and run `bun run reference` in `site/`.

7 operations. Every request needs `Authorization: Bearer vf_...`; [the REST API guide](/docs/integrate/rest-api/) explains conventions, scopes and errors.

## `GET /sync/devices`

List the devices the caller registered in this workspace.

| Field | Value |
| --- | --- |
| Operation | `sync.devices.list` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Envelope<SyncDeviceOut>` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `SyncDeviceOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `created_at` | `string` | yes |  |
| `device_id` | `string` | yes |  |
| `last_pull_seq` | `integer (int64)` | yes |  |
| `last_seen_at` | `string` |  |  |
| `name` | `string` | yes |  |
| `platform` | `string` | yes |  |
| `revoked_at` | `string` |  |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/sync/devices"
```

## `POST /sync/devices`

Register a device for the calling token.

| Field | Value |
| --- | --- |
| Operation | `sync.devices.create` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | `string` | yes | What the person calls the device At most 80 characters. |
| `platform` | `string` |  | ios, android, web or other At most 20 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `SyncDeviceOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `SyncDeviceOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `created_at` | `string` | yes |  |
| `device_id` | `string` | yes |  |
| `last_pull_seq` | `integer (int64)` | yes |  |
| `last_seen_at` | `string` |  |  |
| `name` | `string` | yes |  |
| `platform` | `string` | yes |  |
| `revoked_at` | `string` |  |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Ana's phone"}' \
  "$BASE/api/v1/sync/devices"
```

## `GET /sync/info`

The sync protocol versions, the entities and fields this server knows, and its limits.

| Field | Value |
| --- | --- |
| Operation | `sync.info` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `Info` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `Info`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `entities` | `array of EntityInfo` | yes |  |
| `limits` | `LimitsInfo` | yes |  |
| `protocol` | `Range` | yes |  |
| `server_time` | `string (date-time)` | yes |  |
| `workspace` | `WorkspaceInfo` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/sync/info"
```

## `POST /sync/pair`

Trade a pairing code for a sync token and a registered device.

| Field | Value |
| --- | --- |
| Operation | `sync.pair` |
| Scope | none, any valid token |
| Kind | REST only, sync protocol |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `code` | `string` | yes | The pairing code the Devices page shows At most 40 characters. |
| `device_name` | `string` | yes | What the person calls the device At most 80 characters. |
| `platform` | `string` |  | ios, android, web or other At most 20 characters. |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 201 | Created | `PairResult` |  |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `PairResult`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `device_id` | `string` | yes |  |
| `server_time` | `string (date-time)` | yes |  |
| `token` | `string` | yes |  |
| `workspace` | `PairWorkspace` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"code":"7K2M-9QXA","device_name":"Ana's phone"}' \
  "$BASE/api/v1/sync/pair"
```

## `GET /sync/pull`

Read the changes after a cursor, parents first.

| Field | Value |
| --- | --- |
| Operation | `sync.pull` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `cursor` | query | `integer (int64)` |  | The next_cursor of the previous page; 0 for the first pull |
| `limit` | query | `integer` |  | Changes per page: 500 by default, at most 1000 |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `PullResult` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `PullResult`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `changes` | `array of PullItem` | yes |  |
| `has_more` | `boolean` | yes |  |
| `next_cursor` | `integer (int64)` | yes |  |
| `server_time` | `string (date-time)` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/sync/pull"
```

## `POST /sync/push`

Send the mutations a device made offline.

| Field | Value |
| --- | --- |
| Operation | `sync.push` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `device_id` | `string` |  | The device; X-Device-Id is what counts and this must match when set At most 26 characters. |
| `mutations` | `array of MutationIn` |  | At most 200 mutations |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `SyncPushOut` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `SyncPushOut`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `results` | `array of Result` | yes |  |

### Example

```sh
curl -X POST -H "Authorization: Bearer vf_..." \
  -H "Content-Type: application/json" \
  -d '{"device_id":"","mutations":[{"base_seq":0,"entity":"","fields":null}]}' \
  "$BASE/api/v1/sync/push"
```

## `GET /sync/snapshot`

Read the live rows of one entity to rebuild a device.

| Field | Value |
| --- | --- |
| Operation | `sync.snapshot` |
| Scope | `sync` |
| Kind | REST only, sync protocol |

### Parameters

| Name | In | Type | Required | Description |
| --- | --- | --- | --- | --- |
| `entity` | query | `string` | yes | The entity to download, one of GET /sync/info |
| `after` | query | `string` |  | The last uid of the previous page |
| `limit` | query | `integer` |  | Rows per page: 500 by default, at most 1000 |
| `snapshot_id` | query | `string` |  | The snapshot_id of the first page, sent back on every later one |

### Responses

| Status | Meaning | Body | Error code |
| --- | --- | --- | --- |
| 200 | OK | `SnapshotPage` |  |
| 401 | Error (RFC 9457 problem details) | problem+json | `unauthorized` |
| 422 | Validation failed; errors maps each field to its messages and issues lists each problem with its code, key and params | problem+json | `validation_failed` |
| other | Error (RFC 9457 problem details) | problem+json |  |

### Fields of `SnapshotPage`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `entity` | `string` | yes |  |
| `has_more` | `boolean` | yes |  |
| `next_after` | `string` | yes |  |
| `rows` | `array of object` | yes |  |
| `snapshot_cursor` | `integer (int64)` | yes |  |
| `snapshot_id` | `string` | yes |  |

### Example

```sh
curl -H "Authorization: Bearer vf_..." \
  "$BASE/api/v1/sync/snapshot?entity=account"
```

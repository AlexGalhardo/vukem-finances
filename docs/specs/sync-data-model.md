# Sync data model

This is the contract every synced table of vukem-finances follows (decisions D-004, D-005, D-008, D-013). The server protocol and the mobile client build on it; no later epic may add a synced table that skips it. Synced entities today: `account`, `category`, `transaction`, `budget`. Not synced: imports (a server-side artifact), subscriptions, invites, users and memberships.

## Stable ids

Every synced table has `uid`, a ULID of 26 characters in upper-case Crockford base32, next to the integer `id`. The integer `id` stays the internal primary key and the target of foreign keys. The wire and the API use `uid` only, and references between entities are `<entity>_uid` fields (for example `account_uid`).

- A client generates the ULID itself (`crypto.getRandomValues`, no dependency). The server checks it with `changes.ValidUID`: 26 characters, valid alphabet, upper case, and a timestamp at most 24 hours in the future.
- The server accepts a `uid` that is unused in that workspace. Uniqueness is `(workspace_id, uid)`; two workspaces may hold the same `uid`.
- When the writer is the web UI, the server fills `uid` in `BeforeCreate`.

## Versions

Each synced row has `version` (int64, starts at 1, plus 1 on every change, including delete and restore). The bump happens in the database, in the trigger, so a Go struct held by the caller is stale after a write: every service write and every sync response re-reads the row to return the real `Version`.

## Soft delete

Synced models embed `orm.SoftDeletes` (`deleted_at`). A delete sets `deleted_at`; nothing is removed physically by a request.

- Deleting an account soft-deletes its transactions in the same database transaction, each one logged.
- Deleting a category leaves transactions in place with their category reference, which reads as uncategorized once the category is deleted; its budget is soft-deleted with it.
- A deleted row disappears from every read, total, budget, report and export. The only readers of deleted rows are `Restore` and the sync reader.
- Restore (`deleted_at` back to null) is allowed for 30 days after the delete.
- A row is purged physically only by compaction, after the retention window.
- Unique indexes are partial: `categories (workspace_id, name) WHERE deleted_at IS NULL` and `budgets (workspace_id, category_id) WHERE deleted_at IS NULL`, so a deleted name can be reused, and two devices that create the same name offline get a clear `duplicate_name` conflict instead of a database error.

## Change log

Table `changes`:

| Column | Meaning |
| --- | --- |
| `seq` | `INTEGER PRIMARY KEY AUTOINCREMENT`; global, strictly increasing; gaps are normal |
| `workspace_id` | the workspace of the row; the log is readable only within it |
| `entity` | the registered entity name (`account`, `category`, `transaction`, `budget`) |
| `uid` | the row's `uid` |
| `op` | `upsert` or `delete` |
| `row_version` | the row's `version` after the change |
| `fields` | JSON array of the changed column names |
| `created_at` | when the change was logged |

There is an index `(workspace_id, seq)`. Clients treat the cursor as an opaque, monotonic integer per workspace.

The log is state-based: it says which row changed, not what the row held, and a pull reads the current row. This also means the log holds no field values, so nothing can leak across workspaces through it.

SQLite triggers on every synced table (`AFTER INSERT`, `AFTER UPDATE`) append to `changes` and bump `version`, so imports, jobs and future code are logged without discipline. They are written by `changes.InstallTriggers(schema, entity)` (a migration passes a literal `Entity`, so its checksum does not move when the registry changes; `uid and workspace_id` are also protected by a `BEFORE UPDATE` trigger). A `delete` op is written when `deleted_at` goes from null to non-null; a restore is an `upsert`. An update that changes no synced field writes nothing. `updated_at` and `version` themselves are not listed in `fields`.

A startup check, `changes.VerifyTriggers`, fails the boot in non-test environments when a registered table lacks its triggers, and a test runs the same check.

Other tables written by the sync machinery: `sync_state (workspace_id, compacted_through_seq)` and `sync_conflicts (workspace_id, entity, uid, field, lost_value, lost_by_device, kind, created_at)`.

## Entity registry

`changes.Register(changes.Entity{Name, Table, Fields, Refs})` lists each synced entity, the columns a client may write, and its parents by `uid` (`transaction` refs `account` and `category`; `budget` refs `category`). A guard test fails when a table with a `uid` column is not registered or has no triggers. Later epics add their tables here.

## Table rebuilds

A table rebuild (`schema/rebuild.go`) can drop the triggers of the table. A migration that rebuilds a synced table must run the rebuild with `dropTriggers`, then call `changes.InstallTriggers` for that table. A test migrates a synced table through a rebuild and runs `VerifyTriggers`.

## Retention and compaction

`TOMBSTONE_RETENTION_DAYS` defaults to 90. The daily job `changes_compact`:

1. deletes `changes` rows older than the window, except the newest per `(workspace_id, entity, uid)`;
2. physically deletes soft-deleted rows older than the window;
3. records `sync_state.compacted_through_seq` per workspace.

A client whose cursor is below `compacted_through_seq` receives `resync_required` (`changes.ErrResyncRequired`) and must pull a full snapshot. Rows in `sync_conflicts` are kept for 30 days.

## Conflict policy

Row-level versions with field-level merging, as a pure function `changes.Merge(current Row, incoming Mutation, since []Change) Result`. A mutation carries `base_seq` (the last `seq` the client saw for that row) and only the fields it changed. `since` holds the changes of that row with `seq > base_seq`.

1. **Disjoint fields apply cleanly.** A field not changed by any `since` change applies. Example: the phone edits `description` of a transaction while the web UI changed its `amount_cents` after the phone's `base_seq`; both edits survive.
2. **Same field: the later arrival wins.** A field changed both by the client and by a later change is a conflict on that field, and the mutation wins by server order. The previous value is stored in `sync_conflicts` and returned in the response. Example: the web sets a budget limit to 50000, then the phone, offline since before that, sets it to 60000; the limit becomes 60000 and 50000 is kept as `lost_value`.
3. **Delete wins over a concurrent edit.** An edit of a row deleted after `base_seq` is dropped and logged as a conflict of kind `edited_deleted_row`; only an explicit restore brings the row back. Example: the phone renames an account that the web deleted meanwhile; the rename is dropped.
4. **Create with an existing uid.** When the fields are equal it is an idempotent retry (nothing changes); when they differ it is an edit under rule 2. Example: the phone resends a lost create after a timeout, and nothing duplicates.
5. **Missing or deleted parent.** A mutation that references a missing or deleted parent is rejected with `unknown_reference` or `parent_deleted`; the other operations of the batch are not affected. Example: a transaction that points at a category deleted on the web is rejected with `parent_deleted`.
6. **Duplicate name.** Creating a category with a live name already in use in the workspace is rejected with `duplicate_name`; the client renames or adopts the existing one. Example: two phones create "Food" offline; the second push gets `duplicate_name`.
7. **Derived values are never synced.** Balances, totals and budget progress are computed by each client from synced rows.

Last-writer-wins can overwrite an edit of an amount. The lost value is kept 30 days and reported to the client; the policy applies to money fields too (D-023).

## Money limit

`amount_cents`, `opening_cents` and `limit_cents` are limited to plus or minus 9007199254740991 (2^53 - 1) at every write path and in `ParseAmount`, so any client can hold them exactly. Values are plain JSON numbers.

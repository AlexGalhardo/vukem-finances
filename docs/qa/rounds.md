# QA rounds

Every problem the quality and security campaign (epic 24) finds is recorded here before it is fixed: surface, steps to
reproduce, expected, actual, evidence, severity (Critical, High, Medium, Low) and status (open, fixed, accepted). A fix
carries a test written first, in its own commit, with `Refs: epic 24 task N` and `Refs: QA-NNN`.

## Round 1 (in progress)

### QA-001: a copied session cookie still signs in after logout

- **Surface:** web, `POST /logout`, the default `SESSION_DRIVER=cookie` (the session is an encrypted cookie).
- **Steps:** sign in; copy the `vukem_session` cookie; `POST /logout` with the browser; send the copied cookie to
  `GET /dashboard` from another client.
- **Expected:** the copied cookie no longer authenticates (a redirect to `/login`).
- **Actual:** `GET /dashboard` answers 200 as the signed-in person, until the cookie expires (idle 2 hours, absolute 24 hours).
  The browser that logged out is signed out; only a replayed copy survives.
- **Evidence:** reproduced in the web suite while writing `TestSecurity_Web_Sessions`: the replayed cookie answered 200.
- **Severity:** Medium. Exploiting it needs the cookie of the victim (it is `HttpOnly`, `Secure` and `__Host-` in production).
- **Status:** fixed. The author chose database sessions as the default for both editions: `bootstrap.New` sets
  `vukem.WithEnvDefaults({"SESSION_DRIVER": "database"})`, so the process environment or `.env` can still override it. The
  framework creates the `sessions` table itself (already on the allow-list of the isolation guard, so no migration was added),
  stores `hex(SHA-256(session id))` as the row id, and prunes expired rows with its hourly `session:prune` task. Logging out
  deletes that one row: another device of the same person stays signed in, and `RevokeOtherSessions` now has rows to delete.
  An upgrade from the cookie driver signs people in once more; an old cookie is simply not a session. Regression tests:
  `TestSecurity_Web_Sessions` (replay after logout, another device, an old cookie, no session id in the logs or as a row id).

### QA-002: the base URL check accepts "localhost." and alternative spellings of an IPv4 address

- **Surface:** AI, `POST /ai/key` with the OpenAI-compatible provider (`ai.CheckBaseURL`).
- **Steps:** save a base URL of `https://localhost./v1`, `https://2130706433/v1`, `https://0x7f.0.0.1/v1`,
  `https://017700000001/v1` or `https://127.1/v1` in the saas edition.
- **Expected:** refused at save time, like `https://127.0.0.1/v1`.
- **Actual:** saved. The connect-time check of the safe client still refuses to connect to an address on this machine
  (`TestSecurity_SSRF_SaaSNeverConnectsToThisMachine` proves no connection is made), so this is a gap of the early check only.
- **Evidence:** `TestBaseURL` and `TestSecurity_SSRF_AIBaseURLInSaaS` failed before the fix.
- **Severity:** Low (defense in depth).
- **Status:** fixed. A trailing dot is trimmed before the host is judged, and a host made only of decimal or `0x` labels that
  is not a canonical IP literal is refused.

### QA-003: a Windows-1252 bank CSV is stored as invalid UTF-8

- **Surface:** statement import (`finance.CSVParser`), found by `FuzzCSVParser`.
- **Steps:** import a CSV whose description holds a Windows-1252 byte (`Caf` then `0xE9`, a common export of a Brazilian bank).
- **Expected:** the text is read as `Café`, as the OFX parser does (`decodeOFXText`).
- **Actual:** the byte is kept as it is, so a description that is not valid UTF-8 reaches the database, the exports and the
  JSON of the pages. The sync API and the tags already refuse such text.
- **Evidence:** `FuzzCSVParser` crash `1d454fa8f124f5c0` (now a regression seed) and `TestImportCSVParserDecodesWindows1252AndDropsTheBOM`.
- **Severity:** Low (data quality, no isolation or injection effect).
- **Status:** fixed. The CSV is decoded like an OFX file: a BOM is dropped, valid UTF-8 is kept, anything else is Windows-1252.

### QA-004: a recurrence that ends on 0001-01-01 never ends

- **Surface:** the recurrence engine (`Rule.Occurrences`), found by `FuzzRecurrenceOccurrences`.
- **Steps:** a weekly rule with interval 3, start date and end date `0001-01-01`, listed over `0001-01-01` to `0001-03-01`.
- **Expected:** one occurrence, the start date.
- **Actual:** three: the end date was the zero `time.Time`, which the engine read as "no end date".
- **Evidence:** crash `ca1206377a9ce3d8` (now a regression seed) and `TestRecurrenceEndDateInYearOne`.
- **Severity:** Low (a date no person enters; the TypeScript engine in `resources/js/lib/recurrence.ts` was not checked).
- **Status:** fixed with an explicit `hasEnd` flag.

The entries of task 10 (load, budgets and drills) are numbered from QA-500.

## QA-500: a sync push scans every row of the workspace once per transaction

- **Surface:** `POST /api/v1/sync/push`.
- **Steps:** seed one workspace of 100000 transactions, push a batch of 20 new transactions (`go run ./scripts/qa/loadtest`, scenario `sync_push`).
- **Expected:** a push of 20 mutations well inside the 300 ms budget.
- **Actual:** 866 ms median for one client, 26 s p95 under 20 clients. The monthly quota count used `substr(occurred_on, 1, 7) = ?`, which SQLite cannot answer from the index, so it read every row of the workspace for every pushed transaction.
- **Evidence:** `TestMonthQuotaCountUsesTheDateIndex` (query plan). After the fix one client measures 40 ms median for the same push.
- **Severity:** Medium.
- **Status:** fixed (`fix(sync): count the month quota by a date range so the index answers it`).

## QA-501: a crash while an import is queued loses the job for good

- **Surface:** web import (`POST /import/{import}`), the import queue.
- **Steps:** `go run ./scripts/qa/drills kill-import`: upload a 20000 row CSV, confirm it, kill the server while the confirm request runs, start it again.
- **Expected:** after the restart the import finishes once (all rows, none doubled), or the person can confirm it again.
- **Actual:** the rows are never half written (the import is one transaction), but the import stays `queued` forever: the queue is in process memory (there is no `queue_jobs` table), nothing re-dispatches a `queued` import at boot, and `ConfirmImport` only accepts an `uploaded` one, so it cannot be confirmed again either.
- **Evidence:** the drill fails with `the import is still "queued" 45 s after the restart with 0 rows`.
- **Severity:** Medium.
- **Status:** fixed (`fix(finance): run queued imports again at boot`). The imports still `queued` are re-dispatched when the web server starts. The import already writes its rows, counts and status in one transaction under the workspace lock and re-reads the status inside it, so a repeat or a concurrent run writes nothing twice (`TestImportJobRunConcurrentlyWritesRowsOnce`, `TestImportRecoveryRunsAQueuedImportOnce`). The sync queue default is kept.

## QA-502: the full migration rollback failed

- **Surface:** `migrate:rollback` of the whole history.
- **Steps:** migrate a fresh database up, roll every migration back.
- **Expected:** every `Down` runs.
- **Actual:** the `Down` of the credit cards migration failed (`error in trigger transactions_changes_update after drop column: no such column: OLD.statement_ref`); behind it the Downs of the transfers, tags and sync columns migrations failed too (a trigger still named the dropped column; a `Rebuild` that did not say which columns and index to lose).
- **Evidence:** `TestFullRollbackFromFreshDatabase`.
- **Severity:** Medium.
- **Status:** fixed (`fix(db): make a full migrate:rollback run from a fresh database`). The migrations are frozen, so the registry wraps their `Down` (`database/migrations/rollback_fixes.go`).

## QA-503: lists, the dashboard and the export miss their budgets at 100000 rows

- **Surface:** dashboard, transaction list, reports, CSV export, MCP `transactions_list`.
- **Steps:** `go run ./scripts/qa/loadtest -check`.
- **Expected:** the budgets of `perf-budgets.md`.
- **Actual:** see the table in `perf-budgets.md`: one client alone already measures about 470 ms for the dashboard, 250 ms for a list page, 170 to 640 ms for the reports and 1.2 s for the 50000 row export, and 20 clients multiply that by the queueing of a single SQLite writer and the CPU of the machine.
- **Evidence:** the load test output.
- **Severity:** Medium.
- **Status:** open. Budgets are not loosened. Suspected causes, not yet confirmed: `TransactionPage` runs a full `COUNT(*)` of the workspace for every page, and the dashboard and the reports aggregate over all rows of the period on each request. Options are a covering index on `(workspace_id, deleted_at, occurred_on)` (a migration), a cached total, or a summary table; each changes the data model, so it goes to the orchestrator.

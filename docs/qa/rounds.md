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

### QA-010: the address a workspace invited stays in its audit log after that person deletes their account

- **Surface:** account deletion, `privacy.PurgeAccount`, the `audit_events` of another person's workspace.
- **Steps:** a member invites `ana@example.com` to the workspace (the audit row `invite.created` has the subject
  `ana@example.com as viewer`); Ana has an account, asks to delete it and the grace period ends; the purge runs.
- **Expected:** the address of the deleted person is in no table. The purge already deletes the `invites` rows, the password
  reset rows and the mail suppression of that address.
- **Actual:** `audit_events.subject` still holds the address in the inviting workspace, for as long as that workspace lives.
- **Evidence:** `TestDataLifecycle_AccountDeletionLeavesNoTraceOfThePerson` (it scans every column of every table for the address).
- **Severity:** Medium (personal data kept after a deletion; the row is visible to the owner and admins of the other workspace only).
- **Status:** fixed. `PurgeAccount` rewrites the subject of the `invite.created` rows that name the address to `[removed]` (the row stays, so
  the log still shows an invitation was made). Regression test: `TestDataLifecycle_AccountDeletionLeavesNoTraceOfThePerson`.

### QA-011: a session opened while the account waited for deletion outlives the purge

- **Surface:** account deletion, `privacy.PurgeAccount`, the `sessions` table (`SESSION_DRIVER=database`, the default).
- **Steps:** request the deletion of the account; sign in again during the grace period (it leads to the offer to cancel, so it
  opens a session); let the grace period end and run the purge.
- **Expected:** no row names the person any more.
- **Actual:** the `sessions` row of that sign-in stays, with `user_id = web:<id>` and the session data, until it expires (2 hours
  idle, 24 hours absolute) and the hourly `session:prune` removes it. The deletion revoked the sessions that existed at the time
  of the request, and the purge does not revoke again.
- **Evidence:** `TestDataLifecycle_AccountDeletionLeavesNoTraceOfThePerson`.
- **Severity:** Low (the row holds an id that no user has any more, and it dies within a day).
- **Status:** fixed. `PurgeAccount` ends every credential of the person again (`RevokeAllAccess`, outside the transaction) before it
  deletes their rows, and the purge job gives it the accounts service. Regression test: `TestDataLifecycle_NoSessionOutlivesThePurge`.

### QA-012: the operator guide does not say that a backup keeps what was deleted

- **Surface:** `deploy/README.md`, "Backup".
- **Steps:** read how to take and keep snapshots; delete an account or a workspace and look for what happens to old snapshots.
- **Expected:** the guide says that a snapshot keeps the data as it was, deletion never reaches it, and the operator decides when
  to rotate it (`docs/security.md` already lists "backups are retained until rotated", epic 34 leaves existing backups to the
  retention policy of epic 03).
- **Actual:** the guide only says "Old snapshots are not removed for you".
- **Evidence:** `TestDataLifecycle_BackupsAreDocumentedAsRetainedUntilRotated`.
- **Severity:** Low (documentation).
- **Status:** fixed. "Backup" now says that a snapshot keeps what was deleted until the operator rotates it. Regression test:
  `TestDataLifecycle_BackupsAreDocumentedAsRetainedUntilRotated`.

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

- **Surface:** dashboard, transaction list, reports, CSV export, sync push under load.
- **Steps:** `go run ./scripts/qa/loadtest -check`.
- **Expected:** the budgets of `perf-budgets.md`.
- **Actual (profile, `EXPLAIN QUERY PLAN` and timings per statement, 100000 rows):**
  - the dashboard ran 420 ms: the balance `SUM` by account and kind took 180 ms and the count by account 71 ms, both going back to the table for every row; about 100 ms more was hydrating the 25000 rows of six months through the ORM;
  - the list ran `COUNT(*)` of the workspace for the total: 34 ms, a table lookup per row;
  - the net worth series (default report) took 294 ms, and the 12 month report ran it twice (1268 ms in all): `GROUP BY occurred_on, account_id, kind, amount_cents` with a temp b-tree and a table lookup per row.
- **Fix applied:** migration `20261126120000_add_transaction_cover_indexes` adds two covering indexes. The same profile then shows the dashboard at 221 ms, a list page at 18 ms, the default report at 315 ms and the 12 month report at 868 ms. `TestTransactionCoverIndexesAnswerTheHotQueries` checks the plans and the rollback.
- **Evidence:** the table in `perf-budgets.md`. After the covering indexes, with one client: dashboard 448 ms p95 (budget 300), reports 359 ms, export 1478 ms (budget 1000). After the monthly summary and the export change, with one client every scenario is inside its budget (dashboard 13 ms, reports 4 ms, export 754 ms, filtered list 68 ms, push 45 ms) and with 20 clients the dashboard (77 ms), the reports (13 ms) and the export (751 ms) are too.
- **Severity:** Medium.
- **Status:** partly fixed in v0.7.0; the rest stays accepted (`accepted.md`).
  - Fixed: a per-workspace summary, `transaction_month_totals` (migration `20261127100000`), kept by insert, update and delete triggers on `transactions`, so creates, edits, soft deletes, restores, transfers, installments, imports, sync push and the purge are all counted by the database. The dashboard, the budget spending and the reports (cash flow, categories, accounts, totals, net worth) read whole months from it and only the days of a partial month from the rows. Money stays exact: the cents are summed as a high and a low part, joined in Go. `TestMonthTotalsBackfillAndTriggers` (a database with rows, the backfill, random single column writes, rollback) and `TestMonthTotalsEqualARecomputation` (random service operations in one workspace, the range reader against a direct sum, another workspace untouched) prove it equals a recomputation. The upgrade drill ran the backfill on a v0.6.0 database; a 1000000 row table backfills in about 2 s.
  - Fixed: the CSV export read whole models and continued each chunk with an `OR` of two conditions that made SQLite walk the index three times slower: it now reads plain columns and continues with a row value comparison (50000 rows: 590 ms to 185 ms in the service, 1478 ms to 754 ms p95 in the load test).
  - Still over with 20 clients: `transactions_filtered` (1541 ms p95, budget 300) and `sync_push` (1576 ms p95, budget 300). Neither is an aggregate: the filter is a `LIKE '%text%'` scan and a push is 20 writes by the single SQLite writer. Options for a later release: full text search for the filter, and a cheaper insert path for a push.

The entries of task 12 (accessibility audit) are numbered from QA-600. All of them were found by axe in
`e2e/a11y/pages.spec.ts`, which is the failing test for each: it failed on these rules before the fixes and passes after.

## QA-600: the sidebar group labels fail the contrast minimum

- **Surface:** the sidebar of every signed-in page (`SidebarGroupLabel`), light and dark.
- **Steps:** open any page and run axe (`bunx playwright test e2e/a11y/pages.spec.ts`).
- **Expected:** text contrast of at least 4.5:1 (WCAG 1.4.3).
- **Actual:** `color-contrast` (serious) on the five group labels (Money, Planning, Insights, Data and Settings): the label used `text-sidebar-foreground/70`, which lowers the already muted sidebar text below the minimum.
- **Evidence:** `e2e/a11y/pages.spec.ts`, rule `color-contrast`.
- **Severity:** Medium.
- **Status:** fixed: the label uses `text-sidebar-foreground` (the muted token, at full strength).

## QA-601: the search button's accessible name does not contain its visible text

- **Surface:** the top bar button that opens the command palette.
- **Steps:** open any page and run axe.
- **Expected:** the accessible name starts with the visible label (WCAG 2.5.3, label in name).
- **Actual:** `label-content-name-mismatch` (serious): an `aria-label` was set while the visible content was the shortcut hint, so a person using speech control saying the visible text did not match the name.
- **Evidence:** `e2e/a11y/pages.spec.ts`, rule `label-content-name-mismatch`.
- **Severity:** Medium.
- **Status:** fixed: the `aria-label` is gone and the label text is always in the button (visually hidden below the 2xl width), so the name is the visible text.

## QA-602: the sidebar header and footer are outside every landmark

- **Surface:** the sidebar of every signed-in page.
- **Steps:** open any page and run axe.
- **Expected:** all content is inside a landmark.
- **Actual:** `region` (moderate): the workspace header block and the Documentation link are not in a landmark, because only the navigation was.
- **Evidence:** `e2e/a11y/pages.spec.ts`, rule `region`.
- **Severity:** Low.
- **Status:** fixed: the sidebar is a `complementary` landmark named after the application.

## QA-603: the hidden table of a chart is a scroll region that cannot take focus

- **Surface:** `/reports`, the table behind each chart.
- **Steps:** open `/reports` and run axe.
- **Expected:** no scroll region without keyboard access (WCAG 2.1.1).
- **Actual:** `scrollable-region-focusable` (serious): the screen-reader-only table kept `overflow-x-auto` on its container.
- **Evidence:** `e2e/a11y/pages.spec.ts`, rule `scrollable-region-focusable`.
- **Severity:** Low.
- **Status:** fixed: while the table is visually hidden its container does not scroll.

## QA-604: a table wider than its box cannot be scrolled with the keyboard

- **Surface:** every `Table` (found on `/devices` in Spanish, where longer text made the table overflow).
- **Steps:** switch to Spanish, open `/devices`, run axe. Without a mouse the cut-off columns cannot be reached.
- **Expected:** a box that scrolls sideways can take focus.
- **Actual:** `scrollable-region-focusable` (serious).
- **Evidence:** `e2e/a11y/pages.spec.ts`, es, `/devices`.
- **Severity:** Medium.
- **Status:** fixed: the table container becomes a tab stop only while its content overflows (a `ResizeObserver` in `components/ui/table.tsx`).

## QA-605: Enter on a focused suggestion accepts another suggestion

- **Surface:** `/suggestions`, the list of pending categories.
- **Steps:** with two or more pending suggestions, move the focus to a row that is not the active one (a screen reader or a script can) and press Enter.
- **Expected:** the focused suggestion is accepted.
- **Actual:** the active row was accepted, so another transaction got a category.
- **Evidence:** `e2e/journeys/j06-ai.spec.ts`.
- **Severity:** Medium.
- **Status:** fixed: a row that receives the focus becomes the active row.

## QA-606: the end-to-end suite passes the AI call limit of its one workspace

- **Surface:** `e2e/screens-a-tour.spec.ts`, the AI rate limit (`app/ai/usage.go`).
- **Steps:** run the whole Playwright suite. Every AI spec uses the same workspace.
- **Expected:** the suggestions spec gets its category from the fake provider.
- **Actual:** the call was refused with "too many AI calls this minute", the page said AI could not finish, and the spec failed in CI and in three local runs. The product behaved as designed.
- **Evidence:** the server log of the failing run, and PR #10 CI.
- **Severity:** Low (test infrastructure).
- **Status:** fixed: `AI_CALLS_PER_MINUTE` configures the limit, 20 by default, and only the end-to-end suite raises it.

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

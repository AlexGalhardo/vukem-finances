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
- **Evidence:** reproduced in the web suite while writing `TestSecurity_Web_Sessions` (an assertion that the replayed cookie
  redirects to `/login` fails). The assertion is not committed, so the tree stays green.
- **Severity:** Medium. Exploiting it needs the cookie of the victim (it is `HttpOnly`, `Secure` and `__Host-` in production).
- **Status:** open. The fix is a decision for the author, not a patch: either run with `SESSION_DRIVER=database` (a
  deployment and data decision; server-side sessions are deleted on logout) or make `POST /logout` bump
  `users.session_version`, which signs the person out of every device, not only this one.

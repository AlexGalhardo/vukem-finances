# Security: threat model

This is the threat model of vukem-finances. It names what is worth protecting, where trust changes, and, per surface, the
threats (STRIDE: spoofing, tampering, repudiation, information disclosure, denial of service, elevation of privilege), the
control that answers each one and the test that proves the control. A row without a test is a gap and is listed under
[Findings](#findings).

How to report a vulnerability: see [SECURITY.md](../SECURITY.md). The rules of the code that these controls come from are in
[AGENTS.md](../AGENTS.md); the campaign that keeps this document honest is epic 24 (`docs/qa/rounds.md`).

Tests named `TestSecurity_<Area>_<Case>` are the attack suite (`go test ./tests/... -run TestSecurity_`). Where a row names
a family such as `TestSecurity_MCP_*`, it is the suite of that area in `tests/security_*_test.go`.

## Assets

| Asset | Where it lives | Why it matters |
| --- | --- | --- |
| Financial data (accounts, transactions, budgets, goals, notes) | SQLite, in clear (D-005: SQL sums and filters it) | The product itself; a leak between workspaces is the worst bug |
| Sessions and the CSRF secret | the session store and a cookie | Account takeover |
| Passwords, two-factor secrets and recovery codes | `users`, hashed or sealed | Account takeover |
| API tokens (`vf_<id>.<secret>`) | `access_tokens`, a SHA-256 of the secret | Programmatic access to one workspace |
| OAuth grants, access and refresh tokens of remote MCP | `mcp_grants`, `oauth_*`, tokens signed with a key derived from `APP_KEY` | Access by an AI client |
| Sync devices and pairing codes | `sync_devices`, hashed codes | Access by the mobile app |
| Telegram links, link codes and the bot token | `telegram_links`, hashed codes, `TELEGRAM_BOT_TOKEN` | Access through a chat; the bot token lets anyone act as the bot |
| AI provider keys | `ai_credentials`, sealed with the workspace as additional data | Someone else's spend and data |
| Stripe and AbacatePay secrets, webhook secrets | environment | Forged payments, free Pro plans |
| `APP_KEY` | environment | Derives every signing and sealing key; its loss breaks all of the above |
| Backups and exports | `db:backup` files, CSV and JSON exports | A full copy of the data outside the controls of the app |
| Logs | stdout and stderr | Must hold no secret and no financial text |

## Trust boundaries

| Boundary | Trusted side | Untrusted side | Crossing control |
| --- | --- | --- | --- |
| Browser to server | the server | the browser, its cookies, every field | session, CSRF, role check in each handler, validation |
| API client to server | the server | the bearer of a token | token hash, scope ceiling, live role, workspace binding |
| MCP client and its model to server | the operation registry | the client, the model it runs and the text it reads | OAuth, scopes, `Invoke`, confirmation of destructive tools |
| MCP client metadata fetch | the server | a URL the client names | HTTPS only, public addresses after DNS, no redirects, 5 s and 5 KiB |
| Telegram to server | the server | anyone who can reach the webhook or write to the bot | secret header, single-use link codes, live role per message |
| Stripe and AbacatePay to server | the server | anyone who can post to the webhook | signature or shared secret, re-read of the charge |
| Server to AI providers | the server | the provider and the base URL a workspace owner sets | safe client, `CheckBaseURL`, redaction, schema-only output |
| Mobile device to server | the server | the device and what it pushes | token scope, per-entity role, validation of every mutation |
| Desktop webview to local server | the loopback server | other local processes and web pages | `desktopgate`: loopback peer and Host, launch token, cookie |
| Host and operator | the application | the machine, the proxy and the backups | Caddy, Docker, `APP_KEY` outside the image, documented backups |
| Imported files to parsers | the parsers | CSV and OFX files | size, row and cell bounds, no entities, `ParseAmount` only |

## Web application (browser)

### Web: spoofing and session

| Threat | Control | Test |
| --- | --- | --- |
| Session theft or fixation | The session is a database row (`SESSION_DRIVER=database`, the default of the app) that rotates on login and is deleted on logout, so a copied cookie dies with it; the cookie is `HttpOnly`, `SameSite=Lax`, `__Host-` and `Secure` in production | `TestSecurity_Web_Sessions` (see QA-001 in `docs/qa/rounds.md`) |
| Credential stuffing | Per-IP and per-account limits; failed two-factor codes limited per account | `TestSecurity_TwoFactorPerAccountLimit` |
| User enumeration through login, reset and registration | The same answer for a known and an unknown address | `TestSecurity_Web_Enumeration` |
| Self-asserted admin or unverified email | Verified email is required in saas; an address is never marked verified at sign-up | `TestSecurity_SaaSRequiresEmailVerification`, `TestSecurity_SelfAssertedAdminEmailIsNotAdmin` |

### Web: tampering and request forgery

| Threat | Control | Test |
| --- | --- | --- |
| CSRF on a state-changing route | The framework CSRF middleware on every browser route; the API, the Telegram webhook and the MCP endpoints skip it because no cookie authenticates them | `TestSecurity_Web_CSRF` |
| Open redirect | Redirect targets are same-origin paths only | `TestSecurity_Web_OpenRedirect` |
| SQL injection through search, filters and sort | Query builder with bound parameters; sort keys are an allow-list | `TestSecurity_SearchTakesSQLMetacharactersAsText`, `TestSecurity_Injection_SQL` |
| Formula injection in an export | Every text cell passes `finance.SpreadsheetSafe` | `TestSecurity_Injection_CSVFormula` |

### Web: information disclosure and elevation

| Threat | Control | Test |
| --- | --- | --- |
| A row of another workspace | Every query goes through the workspace scope; a foreign id is a 404 | `TestIsolationGuard`, `TestCrossTenantMatrix` |
| A role does more than its table says | Each write handler starts with `h.can(c, ability)`; the table is `app/workspaces/policy.go` | `TestSecurity_ViewerCannotEscalate`, `TestSecurity_AdminCannotPromoteToOwner` |
| Missing security headers, clickjacking | CSP with `default-src 'self'` and `frame-ancestors`, `X-Content-Type-Options`, `Referrer-Policy`; HSTS in production | `TestSecurity_Web_Headers`, `TestSecurity_Web_Production` |
| Stored XSS through a description or a name | React escapes text; Telegram and mail text pass `EscapeHTML`; no `dangerouslySetInnerHTML` of user text | `TestSecurity_Injection_HTML` |
| Secrets in a page prop, flash or log | The AI field is `apiSecret` so the framework never flashes it; secrets print `[redacted]` | `TestSecurity_AIKeyNeverReturned` |

## Public API

### API: spoofing and elevation

| Threat | Control | Test |
| --- | --- | --- |
| A stolen or guessed token | Only the SHA-256 of the secret is stored and the token is shown once; failed lookups are limited per address | `TestSecurity_TokenShownOnce`, `TestAPITokenGuard*` |
| A token used in another workspace | A token is bound to one workspace through `access_tokens.resource` | `TestSecurity_TokenOtherWorkspace` |
| A token outlives its member | Removing a member revokes their tokens of the workspace; the live role caps the scope on every request | `TestSecurity_RemovedMemberToken` |
| A scope wider than needed, or `*` | No `*` scope exists and a token with one is refused; scope is a ceiling intersected with the role | `TestAPIScopeMatrix` |
| A foreign uid | The uid of a row becomes its id inside the workspace; a foreign uid is a 404 (order: 401, 404, 429, 403, 404) | `TestOperationIsolation`, `TestCrossTenantMatrix` |

### API: tampering, disclosure and denial of service

| Threat | Control | Test |
| --- | --- | --- |
| Hostile JSON (duplicate keys, deep nesting, huge numbers) | Strict binding, `validate` tags, amounts through `finance.ParseAmount`, `models.MaxCents` | `TestSecurity_Injection_JSON` |
| A replayed write | The idempotency key of a write, 24 hours, per caller | operation registry tests |
| Floods | 120 reads and 60 writes a minute per token | `TestAPIRateLimit*` |
| Secrets or internal errors in a response | Problem+json with a stable code; an internal error has a generic message | `TestSecurity_InternalErrorsAreGeneric` |

## MCP over stdio (personal edition)

### MCP stdio: spoofing and elevation

| Threat | Control | Test |
| --- | --- | --- |
| The server is exposed to the network | `mcp:serve --stdio` runs only in the personal edition and opens no listener; with saas it exits before serving | `TestStdoutOnlyProtocol`, `TestEditionRefusal`, `TestSecurity_MCP_NoRemoteExposure` |
| The agent writes or deletes without being asked | Read only by default (`MCP_TOOLSETS`); destructive tools are off unless `MCP_ALLOW_DESTRUCTIVE` and then ask the person through `mcp.Confirm` | `TestDestructiveGate`, `TestWritesAreRefusedByDefault` |
| A tool outside the toolset | It is not registered and the actor holds no scope for it | `TestToolsets` |

### MCP stdio: tampering and disclosure

| Threat | Control | Test |
| --- | --- | --- |
| Prompt injection from stored text (a transaction description) | Results are data fields; the server adds no instruction built from a result; no tool acts on what it reads | `TestSecurity_InjectedDescriptionIsOnlyData`, `TestSecurity_MCP_ResultsHoldNoServerInstruction` |
| A secret in a result | A result never carries a password hash, a token or the app key | `TestSecurity_NoSecretLeaves` |
| Output on stdout corrupts the protocol | Only protocol messages go to stdout; logs go to stderr | `TestStdoutOnlyProtocol`, `TestEditionRefusal` |

## MCP remote with OAuth (saas edition)

### MCP remote: spoofing and token abuse

| Threat | Control | Test |
| --- | --- | --- |
| An authorization code intercepted | PKCE `S256` only, exact `redirect_uri`, code single use and short lived; a replay ends the grant | `TestSecurity_MCP_Authorize`, `TestSecurity_MCP_Code` |
| A stolen refresh token | Rotation with reuse detection that revokes the whole grant; tokens stored hashed | `TestSecurity_MCP_Refresh` |
| A token meant for another resource (confused deputy, passthrough) | The audience is this server only; tokens of another issuer or audience are refused; the server calls no other service with a client token | `TestSecurity_MCP_AccessToken` |
| A forged or tampered token | Signed with a key derived from `APP_KEY`; `alg none` and other algorithms refused | `TestSecurity_MCP_AccessToken` |
| A client that lies about who it is | A client named by a metadata document is shown by its verified host; the claimed name is shown apart | `TestOAuthConsentDoesNotTrustAClaimedName` |
| Dynamic Client Registration abuse | Registration is absent: clients are pre-registered or identified by a metadata document | `TestSecurity_MCP_Authorize` |

### MCP remote: authorization and isolation

| Threat | Control | Test |
| --- | --- | --- |
| A grant of workspace A reaches workspace B | The grant is bound to `<user>:<workspace>` in `mcp_grants`; the workspace never comes from arguments | `TestSecurity_MCP_Isolation` |
| Scope escalation by parameter or by tool arguments | Scopes are fixed at consent and cut to the role at consent and at every call; a promotion never widens a grant | `TestSecurity_MCP_Scopes`, `TestMCPPromotionDoesNotWidenAGrant` |
| A removed member or a reset password keeps access | The verifier reads the membership on every request; `revokeAccess`, `RevokeOtherSessions` and `revokeTokens` end the grants | `TestMCPGrantsEndWithTheirPerson`, `TestMCPRemoteIsolation` |
| A destructive tool runs unconfirmed, or the same call confirms itself | Tools that remove data are for owners and admins and ask the person through the protocol; a client that cannot ask is refused; no argument confirms | `TestSecurity_MCP_Destructive` |
| Error text leaks another workspace's data | A foreign uid is `not_found`; internal errors are generic | `TestSecurity_MCP_Errors` |

### MCP remote: transport and denial of service

| Threat | Control | Test |
| --- | --- | --- |
| DNS rebinding and cross-site calls | `Origin` is validated on every request (403); only `POST` is served | `TestSecurity_MCP_Transport` |
| Session confusion | No `Mcp-Session-Id` is issued or honored (revision `2026-07-28` has no protocol sessions) | `TestSecurity_MCP_Transport` |
| SSRF through the client metadata document | HTTPS only, public addresses after DNS, no redirects, 5 s and 5 KiB | `TestClientMetadataSSRF`, `TestSecurity_MCP_ClientMetadata` |
| Floods | 120 reads and 60 writes a minute per grant, 300 requests a minute per address on `/mcp`, 30 a minute per address on each OAuth endpoint | `TestMCPRateLimit`, `TestSecurity_MCP_RateLimit` |
| A call without a trace | Every call writes a `mcp call` line with client, tool and outcome, never an input | `TestMCPAudit` |

## Sync (mobile)

### Sync: spoofing, tampering and elevation

| Threat | Control | Test |
| --- | --- | --- |
| A pairing code guessed or reused | Single use, short lived, hashed, rate limited, concurrency safe | `TestSecurity_PairingCodeSingleUse`, `TestSecurity_PairingConcurrent`, `TestSecurity_PairingRateLimit` |
| A device pushes rows of another workspace | A push resolves every uid inside the token's workspace only and never trusts `version`, `workspace_id` or timestamps | `TestSecurity_PushOtherWorkspaceUID` |
| A viewer or member writes what the role forbids | The role is checked per entity (`abilityFor`) inside the batch | `sync_push` tests |
| A replayed push | Results stored by `op_id` for 30 days; a retry changes nothing | `sync_ops` tests |
| A hostile amount, date or field | The same validation as the services (`finance.ParseAmount`, `models.MaxCents`) | `sync_limits` tests |

## Telegram bot

### Bot: spoofing, elevation and disclosure

| Threat | Control | Test |
| --- | --- | --- |
| A forged update | The secret header of the webhook; each `update_id` is claimed once | `TestSecurity_TelegramWebhook` |
| Linking a chat to someone else's account | A single-use code made for the signed-in person, hashed, 10 minutes; never by account id, email or password | `TestSecurity_LinkCodeSingleUse`, `TestSecurity_NoLinkByAccountID` |
| A removed member keeps the bot | The role is read from the membership at every message and tap | `TestSecurity_RemovedMemberLosesBot` |
| A group chat reads private data | Only private chats are served | `TestSecurity_GroupChatIgnored` |
| The bot token or a message in logs | The client scrubs the token; logs hold update ids and error kinds only | `TestSecurity_TokenNeverInLogsOrResponses` |
| Flooding the bot | Per-chat limits kept in `telegram_counters` | `TestSecurity_TelegramRateLimit` |

## Webhooks and billing

### Webhooks: spoofing, replay and money

| Threat | Control | Test |
| --- | --- | --- |
| A forged Stripe or AbacatePay event | Stripe signature with 5 minutes of tolerance; AbacatePay shared secret in constant time and HMAC, never from the query string, 64 KiB cap; the body only names the charge, which is re-read at the provider | `TestSecurity_PixWebhook`, `TestSecurity_Webhook_StripeForgery`, `TestSecurity_Webhook_ResendForgery` |
| A replayed or reordered event | Conditional `UPDATE`s with the affected rows checked; `SyncSubscription` never writes the revocation columns | `TestPixConcurrentWebhookRequests`, `TestSecurity_Webhook_StripeReplayAndConcurrency` |
| Double activation under concurrent deliveries | One conditional `UPDATE ... WHERE status IN ('pending','expired')` in the entitlement transaction | `TestSecurity_Money_ServerOwnsThePrice`, `TestSecurity_Webhook_ConcurrentPixDeliveriesActivateOnce` |
| A refund or dispute keeps access | Revocation through `access_revoked_at`; access is derived in `Entitled`, never a stored plan flag | `TestRefundRevokesPlan`, `TestDisputeRevokesPlan`, `TestPixRefund*` |
| A client names the price or the months | The amount is computed on the server from `PIX_PRICE_PRO_MONTHLY_CENTS` and `PIX_MONTHS_OPTIONS` | `TestSecurity_Money_ServerOwnsThePrice`, `TestSecurity_Webhook_ConcurrentPixDeliveriesActivateOnce` |
| The plan quota passes its limit under concurrency | The check runs under the workspace lock in the writing transaction | `TestQuotaAtomicWhenPlanChanges`, `TestSecurity_Money_PlanLimitsHoldUnderConcurrency` |

## AI layer

### AI: tampering, disclosure and SSRF

| Threat | Control | Test |
| --- | --- | --- |
| The provider key leaks | `ai.Key` prints `[redacted]`; sealed at rest with the workspace as additional data; never in a response, prop, log, audit event or export | `TestSecurity_AIKeyNeverReturned`, `TestAIExcludedFromExportAndAPI` |
| Prompt injection from a description | Untrusted text is data in a JSON field, passes `ai.Sanitize`; output is accepted only as the JSON of a schema, for ids the server sent, and never writes | `TestSecurity_DescriptionInjection`, `TestSecurity_DescriptionInjectionWithAutoApply` |
| SSRF through the custom base URL | `ai.CheckBaseURL` and the safe client (30 s, 1 MiB, no redirects); `AI_ALLOW_PRIVATE_HOSTS` works only in the personal edition | `TestSecurity_SSRF_AIBaseURLInSaaS`, `TestSecurity_SSRF_SaaSNeverConnectsToThisMachine` |
| Someone else's spend | Bring your own key; 20 calls a minute and a monthly limit per workspace | `ai_test` limits |
| Personal data sent to the provider | Free text goes through `ai.Redact`; account and tag names go as opaque labels | `TestEntrySendsOnlyRedactedText`, `TestKeyRedacted` |

## Mobile app

### Mobile: device and transport

| Threat | Control | Test |
| --- | --- | --- |
| A lost or stolen device | The device token is a revocable sync token; revoking the device or removing the member stops it at once | `sync_devices` tests |
| Local data read by another app | The data is in the app's own SQLite store; the app holds no provider or payment secret | mobile review (not automated) |
| A hostile server or a downgrade | The client speaks protocol 1 (`X-Sync-Protocol`) and treats every answer as data | `mobile/` integration tests |
| Conflicting edits overwrite data | Merge is deterministic and tested by scenarios and a property test | `TestSyncScenarios`, `TestSyncConverge*` |

## Desktop app

### Desktop: loopback server

| Threat | Control | Test |
| --- | --- | --- |
| Another local process or web page reaches the server | `desktopgate` wraps the whole handler: loopback peer and Host, no proxy headers, same-origin fetches, the `vf_desktop` cookie from a one-time launch token | `TestSecurity_GateNoCookie`, `TestSecurity_GateTokenSpentOnce` |
| DNS rebinding to `127.0.0.1` | The Host is checked against the loopback address and port | `TestSecurity_GateRebinding` |
| The cookie bypasses the setup secret | `desktopgate.Verified(ctx)` is the only way to skip the setup code, and only the gate reads the cookie | `TestSecurity_GateCookieAloneCannotBypassSetupSecret` |
| A route outside the gate | Native actions go through `/desktop/*` routes of the gate; nothing is served outside it | `TestSecurity_GateHeadersSurviveTheApp` |
| An unsigned build is replaced | Builds are unsigned; the checksum of each release is published | release process |

## Findings

Open and closed problems of the campaign are in `docs/qa/rounds.md`; problems the author accepted are in
`docs/qa/accepted.md`. Known limits of this model:

- **Session rows hold the session payload in clear** (the signed-in user id, the workspace in use, flash messages). The row id is
  `hex(SHA-256(session id))`, so a copy of the database yields no cookie, but it does name who was signed in. Treat the database
  file and its backups as secret. (QA-001 moved sessions from encrypted cookies to these rows.)
- **Financial data is not encrypted at rest** (D-005: SQL must sum and filter it). Protect the host, the volume and the backups.
- **Backups are retained until rotated**: `db:backup` files hold a full copy and live outside the controls of the app.
- **Desktop builds are unsigned.**
- **The real-device checks** (screen readers, the mobile device store) are outside what the automated suite proves.

## Regression cases of earlier audits

The classes of problem found by the audit of the previous application are kept as cases of the suite and not as prose. Each
class is covered by the rows above: cross-workspace reads and writes (isolation guard and cross-tenant matrix), webhook
forgery and replay, token and pairing code reuse, import bombs and formula injection, secrets in logs, and a removed member
keeping access on any surface.

## When to update this document

A new surface, boundary or asset, a new operation offered on a new surface, or a finding of a round updates the tables here in
the same pull request. The inventory (`docs/qa/inventory.md`) is the measure of which entries have a covering test.

# End-to-end journeys

Epic 24, task 13. Twelve journeys, each with a runner. Every provider is a fake (AbacatePay, Telegram, the AI provider,
Stripe in the Go tests): nothing here reaches a real service or real money.

Run the browser journeys with `bunx playwright test e2e/journeys` and the others with
`bash scripts/qa/journeys-nonweb.sh` (it needs `bun install --frozen-lockfile` in the root, `mobile/` and `site/`, and
Node 22.13 or newer).

| # | Journey | Runner | What is covered | Not covered here |
| --- | --- | --- | --- | --- |
| 1 | Personal install: accounts, import, categories, budget, report, export, backup, restore | `e2e/journeys/j01-personal.spec.ts` and step 1 of the script (`scripts/qa/drills backup-restore`) | account, category, a CSV import with mapping and preview, a budget, a report, the CSV and JSON exports, then backup and restore of the database | the Docker install (`scripts/docker-persist-test.sh` needs Docker) and OFX in this run (`e2e/import-categories.spec.ts` covers it) |
| 2 | SaaS: Free limits, PIX upgrade, refund revokes, downgrade | `e2e/journeys/saas-j02-billing.spec.ts` and step 2 | PIX payment through the fake AbacatePay, the refund webhook and the revoked plan; the Stripe checkout, subscription, refund and dispute paths through the Go tests with the fake Stripe | Stripe test mode is not driven from a browser, and the sign-up with email verification of the saas edition |
| 3 | Family: invite, roles, viewer read-only, removal | `e2e/journeys/j03-family.spec.ts` | invitation link, registration, sign-in, a viewer with no write button and no invite form, removal and the loss of access | |
| 4 | Cards and installments across a statement boundary on day 31 | `e2e/journeys/j04-cards-day31.spec.ts` | a card closing on day 31, a purchase on the closing day and one the day after, three installments, closings clamped to Feb 28 | |
| 5 | Recurring bills after downtime | `e2e/journeys/j05-recurring-downtime.spec.ts` and step 5 | backfill of a rule started three months ago, idempotent on reload; the scheduled catch-up and its cap through the Go tests | a browser cannot move the server clock, so the real downtime is the Go test |
| 6 | AI categorization and the assistant | `e2e/journeys/j06-ai.spec.ts` and step 6 | the fake provider, a keyboard review of a suggestion, the assistant answering from the server tables | |
| 7 | Telegram: link, entry, undo, unlink | `e2e/journeys/j07-telegram.spec.ts` and step 7 | the fake Bot API (`e2e/fake-telegram.mjs`, extended with inline-button taps): link by code, an expense by chat, the draft saves nothing until the tap, undo, unlink | |
| 8 | Mobile offline edits, pairing, two-way sync with conflicts | step 8 (`scripts/mobile-sync-e2e.sh`) | the mobile sync client against the real server, the shared scenarios and the convergence property test | the Expo app on a device |
| 9 | CLI over the API and in local mode | step 9 | `TestCLILocalAndRemoteAgree` and the CLI unit tests | |
| 10 | MCP over stdio with a scripted client and remote with OAuth | step 10 | initialize, tool list against the registry, the destructive gate, the HTTP transport, OAuth consent, isolation, scopes and grants | |
| 11 | Desktop headless smoke | step 11 (`desktop/scripts/smoke.sh`) | the binary starts without a window, follows the entry URL, refuses a request without the cookie, stops and frees the port | the window, the menu and the installers |
| 12 | Landing page and docs site build checks | step 12 | `bun run typecheck`, `check:claims`, `build` and `check:built` in `site/` | the site is the work of another epic: this step runs its checks and changes nothing in `site/` |

The browser journeys leave the demo data as the other specs expect it (they delete what they create). Journey 1 imports
a fixed statement (`tests/fixtures/csv/journey-statement.csv`), so a second run needs a new database; every Playwright
run makes one.

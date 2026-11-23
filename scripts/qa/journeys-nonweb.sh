#!/usr/bin/env bash
# Runs the journeys of docs/qa/journeys.md that have no browser (epic 24, task 13): the CLI, MCP over stdio and
# remote with OAuth, the Telegram bot, mobile sync, the desktop headless smoke, backup and restore, the payment
# webhooks and the site build checks. Providers are always the fakes (Stripe, AbacatePay, Telegram, AI): nothing
# here reaches a real service. The browser journeys are `bunx playwright test e2e/journeys`.
#
# Needs Go, Bun, Node 22.13 or newer, and `bun install --frozen-lockfile` in the root, mobile/ and site/.
set -uo pipefail

cd "$(dirname "$0")/../.."

failed=()

step() {
	local name="$1"
	shift
	echo
	echo "=== journey ${name}"
	if "$@"; then
		echo "--- ${name}: ok"
	else
		echo "--- ${name}: FAILED"
		failed+=("${name}")
	fi
}

site_checks() {
	(
		set -e
		cd site
		bun install --frozen-lockfile
		bun run typecheck
		bun run check:claims
		bun run build
		bun run check:built
	)
}

# 1 (the part without a browser): backup and restore of the personal database.
step "1 backup and restore" go run ./scripts/qa/drills backup-restore
# 2 (the part without a browser): Stripe and PIX webhooks against the fake providers, refund and dispute revoke.
step "2 billing webhooks" go test ./tests/ -count=1 \
	-run 'TestStripeWebhookTurnsTheWorkspaceIntoPro|TestCheckoutStartsATrialForTheWorkspace|TestPixPollActivatesAndTheBillingPageShowsIt|TestRefundRevokesPlan|TestRefundRevokesWhenThePeriodIsCurrentOrUnknown|TestDisputeRevokesPlan|TestOnlyTheOwnerManagesBilling'
# 5: recurring bills after downtime.
step "5 recurring catch-up" go test ./tests/ -count=1 -run 'TestCatchUpAfterDowntime|TestCatchUpIsCapped|TestGenerateIsIdempotent'
# 6: AI categorization and the assistant against the fake provider.
step "6 ai" go test ./tests/ -count=1 -run 'TestCategorizeJob|TestAIFeaturesHTTP|TestAutoApplyOffByDefault|TestUndoAutoApplied|TestConversationsArePrivate'
# 7: Telegram link, entry, undo and unlink against the fake Bot API.
step "7 telegram" go test ./tests/ -count=1 -run 'TestLinkFlow|TestEntryFlow|TestUndoWindow|TestDoubleTapCreatesOne|TestBotCommands'
# 8: the mobile client against the real server.
step "8 mobile sync" bash scripts/mobile-sync-e2e.sh
# 9: the CLI over the API and in local mode.
step "9 cli" go test ./tests/ ./app/cli/... -count=1 -run 'TestCLILocalAndRemoteAgree|TestLocalMode|TestRemote$|TestOutput$|TestExitCodes'
# 10: MCP over stdio with a scripted client and MCP remote with OAuth.
step "10 mcp" go test ./tests/ ./app/mcp/... -count=1 -run 'TestInitialize$|TestToolsMatchRegistry|TestStdoutOnlyProtocol|TestDestructiveGate|TestHTTPTransport$|TestOAuthConsent$|TestMCPRemoteIsolation|TestMCPScopesAreFixedAtConsent|TestMCPGrantsEndWithTheirPerson'
# 11: the desktop app without a window.
step "11 desktop headless smoke" bash desktop/scripts/smoke.sh
# 12: the landing page and the docs site build checks (owned by the site epic; run, not changed, here).
step "12 site build checks" site_checks

echo
if [ "${#failed[@]}" -gt 0 ]; then
	echo "journeys failed: ${failed[*]}"
	exit 1
fi
echo "all non-web journeys passed"

#!/usr/bin/env bash
# Runs the whole Playwright suite with the server output captured, then scans
# every captured log (and the Playwright output) for secrets (epic 24, task 7).
# Exit 0 only when the suite passed and the scanner found nothing.
set -euo pipefail

cd "$(dirname "$0")/../.."

out="${LOGSCAN_DIR:-$(mktemp -d "${TMPDIR:-${TEMP:-.}}/vf-logscan.XXXXXX")}"
mkdir -p "$out"
export E2E_LOG_DIR="$out"
known="$out.known"
echo "logscan: captured output goes to $out"

suite=0
bunx playwright test 2>&1 | tee "$out/playwright.log" || suite=$?
# pipefail makes the status the suite's, not tee's.

# The literal secrets the run was configured with must never show up either.
if [ -f e2e/.tmp/env.json ]; then
	bun -e '
		const { appEnv, saasEnv } = JSON.parse(await Bun.file("e2e/.tmp/env.json").text());
		const keys = ["APP_KEY", "FINANCE_ADMIN_PASSWORD", "TELEGRAM_BOT_TOKEN", "ABACATEPAY_API_KEY", "ABACATEPAY_WEBHOOK_SECRET"];
		const values = new Set();
		for (const env of [appEnv, saasEnv]) for (const k of keys) if (env[k]) values.add(env[k]);
		console.log([...values].join("\n"));
	' >"$known"
else
	: >"$known"
fi

scan=0
go run ./scripts/qa/logscan -known "$known" "$out" || scan=$?

if [ "$suite" -ne 0 ]; then
	echo "logscan: the end-to-end suite failed (status $suite)" >&2
	exit "$suite"
fi

exit "$scan"

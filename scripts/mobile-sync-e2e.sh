#!/usr/bin/env bash
# Runs the mobile sync client against the real Go server over HTTP (epic 20, task 11).
#
# It builds the server, starts it on a free port with a throwaway SQLite file and the seeded
# workspace, pairs client engines through the real HttpTransport, runs the shared scenarios that
# one owner of a Pro workspace can run (testdata/sync), a retry after a lost response and the
# convergence property test with 50 seeded runs, then stops the server and proves the port is
# closed. The Jest suite is mobile/__tests__/integration/e2e.test.ts, which `bunx jest` skips by
# configuration: it needs this server.
#
# Needs: go, node 22.13 or newer, bun, curl, and `cd mobile && bun install --frozen-lockfile` done.
# Nothing here reaches a network other than 127.0.0.1, and nothing is written outside a temp dir.
set -euo pipefail

cd "$(dirname "$0")/.."

WORK="$(mktemp -d -t vf-sync-e2e.XXXXXX)"
SERVER_PID=""

stop_server() {
	if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
		kill "$SERVER_PID" 2>/dev/null || true
		wait "$SERVER_PID" 2>/dev/null || true
	fi
	SERVER_PID=""
}

cleanup() {
	status=$?
	stop_server
	if [ "$status" -ne 0 ] && [ -f "$WORK/server.log" ]; then
		echo "--- the last failed requests of the server log ---" >&2
		grep -E "status=(4|5)[0-9][0-9]" "$WORK/server.log" | tail -n 25 >&2 || true
	fi
	rm -rf "$WORK"
	exit "$status"
}
trap cleanup EXIT

# The Go binary wants a Windows path for the database file when this runs in Git Bash.
native() {
	if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi
}

EXE=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) EXE=".exe" ;; esac

PORT="$(node -e 'const s=require("net").createServer().listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')"
ADMIN_EMAIL="sync-e2e@example.test"
ADMIN_PASSWORD="Aa1-$(node -e 'console.log(require("crypto").randomBytes(12).toString("hex"))')"

APP="$WORK/app$EXE"
echo "building the server"
go build -o "$APP" ./cmd/app

export APP_NAME=vukem-finances
export APP_ENV=development
export APP_KEY="base64:$(node -e 'console.log(require("crypto").randomBytes(32).toString("base64"))')"
export APP_EDITION=personal
export PORT
export APP_URL="http://127.0.0.1:$PORT"
export DATABASE_URL="file:$(native "$WORK")/db.sqlite"
export ADMIN_EMAILS="$ADMIN_EMAIL"
export FINANCE_ADMIN_EMAIL="$ADMIN_EMAIL"
export FINANCE_ADMIN_PASSWORD="$ADMIN_PASSWORD"
export MAIL_MAILER=log
# Every client of this run comes from 127.0.0.1 and names its own address in X-Forwarded-For, so
# each device has its own rate limit buckets, as phones on different networks would.
export TRUSTED_PROXIES=127.0.0.1

"$APP" migrate
"$APP" db:seed FinanceAdminSeeder FinanceDemoSeeder

echo "starting the server on port $PORT"
"$APP" >"$WORK/server.log" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 120); do
	if curl -fsS -o /dev/null "http://127.0.0.1:$PORT/login" 2>/dev/null; then break; fi
	if ! kill -0 "$SERVER_PID" 2>/dev/null; then echo "the server exited" >&2; exit 1; fi
	sleep 0.5
done
curl -fsS -o /dev/null "http://127.0.0.1:$PORT/login"

echo "running the sync suite"
(
	cd mobile
	SYNC_E2E_URL="http://127.0.0.1:$PORT" \
		SYNC_E2E_DB="$(native "$WORK")/db.sqlite" \
		SYNC_E2E_ADMIN_EMAIL="$ADMIN_EMAIL" \
		SYNC_E2E_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
		bunx jest --runInBand --testPathIgnorePatterns '/node_modules/' --runTestsByPath __tests__/integration/e2e.test.ts
)

echo "stopping the server"
stop_server

# The last step: the port must refuse connections now.
if node -e 'const s=require("net").connect({host:"127.0.0.1",port:Number(process.argv[1])});s.on("connect",()=>process.exit(1));s.on("error",()=>process.exit(0))' "$PORT"; then
	echo "ok: the server stopped, port $PORT refuses connections"
else
	echo "the server is still listening on port $PORT" >&2
	exit 1
fi

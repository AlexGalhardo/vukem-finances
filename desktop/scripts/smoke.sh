#!/usr/bin/env bash
# Smoke test of the desktop app without a window: starts the binary in headless
# mode, follows the entry URL to the app, checks that a request without the
# cookie is refused, then stops the process and proves the port is closed.
#
# Usage: bash desktop/scripts/smoke.sh [path-to-binary]
# Without an argument the script builds the binary from desktop/.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
pid=""

cleanup() {
	if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
		kill "$pid" 2>/dev/null || true
		wait "$pid" 2>/dev/null || true
	fi
	rm -rf "$work"
}
trap cleanup EXIT

fail() {
	echo "smoke: FAIL: $*" >&2
	if [ -f "$work/stderr.log" ]; then
		sed 's/^/smoke: stderr: /' "$work/stderr.log" >&2 || true
	fi
	exit 1
}

status() { curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$@" || true; }

bin="${1:-}"
if [ -z "$bin" ]; then
	exe=""
	case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
	bin="$work/vukem-finances-desktop$exe"
	(cd "$here" && go build -o "$bin" .)
fi

"$bin" --headless --print-url --data-dir "$work/data" >"$work/stdout.log" 2>"$work/stderr.log" </dev/null &
pid=$!

url=""
for _ in $(seq 1 150); do
	url="$(head -n 1 "$work/stdout.log" 2>/dev/null | tr -d '\r' || true)"
	[ -n "$url" ] && break
	kill -0 "$pid" 2>/dev/null || fail "the app exited before printing the entry URL"
	sleep 0.2
done
[ -n "$url" ] || fail "no entry URL after 30s"

origin="${url%%/desktop/enter/*}"
port="${origin##*:}"
case "$url" in http://127.0.0.1:*/desktop/enter/*) ;; *) fail "unexpected entry URL shape: ${url%%/enter/*}/enter/..." ;; esac

# 1. A request without the cookie is refused, even the health check.
[ "$(status "$origin/up")" = "403" ] || fail "GET /up without the cookie must answer 403"
[ "$(status "$origin/")" = "403" ] || fail "GET / without the cookie must answer 403"

# 2. A wrong Host is refused even with the right cookie (DNS rebinding).
jar="$work/cookies.txt"
[ "$(status -c "$jar" "$url")" = "303" ] || fail "the entry URL must answer 303 and set the cookie"
[ "$(status -b "$jar" "$origin/up")" = "200" ] || fail "GET /up with the cookie must answer 200"
[ "$(status -b "$jar" -H 'Host: attacker.example' "$origin/up")" = "403" ] || fail "a foreign Host must answer 403"

# 3. The launch token works once.
[ "$(status "$url")" = "401" ] || fail "the spent token must answer 401"

# 4. The first page after the exchange is the setup form.
[ "$(status -b "$jar" "$origin/setup")" = "200" ] || fail "GET /setup with the cookie must answer 200"

# 5. Stop the process and prove the port is closed.
kill "$pid"
for _ in $(seq 1 100); do
	kill -0 "$pid" 2>/dev/null || break
	sleep 0.2
done
if kill -0 "$pid" 2>/dev/null; then fail "the process did not stop"; fi
wait "$pid" 2>/dev/null || true
pid=""

[ "$(status "http://127.0.0.1:$port/up")" = "000" ] || fail "port $port still accepts connections after the stop"

echo "smoke: OK (entry, cookie, 403 without it, token spent once, stop, port closed)"

#!/usr/bin/env bash
# Proves the SQLite file survives a container recreation: create the owner
# through /setup, recreate the app container, and check that the app still
# knows the owner (GET / goes to /login, not back to /setup).
#
# Needs Docker, a .env with APP_KEY (cp .env.example .env) and ports 3000 and
# 8025 free. It builds the local stack, uses its own volume and removes it.
set -euo pipefail

cd "$(dirname "$0")/.."

COMPOSE=(docker compose -p vf-persist)
URL=http://127.0.0.1:3000
PW="persist-$(date +%s)-$RANDOM-$RANDOM"

fail() { echo "FAIL: $*" >&2; exit 1; }
cleanup() { "${COMPOSE[@]}" down -v >/dev/null 2>&1 || true; }
trap cleanup EXIT

[ -f .env ] || fail ".env is missing: cp .env.example .env and set APP_KEY"

"${COMPOSE[@]}" up -d --build --wait

landing() { curl -s -o /dev/null -w '%{redirect_url}' "$URL/"; }

[ "$(landing)" = "$URL/setup" ] || fail "a fresh volume should send / to /setup, got '$(landing)'"

code="$("${COMPOSE[@]}" logs app | grep -o 'setup code: [A-Z0-9]*' | tail -1 | cut -d' ' -f3)"
[ -n "$code" ] || fail "no setup code in the boot log (is SETUP_TOKEN set in .env? then export it here)"

status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "$URL/setup" \
	-H 'Content-Type: application/json' -H 'Accept: application/json' -H 'Sec-Fetch-Site: same-origin' \
	-d "{\"name\":\"Owner\",\"email\":\"owner@example.test\",\"currency\":\"BRL\",\"code\":\"$code\",\"password\":\"$PW\",\"password_confirmation\":\"$PW\"}")"
case "$status" in
	200 | 302 | 303) ;;
	*) fail "/setup answered $status" ;;
esac

[ "$(landing)" = "$URL/login" ] || fail "after /setup, / should go to /login, got '$(landing)'"

echo "recreating the app container"
"${COMPOSE[@]}" up -d --force-recreate --wait app

[ "$(landing)" = "$URL/login" ] || fail "the owner is gone after the recreate: / went to '$(landing)'"

echo "OK: the data survived the container recreation"

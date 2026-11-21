#!/usr/bin/env bash
# Proves that, behind Caddy, the rate limiter keys on the real client address
# and not on Caddy's, and that a spoofed X-Forwarded-For is only honoured from
# a trusted proxy (the compose.prod.yaml network, TRUSTED_PROXIES).
#
# Needs the production stack up and no owner created yet (POST /setup is the
# throttled route: five tries a minute per address):
#
#   APP_DOMAIN=localhost docker compose -f compose.prod.yaml up -d --build --wait
#   bash scripts/docker-clientip-test.sh
set -euo pipefail

# Git Bash on Windows rewrites container paths such as /dev/null.
export MSYS_NO_PATHCONV=1

cd "$(dirname "$0")/.."

export APP_DOMAIN="${APP_DOMAIN:-localhost}"
COMPOSE=(docker compose -f compose.prod.yaml)
PROJECT="$("${COMPOSE[@]}" config --format json | sed -n 's/^  "name": "\(.*\)",$/\1/p' | head -1)"
NETWORK="${PROJECT}_web"
UNTRUSTED="${PROJECT}_untrusted"
PW="probe-$(date +%s)-$RANDOM-$RANDOM"
BODY="{\"name\":\"Probe\",\"email\":\"x@example.test\",\"currency\":\"BRL\",\"code\":\"wrong\",\"password\":\"$PW\",\"password_confirmation\":\"$PW\"}"

fail() { echo "FAIL: $*" >&2; exit 1; }

cleanup() {
	docker network disconnect "$UNTRUSTED" "$APP_CONTAINER" >/dev/null 2>&1 || true
	docker network rm "$UNTRUSTED" >/dev/null 2>&1 || true
}
trap cleanup EXIT

APP_CONTAINER="$("${COMPOSE[@]}" ps -q app)"
[ -n "$APP_CONTAINER" ] || fail "the production stack is not running"
IMAGE="$(docker inspect -f '{{.Config.Image}}' "$APP_CONTAINER")"

# The limiter state is in memory: a restart gives each section a clean window.
reset_app() {
	"${COMPOSE[@]}" restart app >/dev/null
	for _ in $(seq 1 60); do
		[ "$(docker inspect -f '{{.State.Health.Status}}' "$APP_CONTAINER")" = healthy ] && return 0
		sleep 1
	done
	fail "the app did not become healthy again"
}

# post_via_caddy <forged X-Forwarded-For> prints the HTTP status.
post_via_caddy() {
	curl -sk -o /dev/null -w '%{http_code}' -X POST "https://${APP_DOMAIN}/setup" \
		-H 'Content-Type: application/json' -H 'Accept: application/json' \
		-H 'Sec-Fetch-Site: same-origin' -H "X-Forwarded-For: $1" -d "$BODY"
}

# post_direct <network> <forged X-Forwarded-For> prints the HTTP status. The
# request comes from a throwaway container of the app image (busybox wget) on
# <network>, straight to the app, with no Caddy in between.
post_direct() {
	{ docker run --rm --network "$1" --entrypoint wget "$IMAGE" -S -O /dev/null \
		--header 'Content-Type: application/json' --header 'Accept: application/json' \
		--header 'Sec-Fetch-Site: same-origin' --header "X-Forwarded-For: $2" \
		--post-data "$BODY" http://app:3000/setup 2>&1 || true; } | grep -o 'HTTP/1\.[01] [0-9]*' | tail -1 | cut -d' ' -f2
}

# Counts how many of six requests were throttled (429); the others were 403.
count_throttled() {
	THROTTLED=0
	local status
	for status in "$@"; do
		case "$status" in
			429) THROTTLED=$((THROTTLED + 1)) ;;
			403) ;;
			*) fail "unexpected status '$status' (is an owner already created?)" ;;
		esac
	done
}

echo "1. through Caddy, a different forged X-Forwarded-For on every request"
reset_app
statuses=()
for i in 1 2 3 4 5 6; do statuses+=("$(post_via_caddy "10.9.0.$i")"); done
echo "   statuses: ${statuses[*]}"
count_throttled "${statuses[@]}"
[ "$THROTTLED" -ge 1 ] \
	|| fail "the forged header was honoured: Caddy must replace it with the real client address"

echo "2. straight to the app from a trusted peer, a different X-Forwarded-For each time"
reset_app
statuses=()
for i in 1 2 3 4 5 6; do statuses+=("$(post_direct "$NETWORK" "10.9.1.$i")"); done
echo "   statuses: ${statuses[*]}"
count_throttled "${statuses[@]}"
[ "$THROTTLED" -eq 0 ] \
	|| fail "the limiter ignored X-Forwarded-For from a trusted proxy (it keys on the peer address)"

echo "3. straight to the app from an untrusted peer, a different X-Forwarded-For each time"
reset_app
docker network create "$UNTRUSTED" >/dev/null
docker network connect --alias app "$UNTRUSTED" "$APP_CONTAINER"
statuses=()
for i in 1 2 3 4 5 6; do statuses+=("$(post_direct "$UNTRUSTED" "10.9.2.$i")"); done
echo "   statuses: ${statuses[*]}"
count_throttled "${statuses[@]}"
[ "$THROTTLED" -ge 1 ] \
	|| fail "a spoofed X-Forwarded-For from an untrusted peer was honoured"

echo "OK: the limiter sees the real client address"

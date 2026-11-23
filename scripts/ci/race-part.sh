#!/usr/bin/env bash
# Runs one part of the Go tests under the race detector, so CI can spread them over parallel jobs.
#
#   race-part.sh rest   every package except ./tests
#   race-part.sh N      part N (1 to 4) of ./tests, the slowest package by far
#   race-part.sh --list N   prints the test names of part N and runs nothing
set -euo pipefail

cd "$(dirname "$0")/../.."

parts=4

if [ "${1:-}" = "rest" ]; then
	mapfile -t packages < <(go list ./... | grep -v '/tests$')
	exec go test -race -count=1 -timeout=15m "${packages[@]}"
fi

list=false
if [ "${1:-}" = "--list" ]; then
	list=true
	shift
fi

part="${1:-}"
case "$part" in
[1-9]) [ "$part" -le "$parts" ] || { echo "part must be rest or 1 to $parts" >&2; exit 2; } ;;
*) echo "part must be rest or 1 to $parts" >&2; exit 2 ;;
esac

# One race build serves both the listing and the run. Fuzz functions count: a plain run executes their seeds.
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
go test -race -c -o "$out/tests.test" ./tests
cd tests

names="$("$out/tests.test" -test.list '.*' | grep -E '^(Test|Fuzz|Example)' | sort |
	awk -v part="$part" -v parts="$parts" 'NR % parts == part - 1' | paste -sd'|' -)"
if [ -z "$names" ]; then
	echo "part $part has no tests" >&2
	exit 1
fi

if [ "$list" = true ]; then
	tr '|' '\n' <<<"$names"
	exit 0
fi

"$out/tests.test" -test.count=1 -test.timeout=15m -test.run "^($names)\$"

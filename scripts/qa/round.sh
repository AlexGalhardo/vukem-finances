#!/usr/bin/env bash
# Runs one full QA round (epic 24, tasks 14 and 15): every check of the campaign, then the counts of docs/qa/rounds.md.
# Exit 0 only when every check passed and no Critical or High problem is open. A passing round is appended to the
# round log, whose "new" column is what the exit rule reads: two consecutive rows with 0 end the campaign.
#
#   bash scripts/qa/round.sh          run everything and log the round
#   bash scripts/qa/round.sh --count  print the counts only
set -uo pipefail

cd "$(dirname "$0")/../.."

rounds=docs/qa/rounds.md
accepted=docs/qa/accepted.md

# A "partly" fixed problem counts as accepted only while docs/qa/accepted.md names what is left of it.
counts() {
	awk '
		function settle() {
			if (id == "") return
			recorded++
			if (status == "fixed") fixed++
			else if (status == "accepted" || (status == "partly" && (id in listed))) taken++
			else {
				open++
				if (severity == "Critical" || severity == "High") blocking++
			}
		}
		FNR == NR {
			line = $0
			while (match(line, /QA-[0-9]+/)) {
				listed[substr(line, RSTART, RLENGTH)] = 1
				line = substr(line, RSTART + RLENGTH)
			}
			next
		}
		/^##+ QA-[0-9]+:/ {
			settle()
			match($0, /QA-[0-9]+/)
			id = substr($0, RSTART, RLENGTH)
			severity = ""
			status = ""
			next
		}
		/^- \*\*Severity:\*\* / && severity == "" { severity = $3; gsub(/[^A-Za-z]/, "", severity) }
		/^- \*\*Status:\*\* / && status == "" { status = $3; gsub(/[^A-Za-z]/, "", status) }
		END {
			settle()
			printf "%d %d %d %d %d\n", recorded, fixed, taken, open, blocking
		}
	' "$accepted" "$rounds"
}

# The recorded count of the last logged round; empty before the first one.
last_recorded() {
	awk -F'|' '/^## Round log/ { log_on = 1; next } log_on && $2 ~ /^ *[0-9]+ *$/ { n = $5 + 0; seen = 1 } END { if (seen) print n }' "$rounds"
}

read -r recorded fixed taken open blocking < <(counts)
if [ "${1:-}" = "--count" ]; then
	echo "recorded=${recorded} fixed=${fixed} accepted=${taken} open=${open} open_critical_high=${blocking}"
	[ "$blocking" -eq 0 ]
	exit
fi

failed=()

step() {
	local name="$1"
	shift
	echo
	echo "=== ${name}"
	if "$@"; then
		echo "--- ${name}: ok"
	else
		echo "--- ${name}: FAILED"
		failed+=("${name}")
	fi
}

step "go build" go build ./...
step "go vet" go vet ./...
step "go test" go test ./...
step "exported scaffold" go test -count=1 -timeout=15m -run TestExportedTreeBuilds ./scripts/export-scaffold/
step "typecheck" bun run typecheck
step "lint js" bun run lint:js
step "lint md" bun run lint:md
step "web build" bun run build
step "qa inventory" go run ./scripts/qa/inventory -check
step "govulncheck" go tool govulncheck ./...
step "bun audit" bun audit --audit-level=high
step "recovery drills" bash scripts/qa/drills.sh
step "e2e with log scan" bash scripts/qa/run-e2e-with-logscan.sh
step "non-web journeys" bash scripts/qa/journeys-nonweb.sh

# Counted again because the file may have been edited while the checks ran.
read -r recorded fixed taken open blocking < <(counts)
previous="$(last_recorded)"
new=$((recorded - ${previous:-0}))

echo
echo "round: recorded=${recorded} new=${new} fixed=${fixed} accepted=${taken} open=${open} open_critical_high=${blocking}"

if [ "${#failed[@]}" -gt 0 ]; then
	echo "round: failed checks: ${failed[*]}"
	exit 1
fi
if [ "$blocking" -gt 0 ]; then
	echo "round: ${blocking} Critical or High problem(s) still open"
	exit 1
fi

if ! grep -q '^## Round log' "$rounds"; then
	printf '\n## Round log\n\nWritten by `scripts/qa/round.sh` after a round in which every check passed.\n\n' >>"$rounds"
	printf '| Round | Date | Commit | Recorded | New | Fixed | Accepted | Open |\n' >>"$rounds"
	printf '| --- | --- | --- | --- | --- | --- | --- | --- |\n' >>"$rounds"
fi
number=$(($(awk -F'|' '/^## Round log/ { log_on = 1; next } log_on && $2 ~ /^ *[0-9]+ *$/ { n++ } END { print n + 0 }' "$rounds") + 1))
printf '| %d | %s | %s | %d | %d | %d | %d | %d |\n' \
	"$number" "$(date -u +%F)" "$(git rev-parse --short HEAD)" "$recorded" "$new" "$fixed" "$taken" "$open" >>"$rounds"
echo "round: logged as round ${number}"

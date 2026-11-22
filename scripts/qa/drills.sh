#!/usr/bin/env bash
# Runs the recovery drills (epic 24, task 10): backup and restore, upgrade from
# the previous release, a kill during a sync push and a kill during an import.
# Needs Go, git and tar. Everything runs in throwaway directories it removes.
set -euo pipefail

cd "$(dirname "$0")/../.."

go run ./scripts/qa/drills all

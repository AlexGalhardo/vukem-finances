# Release checklist

What must be true before a release branch is merged and tagged. Each row names the command or the place that proves it.
A row is never ticked from memory: run the command on the commit that will ship.

## Gate on the release branch

Run from the repository root. It must exit 0:

```bash
go build ./... && go vet ./... && go test -race ./... && bun run typecheck && bun run lint:js && bun run lint:md && bun run build && go tool govulncheck ./...
```

## Quality

| Check | Proof |
| --- | --- |
| Every check of the campaign passes | `bash scripts/qa/round.sh` exits 0 |
| No Critical or High problem is open | `bash scripts/qa/round.sh --count` prints `open_critical_high=0` |
| The last two rounds found nothing new | The last two rows of the round log in [rounds.md](rounds.md) have 0 in the "New" column |
| Accepted issues are current | Every row of [accepted.md](accepted.md) still applies, and none is High or Critical |
| Dependencies have no High or Critical advisory | `bun audit --audit-level=high` and `go tool govulncheck ./...` |
| Accessibility has a result for every row | [a11y.md](a11y.md); only the checks that need a real device stay "not run" |
| The load test meets the budgets | [perf-budgets.md](perf-budgets.md), `scripts/qa/loadtest` |
| Recovery works | `bash scripts/qa/drills.sh` |
| The site has no form and only same-origin requests | `go run ./scripts/qa/sitecheck site/out` and `bunx playwright test e2e/site-network.spec.ts` |

## Publication

| Check | Proof |
| --- | --- |
| `SECURITY.md` is published | The file is on the release branch |
| The changelog comes from the train | `scripts/release-train.sh cut` wrote it, and its commit links open on `main` |
| CI is green on the release pull request | `gh pr checks <number>` |
| The docs site builds | `cd site && bun run build` |
| The images come from the tag | GHCR and Docker Hub carry `X.Y.Z`, `X.Y` and `latest` for the tagged commit |
| The installers are attached | The GitHub release lists the installers of Windows, macOS and Linux |

## Checks only a person can run

These stay open in the hand-off until the author runs them: the device checks listed as "not run" in [a11y.md](a11y.md),
and the legal review recorded in `QUESTIONS.md` (M-009).

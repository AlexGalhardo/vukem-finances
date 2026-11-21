# Epics

The work lives in `../plans/` as numbered epics, with the index `../plans/00-roadmap.md`. **An epic is one unit of work: one senior takes it from the first task to the hand-off, and the specialist reviews it.** If it does not fit one senior from start to review, split it. There are no phases and no milestones.

## 1. The epic

An epic is `NN-<slug>.md`, numbered in the roadmap, in English, with these sections in order: Goal, Context and related decisions, Scope, Out of scope, Dependencies, Technical design, Tasks, Test strategy, Acceptance criteria, Risks and mitigation, Size (S, M, L or XL), Hand-off for review.

- Every acceptance criterion says how to verify it: a command and its expected output, a number with its limit, or a behaviour someone can observe. "Works" is never an acceptance criterion.
- An epic names the modules it touches. Two epics that touch the same modules are never in progress together.
- A task is small enough to be done and verified in one sitting, and ends with `Verify:` and the command, with the result to expect.
- A task is delivered as one small commit that cites the epic: `Refs: epic NN task N`.
- An epic states its STOP conditions in "Risks and mitigation": what makes the senior stop and report instead of improvising.
- The roadmap holds the order, the dependencies, the decisions `D-NNN` and the status of each epic. Only the orchestrator writes the status.

## 2. Before starting an epic

1. Read the whole epic, `AGENTS.md` and the decisions it cites.
2. Check that what the epic says about the code is still true. If a cited file, function or contract no longer matches, stop and report; the orchestrator refreshes the epic first.
3. The orchestrator sets the status to `IN PROGRESS (<agent>, <date>)`.

## 3. The loop and the completion protocol

```text
read the epic and what it links
for each task, in order:
    do the task
    run its "Verify:" command
    while it fails: find the cause, fix it, run it again      (never skip, retry or weaken the check)
    commit the task (Refs: epic NN task N)
    tick the task in the epic
run every acceptance criterion; fix and repeat until all pass
write the hand-off and give the epic to the orchestrator
```

A task that still fails after three honest attempts at its cause is a STOP.

1. **The senior delivers:** tests passing, linters and type checks clean, documents updated, and the hand-off written.
2. **The specialist reviews with no prior context.** The orchestrator opens a new specialist session and gives it only the epic, its acceptance criteria and the diff against the integration branch. Never the senior's report, never an opinion.
3. **Only the orchestrator marks an epic done**, after the specialist approved: it integrates the branch, pushes the release branch, cuts the release when due, writes `DONE (<date>, <commits>)` and stops the agents.
4. An epic that cannot meet a criterion is not changed silently: write `BLOCKED (<reason>)`, then fix the specification first or split the epic.
5. A delivery that fails review twice stops: write `BLOCKED (<reason>)`, record the question for the author with a recommendation, and dispatch the next unblocked epic.
6. Every friction with the Vukem framework goes to `../vukem-feedbacks.md` as a `VF-NNN`. A product problem found while working goes to `development/feedback.md` as an `FB-NNN`, not into an epic.
7. The author confirms before anything that cannot be undone or that costs money: a push or merge to `main` outside the release train, repository settings, secrets, a new dependency, a decision that contradicts the roadmap, or a change of an epic's scope.

## 4. Shared files

Touched by one epic at a time: `go.mod`, `package.json`, `routes/`, `database/migrations/registry.go`, `app/models/`, locale files, `.github/workflows/` and `../plans/00-roadmap.md`. When two epics in progress need one of them, the orchestrator orders the merges. Generated files are written only by their generators.

## 5. The feedback queue

Bugs and problems found by testing are recorded as `FB-NNN` with the steps, what was expected, what happened and the evidence.

1. An important fix is worked like a small epic: one open `FB` to a free senior, on its own branch, reviewed by the specialist.
2. A fix that turns out large, or that changes a finance rule, goes back to the orchestrator, who turns it into an epic.
3. Write the test that reproduces the problem first. Commit with `Refs: FB-NNN`.
4. Only the orchestrator closes an entry, after the specialist approved. An id is never reused.

## 6. The QA campaign

When every epic is done or blocked, the work becomes rounds.

1. **A round tests the whole product end to end**, each senior on its own surface: web, public API, bot, sync, desktop, mobile, Docker install, and both editions. Coverage is counted from inventories (`vukem route:list`, the page list, the bot commands), not from memory.
2. **A senior that tests does not fix.** It records each problem as an `FB` entry.
3. **After the round, the problems are fixed** as in section 5.
4. **Then a new round starts from the beginning** on the new tip of the integration branch.
5. **The campaign ends when a full round finds no problem**, or only problems the author accepted as known.

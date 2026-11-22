---
title: "How epics work"
description: "The unit of work of the project: an epic with a goal, tasks that each end in a verification, acceptance criteria and a review before it joins a release."
---

The work of the project is organized in numbered epics. An epic is one unit of work: one person takes it from the first task to the hand-off, and a reviewer checks it. If it does not fit one person from start to review, it is split.

## What an epic contains

An epic is a document with these sections, in order: Goal, Context and related decisions, Scope, Out of scope, Dependencies, Technical design, Tasks, Test strategy, Acceptance criteria, Risks and mitigation, Size and Hand-off for review.

- Every acceptance criterion says how to verify it: a command and its expected output, a number with its limit, or a behaviour someone can observe. "Works" is never a criterion.
- An epic names the modules it touches. Two epics that touch the same modules are never in progress together.
- A task is small enough to be done and verified in one sitting and ends with its verification.
- An epic states its stop conditions: what makes the person stop and report instead of improvising.

## The loop

1. Read the whole epic, `AGENTS.md` and the decisions the epic cites.
2. Check that what the epic says about the code is still true. If a cited file, function or contract no longer matches, stop and report.
3. Do one task, verify it, and commit it as one small Conventional Commit that cites the epic.
4. At the end, run the whole gate and hand the branch to the reviewer with the commands and their results.

## Decisions

Product and architecture decisions are recorded with an id such as D-003 (the two editions) and cited by the epics that depend on them. A decision that is irreversible, costs money or touches data is never made by the implementer: the implementer stops and reports the options.

## Review

The reviewer reads the diff, runs the validation again and fixes the last bugs. A report from the implementer is a claim, not evidence. Only then does the branch join the release train; see [the release train](/docs/contribute/release-train/).

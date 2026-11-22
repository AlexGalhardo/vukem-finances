---
title: "The release train"
description: "How work reaches a release: one open release branch, linear commits, Conventional Commits with fixed scopes, and a cut at about 90 commits."
---

The process is a release train. The open release pull request is the integration line, approved work joins it as linear commits, and each release is rebase-merged into `main` when it holds about 90 commits. Versions start at 0.1.0 and go up by minor versions.

## The rules

1. Nobody commits or pushes to `main` directly. `main` changes only through the rebase-merge of a release pull request.
2. The integration branch is the open release branch, `release/vX.Y.Z`. It is linear: no merge commit, ever, because GitHub refuses to rebase a pull request that needs one.
3. Every epic is a branch from the tip of the release branch, named `epic/NN-<slug>`, and a fix of the feedback queue is `fix/FB-NNN-<slug>`. These branches are never pushed.
4. Only the orchestrator pushes, and only the open release branch, after integrating approved work.
5. Each agent works in its own worktree outside the repository folder.
6. `scripts/release-train.sh` has three commands: `status`, `integrate` and `cut`. `integrate` cherry-picks the commits of a branch onto the release branch with `-x`.
7. Never `git stash`, `git reset --hard`, `git checkout -- .` or a force push.

## Commits

Commits follow Conventional Commits: `type(scope): subject`. The header is at most 100 characters and the subject starts with a lower-case letter and has no final period. The scope is one of the lines of `.github/scopes.txt`, the single list read by the commit hook and the pull request title check. The type decides the section of the release notes: `feat` goes to Added, `fix` to Fixed, a breaking change to Changed, `revert` to Removed.

Each task of an epic is one small commit that cites the epic in its body.

## The cut

At 90 commits or more the train stamps the version, writes the changelog section of the release from the commit history, commits `chore(release): vX.Y.Z`, rebase-merges the pull request, creates the GitHub release and starts the next release branch. Nobody edits `CHANGELOG.md` by hand; the release notes are pages of [the changelog](/docs/changelog/).

## Publishing

A published release builds the container image and pushes it to GHCR and Docker Hub with the tags `X.Y.Z`, `X.Y` and `latest`, and builds the desktop installers for the three systems and attaches them to the release.

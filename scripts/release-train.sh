#!/usr/bin/env bash
# The release train: one open release pull request at a time, on branch release/vX.Y.Z from
# main. Approved work is added to it as linear commits; when it holds about 90 commits it is cut
# (version stamp and changelog in one chore(release) commit), rebase-merged into main, and the
# next release branch is started from the new main.
#
#   release-train.sh status                 the open release branch, its pull request and size
#   release-train.sh integrate <ref>...     cherry-pick the work of <ref> onto the release branch;
#                                           cuts on its own when the branch reaches MAX commits
#   release-train.sh cut [--force]          cut, merge and start the next release (needs MIN
#                                           commits unless --force)
#
# Run it in the main checkout, on the open release branch. It never force-pushes and never
# pushes main: the rebase-merge of the pull request moves main. The GitHub release of each cut
# is created after the merge, with notes regenerated from main (the rebase gives the commits new
# ids), and every commit id linked in CHANGELOG.md is checked against GitHub.
set -euo pipefail

MIN=${RELEASE_TRAIN_MIN:-90}
MAX=${RELEASE_TRAIN_MAX:-95} # the cut adds one commit (chore(release)): 96 at most, GitHub takes 100
URL=https://github.com/AlexGalhardo/vukem-finances
REPO=${URL#https://github.com/}
LIMIT=96
POLL=${RELEASE_TRAIN_POLL:-5}
WAIT_TRIES=${RELEASE_TRAIN_WAIT_TRIES:-120}
CLIFF=(git cliff --offline --config .config/cliff.toml)

die() {
	echo "release-train: $*" >&2
	exit 1
}

branch() { git rev-parse --abbrev-ref HEAD; }

version_of() { # release/v0.2.0 or release/v0.1.0-r2 -> 0.1.0
	sed -n 's#^release/v\([0-9]*\.[0-9]*\.[0-9]*\).*#\1#p' <<<"$1"
}

require_release_branch() {
	local b
	b=$(branch)
	[ -n "$(version_of "$b")" ] || die "the checkout is on $b, not on a release/vX.Y.Z branch"
	# Unstaged local edits may stay (a cherry-pick that touches one of those files stops safely).
	git diff --cached --quiet || die "the index has staged changes"
	git fetch -q origin main
	git merge-base --is-ancestor origin/main HEAD || die "$b does not start from origin/main"
}

count() { git rev-list --count origin/main..HEAD; }

pr_of() { gh pr list --head "$1" --state open --json number --jq '.[0].number // empty'; }

status() {
	local b
	b=$(branch)
	git fetch -q origin main
	echo "branch=$b version=$(version_of "$b") commits=$(count) merges=$(git rev-list --merges --count origin/main..HEAD) pr=$(pr_of "$b")"
}

# The commits of <ref> that are not on main or on any release branch, oldest first, without
# merges, and without those whose patch is already on the release branch.
pending_commits() {
	local ref=$1 c
	# shellcheck disable=SC2046 # one argument per release branch
	git rev-list --reverse --topo-order --no-merges "$ref" --not HEAD origin/main \
		$(git for-each-ref --format='%(refname)' refs/remotes/origin/release/) |
		while read -r c; do
			# Already on the line by patch, or picked before (-x records it; a commit whose conflict
			# was resolved has another patch id, so only the record finds it).
			[ "$(git cherry HEAD "$c" "$c^" 2>/dev/null | cut -c1)" = "-" ] && continue
			[ -n "$(git log -1 --format=%H --fixed-strings --grep="cherry picked from commit $c" HEAD)" ] && continue
			echo "$c"
		done
}

integrate() {
	[ $# -gt 0 ] || die "integrate needs at least one ref"
	require_release_branch
	local ref c
	for ref in "$@"; do
		for c in $(pending_commits "$ref"); do
			if ! git cherry-pick -x "$c" >/dev/null; then
				die "conflict on $(git log -1 --format='%h %s' "$c"): resolve it, run git cherry-pick --continue, then run integrate again"
			fi
			if [ "$(count)" -ge "$MAX" ]; then
				cut_release
			fi
		done
		echo "integrated $ref: $(count) commits on $(branch)"
	done
	git push -q origin "$(branch)"
	ensure_pr
}

# Prints the notes of version $1 for the commits of $2 (a git range), without the header.
render_section() {
	local v=$1 range=$2
	"${CLIFF[@]}" "$range" --tag "v$v" --strip header
}

# The commit range of the release being cut: everything since the last release commit on main.
# $1 is the tip (HEAD while cutting, origin/main after the merge) and $2 how many release
# commits at the tip of main to skip (0 before the merge, 1 after it).
range_since_last_release() {
	local tip=$1 skip=$2 last
	last=$(git log origin/main --format=%H --grep='^chore(release): v' | sed -n "$((skip + 1))p" || true)
	# git-cliff takes only "a..b" ranges. Before the first release the start is the root commit
	# (the initial import, which the changelog skips anyway).
	[ -n "$last" ] || last=$(git rev-list --max-parents=0 "$tip" | tail -n 1)
	echo "$last..$tip"
}

# Replaces the section of version $1, and the body of [Unreleased], in CHANGELOG.md.
write_changelog() {
	local v=$1 section
	section=$(mktemp)
	render_section "$v" "$(range_since_last_release HEAD 0)" >"$section"
	grep -q '^### ' "$section" || printf '## [%s] - %s\n\n### Changed\n\n- Maintenance release (no user-facing changes).\n' "$v" "$(date +%F)" >"$section"
	awk -v sec="$section" -v ver="$v" '
		index($0, "## [" ver "]") == 1 { skipping = 1; next }
		/^## \[Unreleased\]/ && !done {
			print; print ""
			# git-cliff ends the section with blank lines: markdownlint refuses them.
			while ((getline l < sec) > 0) {
				if (l == "") { blanks++; continue }
				for (; blanks > 0; blanks--) print ""
				print l
			}
			blanks = 0; done = 1; skipping = 1; next
		}
		skipping && /^## \[/ { skipping = 0; print "" }
		skipping { next }
		{ print }
	' CHANGELOG.md >CHANGELOG.md.new
	mv CHANGELOG.md.new CHANGELOG.md
	rm -f "$section"
}

# Writes version $1 into the first "version" field of package.json.
stamp_version() {
	local v=$1
	# awk, not sed: BSD sed has no "0,/re/" address.
	awk -v v="$v" '!done && /^[[:space:]]*"version": "/ { sub(/"version": "[^"]*"/, "\"version\": \"" v "\""); done = 1 } { print }' package.json >package.json.new
	mv package.json.new package.json
}

section_of() { # $1 version, $2 git rev:path
	awk -v v="$1" 'index($0, "## [" v "]") == 1 { on = 1; next } on && /^## \[/ { exit } on' <(git show "$2")
}

pr_body() { # $1 version, $2 file to write
	local v=$1
	{
		echo "Release **v$v**: $(count) commits, linear (no merge commit), rebase-merged by the release train."
		echo
		echo "Approved work is added here as it is finished; the release is cut, rebase-merged and the next release started when it holds about $MIN commits."
		echo
		echo "## [$v]"
		section_of "$v" HEAD:CHANGELOG.md
	} >"$2"
}

open_or_update_pr() { # $1 branch, $2 version, $3 body file; prints the number
	local pr
	pr=$(pr_of "$1")
	if [ -z "$pr" ]; then
		pr=$(gh pr create --base main --head "$1" --title "chore(release): v$2" --body-file "$3" | sed 's#.*/##')
	else
		# gh pr edit can fail on an unrelated GraphQL field of the repository; the body is only text.
		gh pr edit "$pr" --title "chore(release): v$2" --body-file "$3" >/dev/null ||
			echo "release-train: the body of #$pr was not updated" >&2
	fi
	echo "$pr"
}

cut_release() {
	local b v pr n next body released=0
	b=$(branch)
	v=$(version_of "$b")
	n=$(count)
	[ "$(git log -1 --format=%s)" = "chore(release): v$v" ] && released=1
	[ "$(git rev-list --merges --count origin/main..HEAD)" = 0 ] || die "$b has a merge commit"
	# The cut adds one commit unless the release commit is already there.
	[ $((n + 1 - released)) -le "$LIMIT" ] || die "$b holds $n commits: the cut makes $((n + 1 - released)), above $LIMIT (GitHub refuses a rebase-merge above 100); split it first"
	if [ "$released" = 0 ]; then
		stamp_version "$v"
		write_changelog "$v"
		git add package.json CHANGELOG.md
		git commit -q -m "chore(release): v$v"
	fi
	git push -q origin "$b"
	body=$(mktemp)
	pr_body "$v" "$body"
	pr=$(open_or_update_pr "$b" "$v" "$body")
	# gh can merge and still exit non-zero (for example on the cleanup of a local branch): trust the state.
	gh pr merge "$pr" --rebase --admin ||
		[ "$(gh pr view "$pr" --json state --jq .state)" = MERGED ] ||
		die "the rebase-merge of #$pr failed"
	local tries=0
	until git fetch -q origin main && [ "$(git rev-parse 'origin/main^{tree}')" = "$(git rev-parse 'HEAD^{tree}')" ]; do
		tries=$((tries + 1))
		[ "$tries" -lt "$WAIT_TRIES" ] || die "main does not carry the tree of $b after $tries checks"
		sleep "$POLL"
	done
	echo "released v$v: #$pr merged into main ($(git rev-parse --short origin/main))"
	publish_release "$v"

	# The next release starts at main; its pull request opens with its first commit (GitHub opens
	# none for a branch equal to its base, and an empty commit breaks a rebase-merge).
	next="0.$(($(cut -d. -f2 <<<"$v") + 1)).0"
	git switch -q -c "release/v$next" origin/main
	git push -q -u origin "release/v$next"
	echo "next release: release/v$next (its pull request opens with the first integrated commit)"
	rm -f "$body"
	check_links "$v"
}

# Creates the GitHub release vX.Y.Z on the release commit of main. The notes are rendered again
# from main, because the rebase-merge gave every commit a new id.
publish_release() {
	local v=$1 notes
	notes=$(mktemp)
	render_section "$v" "$(range_since_last_release origin/main 1)" | sed '1,/^$/d' >"$notes"
	gh release create "v$v" --target "$(git rev-parse origin/main)" --title "v$v" --notes-file "$notes" >/dev/null ||
		echo "release-train: the GitHub release v$v was not created" >&2
	rm -f "$notes"
	echo "published the GitHub release v$v"
}

# Every commit id linked in the section of CHANGELOG.md must exist on GitHub. A failure is a
# stop: the changelog is not edited by hand.
check_links() {
	local v=$1 id bad=0
	for id in $(section_of "$v" origin/main:CHANGELOG.md | grep -o "$REPO/commit/[0-9a-f]\{40\}" | sed 's#.*/##' | sort -u); do
		if ! gh api "repos/$REPO/commits/$id" --jq .sha >/dev/null 2>&1; then
			echo "release-train: the changelog links $id, which GitHub does not answer" >&2
			bad=1
		fi
	done
	[ "$bad" = 0 ] || die "broken commit links in the changelog of v$v: report it, do not edit the changelog by hand"
	echo "changelog links of v$v checked"
}

# Opens the pull request of the release branch when it has commits and none is open.
ensure_pr() {
	local b v body
	b=$(branch)
	v=$(version_of "$b")
	[ "$(count)" -gt 0 ] && [ -z "$(pr_of "$b")" ] || return 0
	body=$(mktemp)
	pr_body "$v" "$body"
	echo "opened $(gh pr create --base main --head "$b" --title "chore(release): v$v" --body-file "$body")"
	rm -f "$body"
}

case "${1:-}" in
status) status ;;
integrate)
	shift
	integrate "$@"
	;;
cut)
	require_release_branch
	[ "${2:-}" = "--force" ] || [ "$(count)" -ge "$MIN" ] || die "$(count) commits: cut at $MIN or more, or pass --force"
	cut_release
	;;
*) die "usage: release-train.sh status | integrate <ref>... | cut [--force]" ;;
esac

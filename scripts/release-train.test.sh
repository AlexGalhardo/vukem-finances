#!/usr/bin/env bash
# Tests scripts/release-train.sh against a local bare remote and a stub gh.
#   bash scripts/release-train.test.sh
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
WORK=$(mktemp -d "${TEMP:-${TMPDIR:-/tmp}}/release-train-test.XXXXXX")
trap 'rm -rf "$WORK"' EXIT

REMOTE=$WORK/origin.git
CLONE=$WORK/clone
STATE=$WORK/state
BIN=$WORK/bin
mkdir -p "$STATE" "$BIN"

pass() { echo "ok - $1"; }
fail() {
	echo "not ok - $1" >&2
	exit 1
}

# The stub gh: "pr merge" rebase-merges into main in the remote like GitHub does (the commits are
# replayed on main and every one gets a new id), "pr create" records the head branch, "api" records the id it was asked.
cat >"$BIN/gh" <<'EOF'
#!/usr/bin/env bash
set -eu
case "$1 $2" in
"pr list") ;;
"pr create")
	while [ $# -gt 0 ]; do
		[ "$1" = "--head" ] && echo "$2" >"$STUB_STATE/head"
		shift
	done
	echo "https://github.com/AlexGalhardo/vukem-finances/pull/1"
	;;
"pr edit") ;;
"pr merge")
	replay=$(mktemp -d "${TEMP:-${TMPDIR:-/tmp}}/release-train-replay.XXXXXX")
	git clone -q "$STUB_REMOTE" "$replay/r" 2>/dev/null
	(
		cd "$replay/r"
		git config core.hooksPath /dev/null
		git switch -q -C main origin/main
		GIT_COMMITTER_DATE="2030-01-01T00:00:00" git cherry-pick "origin/main..origin/$(cat "$STUB_STATE/head")" >/dev/null
		git push -q origin main
	)
	rm -rf "$replay"
	echo merged >"$STUB_STATE/merged"
	;;
"pr view") echo MERGED ;;
"release create")
	printf '%s\n' "$*" >"$STUB_STATE/release"
	while [ $# -gt 0 ]; do
		[ "$1" = "--notes-file" ] && cp "$2" "$STUB_STATE/notes"
		shift
	done
	;;
"api repos/AlexGalhardo/vukem-finances/commits/"*)
	echo "${2#repos/AlexGalhardo/vukem-finances/commits/}" >>"$STUB_STATE/ids"
	;;
*)
	echo "stub gh: unexpected call: $*" >&2
	exit 1
	;;
esac
EOF
chmod +x "$BIN/gh"
export PATH="$BIN:$PATH" STUB_STATE="$STATE" STUB_REMOTE="$REMOTE"
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.test GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.test
export RELEASE_TRAIN_POLL=0

git init -q --bare -b main "$REMOTE"
git clone -q "$REMOTE" "$CLONE" 2>/dev/null
cd "$CLONE"
git config core.hooksPath /dev/null
git config core.autocrlf false
git switch -q -c main
mkdir -p scripts .config
cp "$HERE/scripts/release-train.sh" scripts/
cp "$HERE/.config/cliff.toml" .config/
printf '# Changelog

## [Unreleased]

' >CHANGELOG.md
printf '{\n  "version": "0.0.0",\n  "private": true\n}\n' >package.json
git add -A
git commit -q -m "chore: import the scaffold"
git push -q origin main
git switch -q -c release/v0.1.0
git push -q -u origin release/v0.1.0

section_ids() { # $1 rev, $2 version: the commit ids linked in that section of CHANGELOG.md
	git show "$1:CHANGELOG.md" |
		awk -v v="$2" 'index($0, "## [" v "]") == 1 { on = 1; next } on && /^## \[/ { exit } on' |
		grep -o 'commit/[0-9a-f]\{40\}' | sed 's#.*/##' | sort -u
}

commit_many() { # $1 count, $2 prefix
	local i
	for i in $(seq 1 "$1"); do
		echo "$2 $i" >"file-$2-$i.txt"
		git add "file-$2-$i.txt"
		git commit -q -m "feat(api): add $2 $i"
	done
}

commit_many 91 a
RT="bash scripts/release-train.sh"

[ "$($RT status | sed -n 's/.*commits=\([0-9]*\).*/\1/p')" = 91 ] || fail "status counts 91 commits"
pass "status counts 91 commits"

# Below the minimum the cut is refused unless forced: checked on a scratch release branch.
git switch -q -c release/v0.9.0 origin/main
commit_many 3 s
if $RT cut >/dev/null 2>&1; then fail "a cut with 3 commits is refused"; fi
pass "a cut below 90 commits is refused"
git switch -q release/v0.1.0

$RT cut >"$WORK/cut.log" 2>&1 || {
	cat "$WORK/cut.log" >&2
	fail "cut at 91 commits"
}
pass "cut at 91 commits exits 0"

git fetch -q origin
[ "$(git log -1 --format=%s origin/main)" = "chore(release): v0.1.0" ] || fail "the release commit is on main"
pass "main ends with chore(release): v0.1.0"

[ "$(git rev-list --count origin/main)" = 93 ] || fail "main holds 93 commits (scaffold, 91, release)"
pass "main holds the scaffold, the 91 commits and one release commit"

[ "$(git rev-list --merges --count origin/main)" = 0 ] || fail "main is linear"
pass "main is linear"

git show origin/main:package.json | grep -q '"version": "0.1.0"' || fail "package.json stamped with 0.1.0"
pass "package.json carries version 0.1.0"

git show origin/main:CHANGELOG.md >"$WORK/changelog.md"
grep -q '^## \[0.1.0\] - ' "$WORK/changelog.md" || fail "CHANGELOG.md has the 0.1.0 section"
grep -q '^### Added' "$WORK/changelog.md" || fail "the section has an Added group"
grep -q '^## \[Unreleased\]' "$WORK/changelog.md" || fail "the Unreleased heading stays"
[ "$(grep -c 'https://github.com/AlexGalhardo/vukem-finances/commit/[0-9a-f]\{40\}' "$WORK/changelog.md")" = 91 ] || fail "91 entries link their commit"
pass "CHANGELOG.md has the 0.1.0 section with 91 linked entries"

[ -f "$STATE/release" ] && grep -q 'v0.1.0' "$STATE/release" || fail "the GitHub release v0.1.0 was created"
[ "$(grep -c 'vukem-finances/commit/' "$STATE/notes")" = 91 ] || fail "the release notes link 91 commits"
pass "the GitHub release v0.1.0 was created with linked notes"

[ "$(sort -u "$STATE/ids" | wc -l | tr -d ' ')" = 91 ] || fail "every linked id was checked with gh api"
pass "every linked commit id was checked with gh api"

[ "$(git rev-parse --abbrev-ref HEAD)" = release/v0.2.0 ] || fail "the checkout is on release/v0.2.0"
git ls-remote --exit-code --heads origin release/v0.2.0 >/dev/null || fail "release/v0.2.0 exists on the remote"
[ "$(git rev-parse HEAD~1)" = "$(git rev-parse origin/main)" ] || fail "release/v0.2.0 starts at main"
pass "release/v0.2.0 exists and starts at the new main"

# The rebase gave every commit a new id: the changelog of main still holds the old ones, and the
# first commit of the next release rewrites them to the ids on main.
[ "$(git log -1 --format=%s)" = "docs(release): link the v0.1.0 changelog entries to the commits on main" ] ||
	fail "the first commit of release/v0.2.0 relinks the changelog"
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/release/v0.2.0)" ] || fail "the relink commit is pushed"
new_ids=$(section_ids HEAD 0.1.0)
old_ids=$(section_ids origin/main 0.1.0)
[ "$(wc -l <<<"$new_ids" | tr -d ' ')" = 91 ] || fail "the relinked section has 91 ids"
for id in $new_ids; do
	git merge-base --is-ancestor "$id" origin/main || fail "$id is not on main"
done
for id in $old_ids; do
	if git merge-base --is-ancestor "$id" origin/main; then fail "$id (pre-rebase) is on main: the stub kept ids"; fi
done
pass "the changelog links of v0.1.0 resolve to commits on main after the relink"
for id in $(grep -o 'commit/[0-9a-f]\{40\}' "$STATE/notes" | sed 's#.*/##'); do
	git merge-base --is-ancestor "$id" origin/main || fail "the release notes link $id, which is not on main"
done
pass "the release notes link commits on main"

# Refusal above the limit: the cut adds one commit, so 96 is the most it accepts.
commit_many 97 b
before=$(git rev-parse HEAD)
if $RT cut >"$WORK/refuse.log" 2>&1; then fail "a cut with 97 commits is refused"; fi
grep -q "above 96" "$WORK/refuse.log" || fail "the refusal explains the limit"
[ "$(git rev-parse HEAD)" = "$before" ] || fail "the refused cut changed nothing"
pass "a cut with 97 commits is refused"

# Refusal with a merge commit.
git switch -q -c release/v0.3.0 origin/main
commit_many 2 m
git switch -q -c side
commit_many 1 side
git switch -q release/v0.3.0
git merge -q --no-ff side -m "merge side"
if $RT cut --force >"$WORK/merge.log" 2>&1; then fail "a cut with a merge commit is refused"; fi
grep -q "merge commit" "$WORK/merge.log" || fail "the refusal names the merge commit"
pass "a cut with a merge commit is refused"

# A commit that lands after a release commit, with the tag of that release on it, must not make
# the next changelog repeat the section: git-cliff would print the tag inside the range.
git switch -q -c late origin/main
echo late >late.txt
git add late.txt
git commit -q -m "fix(api): late fix"
git push -q origin late:main
git tag v0.1.0 late
git fetch -q origin
git switch -q -c release/v0.4.0 origin/main
commit_many 3 t
$RT cut --force >"$WORK/cut2.log" 2>&1 || {
	cat "$WORK/cut2.log" >&2
	fail "cut after a late fix"
}
git fetch -q origin
git show origin/main:CHANGELOG.md >"$WORK/changelog2.md"
[ -z "$(grep '^## \[' "$WORK/changelog2.md" | sort | uniq -d)" ] || fail "a section is repeated"
[ "$(awk '/^## \[0.4.0\]/ { on = 1; next } on && /^## \[/ { exit } on' "$WORK/changelog2.md" | grep -c 'commit/')" = 4 ] || fail "the 0.4.0 section holds the late fix and its 3 commits"
pass "no section repeats when a commit lands after a release commit"
git log -1 --format=%s | grep -q '^docs(release): link the .* changelog entries' || fail "the cut relinked the changelog"
for id in $(section_ids HEAD 0.4.0); do
	git merge-base --is-ancestor "$id" origin/main || fail "$id is not on main"
done
pass "every link of the cut version is on main after the relink"

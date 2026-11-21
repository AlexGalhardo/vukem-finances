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

# The stub gh: "pr merge" fast-forwards main in the remote (a rebase-merge of a linear branch
# gives the same tree), "pr create" records the head branch, "api" records the id it was asked.
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
	git --git-dir="$STUB_REMOTE" update-ref refs/heads/main "refs/heads/$(cat "$STUB_STATE/head")"
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
cp "$HERE/CHANGELOG.md" .
printf '{\n  "version": "0.0.0",\n  "private": true\n}\n' >package.json
git add -A
git commit -q -m "chore: import the scaffold"
git push -q origin main
git switch -q -c release/v0.1.0
git push -q -u origin release/v0.1.0

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
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || fail "release/v0.2.0 starts at main"
pass "release/v0.2.0 exists and starts at the new main"

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

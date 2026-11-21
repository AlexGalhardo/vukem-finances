# Commits

1. Conventional Commits: `type(scope): subject`. The header is at most 100 characters, the subject starts with a lower-case letter and has no final period.
2. The scope is one of the lines of `.github/scopes.txt`, the single list read by the commit hook and the pull request title check. A new scope is added there first.
3. Never edit `CHANGELOG.md` by hand. The release cut writes it from the commits.
4. One small, self-contained commit per task of an epic, with `Refs: epic NN task N` in the body (`Refs: FB-NNN` for a fix of the feedback queue). A commit that leaves the tree red in the middle of a series must say so in the hand-off.
5. The commit type decides the section of the release notes (`feat` Added, breaking Changed, `revert` Removed, `fix` Fixed, `Security:` footer Security), so choose it with care.
6. Everything in the repository is in English.

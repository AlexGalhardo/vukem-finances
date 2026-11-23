# The Vukem scaffold export

`vukem new <name> --example finance` creates a project that is this application. The Vukem CLI does
not carry its own copy of the code: it embeds a tree that `scripts/export-scaffold` writes from this
repository, and updates it from a published release archive, never from an unreleased branch.

## Running the export

```sh
go run ./scripts/export-scaffold -out ./scaffold
go run ./scripts/export-scaffold -archive scaffold-v0.7.0.tar.gz -version v0.7.0
```

| Flag | Meaning |
| --- | --- |
| `-out` | Directory to write the tree to. It must not exist or must be empty. |
| `-archive` | Also writes the tree as a `.tar.gz` (sorted entries, no owners, no timestamps). |
| `-ref` | Git revision to export. Default `HEAD`. |
| `-repo` | Path of the repository. Default `.`. |
| `-version` | Text of `VERSION`. Default: the tag at `-ref`, else `dev`. |

The tool reads the blobs of the revision with `git ls-tree` and `git cat-file`, not the working
tree. An export is therefore the same bytes on a Windows checkout (CRLF files, other modes) and on
Linux, and uncommitted work is never exported. Two runs on the same revision give identical bytes
(`diff -r` prints nothing).

## What is in the tree

Every tracked file except:

- Other clients and tooling of this repository: `mobile/`, `desktop/`, `site/`, `clients/`, `e2e/`
  and `playwright.config.ts`, `.github/`, `.agents/`, `.claude/`, `.config/`, `.githooks/`,
  `development/`, `scripts/`, `docs/qa/`, and the documents about the clients and the design work
  (`docs/desktop.md`, `docs/mobile.md`, `docs/site.md`, `docs/design.md`, `docs/design-system.md`,
  `docs/landing-brief.md`).
- Generated or local output: `resources/public/build/`, `storage/`, images (`.png`, `.jpg`, `.gif`,
  `.webp`, `.avif`, `.ico`, `.icns`, `.svg`).
- `tests/doc_snapshots_test.go` and the MCP reference it checks (`docs/reference/mcp.md`): the
  OpenAPI snapshot, the TypeScript client and the MCP reference contain the application name and
  the module path in generated text (descriptions, schema names, token counts), so they can only
  be fresh in this repository. A generated project regenerates them with its own name.
- `go.mod` and `go.sum` (the Vukem CLI writes them for the chosen framework version and runs
  `go mod tidy`), `CHANGELOG.md`, `AGENTS.md` and `CLAUDE.md`.
- `LICENSE`, which is exported as `LICENSE.vukem-finances` (the MIT notice the generated project
  keeps, as `NOTICE`, when the user does not pass `--license`).

Two files inside excluded folders are kept because the app's own tests read them:
`docs/qa/inventory.md` (the authorization case count of the matrix test) and
`site/generated/plans.json` (the plan limits).

`package.json` and `bun.lock` are exported with their exact pins.

Added files: `VERSION` (the release tag) and `MANIFEST.sha256`, one `sha256  path` line per other
file, sorted by path.

## Placeholders

In every text file that is not documentation (`.md`) or a license, the module path
`github.com/AlexGalhardo/vukem-finances` becomes `{{.Module}}` and the application name
`vukem-finances` becomes `{{.Name}}`. A file that receives a placeholder gets the `.tmpl` suffix,
which is the convention of the Vukem scaffolds; the CLI renders those files with `text/template`
(`missingkey=error`) and strips the suffix. References to the app itself stay: a repository URL, the
`AlexGalhardo/` part of an image reference, and the names that the OpenAPI builder mangles from the
module path.

Literal `{{` in a `.tmpl` file (Go templates in strings, JSX `style={{ }}`, docs) would be read as an
action, so the tool writes it as `{{"{{"}}`, which renders back to `{{`. A single `{` just before a
placeholder is written `{{"{"}}`, so `{{{.Name}}` is never produced. Files without a placeholder
are not rendered and keep their `{{` as they are.

## Release asset

`.github/workflows/image.yml` runs the export on a published release and attaches
`scaffold-vX.Y.Z.tar.gz` and `scaffold-vX.Y.Z.MANIFEST.sha256` (the manifest of the archive's tree)
to it. `scripts/update-finance-scaffold.sh` in the Vukem repository downloads both, verifies them,
and replaces its embedded copy.

## Tests

`go test ./scripts/export-scaffold/...` covers the exclusions, the placeholders, rendering of
literal braces, the manifest and determinism. `TestExportedTreeBuilds` exports `HEAD`, renders it
under module `example.test/money` and name `money`, writes a `go.mod` that pins the framework
versions of this repository's `go.mod`, then runs `go mod tidy`, `go build ./...`, `go vet ./...` and
`go test -p 4 ./...` in the result. It is skipped with `-short`.

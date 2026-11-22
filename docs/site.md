# The static site

`site/` is a self-contained static [Astro](https://astro.build) project (its own `package.json` and `bun.lock`, not a workspace of the root) that holds the commercial landing page, at `/`, `/pt-br/` and `/es/`, and the privacy notice. The documentation lives at `/docs/` (epic 23): guides for running, using and integrating the app, and references for the REST API, the CLI and the MCP server that are generated from the sources of the product. The landing page and its claims are described in [landing-brief.md](landing-brief.md).

The site is 100% static (decisions D-019 and D-020): no backend, login, sign-up, checkout, form or call to an API, no cookie and no third-party script, font, image or stylesheet. Every call to action is a plain link to the docs, the GitHub repository, the Docker install page of the docs or the GitHub Releases page. The page says that no hosted service exists.

## Commands

Run them in `site/` after `bun install --frozen-lockfile`.

| Command | What it does |
| --- | --- |
| `bun run dev` | Astro dev server |
| `bun run build` | Builds `site/out` and writes `sitemap.xml`, `robots.txt`, the Markdown twin of every docs page (`/docs/<page>.md`), `llms.txt` and the search index (`/pagefind/`) |
| `go run ./scripts/qa/sitecheck site/out` (repo root) | Audits the built output as an artifact: no form, no resource from another origin, no network call in built JavaScript outside `pagefind/`, no server or worker file, the `CNAME`, no link to an app path (`/login`, `/api`, ...) and no hosted-service phrase on the landing pages; `e2e/site-network.spec.ts` opens every page and records every request |
| `bun run build:search` | Indexes the built docs with Pagefind into `site/out/pagefind/` (part of `build`) |
| `bun run check:built` | Scans `site/out`: no form, other origin, network API, server or worker file; one `h1`, no skipped heading level, landmarks, skip link, alt text, title and description lengths, JSON-LD without a price, hreflang, no hosted-service phrase on the landing page, weight budgets, and a Markdown twin and an `llms.txt` line for every docs page |
| `bun run check:claims` | Checks the claims register of `docs/landing-brief.md` (every evidence path exists), the forbidden phrases of the three catalogs, and the documentation claims: every `app <resource> <verb>` command is in the CLI reference, every setting, repository path, REST route and MCP tool exists, every link and anchor resolves, no bare URL; a failure names the page and line |
| `bun run check:mdx` | Checks every docs page: front matter, title of at most 43 characters, description of at most 160, no `h1`, no raw HTML, a language on every code fence, no skipped heading level, and that the page is in `content/navigation.ts` |
| `bun run reference` | Regenerates `content/generated/` from `docs/openapi.json`, `resources/operations.json`, `docs/reference/cli.md`, `docs/reference/mcp.md`, `.env.example`, `docs/shortcuts.md` and `CHANGELOG.md` |
| `bun run reference:check` | Regenerates in memory and fails when `content/generated/` is stale, missing a page or has an extra one |
| `bun run test:reference` | Tests of the generators: one REST section per operation, one CLI section per command, one MCP section per tool, resource and prompt |
| `bun run last-updated` | Writes `content/last-updated.json` (the date and a hash of the source of each page) from git; never edit it by hand |
| `bun run check:last-updated` | Fails when a page changed after its date |
| `bun run test:site` | Bun tests: catalog parity, plans, SEO, sitemap, base path, terms gating, and planted failures for each check |
| `bun run typecheck` | `tsc --noEmit` over the TypeScript files |
| `bun run screenshots` | Regenerates `public/landing/*` from the app and its demo seed (needs Go, Bun and a Playwright Chromium from the root `node_modules`) |
| `bun run og` | Regenerates `public/og.png` (1200 by 630) |

## The documentation

- Pages are plain Markdown (GFM, no JSX and no raw HTML) with front matter (`title`, `description`): `site/content/docs/<section>/<slug>.md` are written by hand, `site/content/generated/**` is generated and starts with a banner. Both are listed in `site/content/navigation.ts`; a directory entry expands to every page under it.
- Links between pages are absolute `/docs/.../` paths with a trailing slash; the build adds the base path. Link outside only to the repository, and put any other URL in code: a bare URL becomes a link to another origin and fails `check:built`.
- The reference pages are generated, never edited. After a change to `docs/openapi.json`, `docs/reference/*.md`, `resources/operations.json`, `.env.example`, `docs/shortcuts.md` or `CHANGELOG.md`, run `bun run reference` and commit the pages with it. The generators read their sources from `site/lib/reference-config.ts`.
- The pages use no script: the sidebar is `details` elements, code blocks and wide tables are focusable named regions, and the colors follow the system theme.
- Search is [Pagefind](https://pagefind.app) 1.6.0, pinned exactly: `scripts/search.ts` indexes the pages that carry `data-pagefind-body` (the docs only) into `out/pagefind/` and drops the stock UI files. The search box is the site's own (`src/scripts/docs-search.ts`): a labelled input with results in a live region, hidden in the HTML and shown by the script, so the pages still need no JavaScript. Its only requests are same-origin GETs of the static files under `/pagefind/` (the index, the WebAssembly and Pagefind's worker); `check:built` allows `fetch` only inside that folder and requires the index when a page has the box.
- The documentation says the personal edition is 100% offline (D-027) and that the SaaS edition is software an operator hosts, with no hosted service offered.

## Configuration

- `SITE_URL` sets the public address. The default and the production value are `https://finances.vukem.dev` with the base path `/`; an address with a path (a preview) becomes the base path.
- `LANDING_TERMS_APPROVED=true` builds `/terms/` and shows its footer link. Until the author approves `docs/legal/terms.md`, the page does not exist.
- `SITE_OUT` overrides the output directory (the tests use it); the default is `out`.

## Where things are

- Copy: `site/src/landing/copy/{en,pt-br,es}.json`. The three must have the same keys and no empty value.
- Plans: `site/generated/plans.json` is what the editions block reads. It mirrors `Free` and `Pro` of `app/workspaces/limits.go`, has no price and is written by `go run ./cmd/app plans:export`; `plans:export -check` (run by the `go` job of `ci.yml`, and `TestPlansExportIsFresh`) fails when the file is stale.
- Legal drafts: `docs/legal/privacy.md` and `docs/legal/terms.md` (pending review by the author).
- Design tokens: `site/src/landing/landing.css`, the pairs of `docs/design.md` as `light-dark()`.

## Publishing

`.github/workflows/site.yml` builds and checks on pull requests that touch the site or the sources of the generated pages and deploys to GitHub Pages from `main` with the official actions pinned by SHA. These steps are manual and belong to the author, because the host runs the old app today (D-013):

1. In the repository settings, set Pages source to "GitHub Actions".
2. Set the custom domain `finances.vukem.dev` in Pages and enable "Enforce HTTPS". `site/public/CNAME` records the intended host.
3. Change the DNS record of `finances.vukem.dev` from the old app to GitHub Pages. This replaces the old app on that host.
4. Verify HTTPS and `https://finances.vukem.dev/sitemap.xml`.
5. Approve the terms text and set the repository variable `LANDING_TERMS_APPROVED` to `true`.

# The static site

`site/` is a self-contained static [Astro](https://astro.build) project (its own `package.json` and `bun.lock`, not a workspace of the root) that holds the commercial landing page, at `/`, `/pt-br/` and `/es/`, and the privacy notice. The documentation (epic 23) will live at `/docs`. The landing page and its claims are described in [landing-brief.md](landing-brief.md).

The site is 100% static (decisions D-019 and D-020): no backend, login, sign-up, checkout, form or call to an API, no cookie and no third-party script, font, image or stylesheet. Every call to action is a plain link to the docs, the GitHub repository, the Docker install page of the docs or the GitHub Releases page. The page says that no hosted service exists.

## Commands

Run them in `site/` after `bun install --frozen-lockfile`.

| Command | What it does |
| --- | --- |
| `bun run dev` | Astro dev server |
| `bun run build` | Builds `site/out` and writes `sitemap.xml` and `robots.txt` next to the pages |
| `bun run check:built` | Scans `site/out`: no form, other origin, network API, server or worker file; one `h1`, landmarks, skip link, alt text, title and description lengths, JSON-LD without a price, hreflang, no hosted-service phrase, weight budgets |
| `bun run check:claims` | Checks the claims register of `docs/landing-brief.md` (every evidence path exists) and the forbidden phrases of the three catalogs |
| `bun run test:site` | Bun tests: catalog parity, plans, SEO, sitemap, base path, terms gating, and planted failures for each check |
| `bun run typecheck` | `tsc --noEmit` over the TypeScript files |
| `bun run screenshots` | Regenerates `public/landing/*` from the app and its demo seed (needs Go, Bun and a Playwright Chromium from the root `node_modules`) |
| `bun run og` | Regenerates `public/og.png` (1200 by 630) |

## Configuration

- `SITE_URL` sets the public address. The default and the production value are `https://finances.vukem.dev` with the base path `/`; an address with a path (a preview) becomes the base path.
- `LANDING_TERMS_APPROVED=true` builds `/terms/` and shows its footer link. Until the author approves `docs/legal/terms.md`, the page does not exist.
- `SITE_OUT` overrides the output directory (the tests use it); the default is `out`.

## Where things are

- Copy: `site/src/landing/copy/{en,pt-br,es}.json`. The three must have the same keys and no empty value.
- Plans: `site/generated/plans.json` is what the editions block reads. It mirrors `Free` and `Pro` of `app/workspaces/limits.go` and a test compares them; it has no price. The export command `plans:export` of the plan is not implemented yet because it changes the Go app (see the hand-off).
- Legal drafts: `docs/legal/privacy.md` and `docs/legal/terms.md` (pending review by the author).
- Design tokens: `site/src/landing/landing.css`, the pairs of `docs/design.md` as `light-dark()`.

## Publishing

`.github/workflows/site.yml` builds and checks on pull requests that touch the site and deploys to GitHub Pages from `main` with the official actions pinned by SHA. These steps are manual and belong to the author, because the host runs the old app today (D-013):

1. In the repository settings, set Pages source to "GitHub Actions".
2. Set the custom domain `finances.vukem.dev` in Pages and enable "Enforce HTTPS". `site/public/CNAME` records the intended host.
3. Change the DNS record of `finances.vukem.dev` from the old app to GitHub Pages. This replaces the old app on that host.
4. Verify HTTPS and `https://finances.vukem.dev/sitemap.xml`.
5. Approve the terms text and set the repository variable `LANDING_TERMS_APPROVED` to `true`.

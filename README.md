<div align="center">

<h1><a href="https://github.com/AlexGalhardo/vukem-finances">Vukem Finances</a></h1>

<a href="https://github.com/AlexGalhardo/vukem-finances/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/AlexGalhardo/vukem-finances/ci.yml?branch=main&logo=github&label=CI" alt="CI"></a>
<a href="https://github.com/AlexGalhardo/vukem-finances/actions/workflows/image.yml"><img src="https://img.shields.io/github/actions/workflow/status/AlexGalhardo/vukem-finances/image.yml?logo=docker&logoColor=white&label=image" alt="Image"></a>
<a href="https://github.com/AlexGalhardo/vukem-finances/releases"><img src="https://img.shields.io/github/v/release/AlexGalhardo/vukem-finances?include_prereleases&label=version" alt="Version"></a>
<a href="go.mod"><img src="https://img.shields.io/badge/go-1.27.4-00ADD8?logo=go&logoColor=white" alt="Go 1.27.4"></a>
<a href="LICENSE"><img src="https://img.shields.io/github/license/AlexGalhardo/vukem-finances" alt="License"></a>

### Know where your money goes, on your own machine or as a service for your customers

<br/>

</div>

<br/>

## Introduction

**Your financial life should not depend on someone else's cloud, and running a finance product should not mean starting from zero. This is why Vukem Finances exists.**

Vukem Finances is an open source (MIT) personal finance application built with [Vukem](https://vukem.dev): Go on the server, React with Inertia in the browser and SQLite in a single file. It is built around the problems that get in the way of tracking money:

- **Your money is scattered across accounts, cards and spreadsheets.** Keep it in one place:
  - Accounts, categories, transactions and monthly budgets.
  - A [dashboard](docs/guide.md#how-it-fits-together) with charts, and CSV import for what you already have.
- **A finance app holds your most private data.** Choose who runs it:
  - The **personal** edition runs on your computer or your server, with closed sign-up and no plan limits.
  - The **SaaS** edition serves many customers, with public sign-up, email verification and [Stripe plans](docs/guide.md#plans).
  - Both are one binary, chosen by [`APP_EDITION`](docs/guide.md#editions).
- **Families and teams need to share without sharing everything.** Every workspace has its own data and roles:
  - Owner, admin, member and read-only viewer, enforced on the server.
  - Invitations, ownership transfer, a timezone per workspace and an audit log of sensitive actions.
- **Accounts get attacked.** Security is part of the default setup:
  - Optional two-factor authentication, with a limit on wrong codes per account.
  - Rate limits on workspace routes, and tests that refuse access across workspaces.
  - Money stored as integer cents, so no amount is ever a float.
- **Clicking through menus is slow.** Work from the keyboard:
  - A left sidebar, a command palette and [keyboard shortcuts](docs/shortcuts.md).
  - Light and dark themes, built on an [original design system](docs/design.md).
- **Self-hosting is often the hard part.** Start with one command:
  - `docker compose up -d --build --wait` runs it on your computer.
  - A [VPS stack](deploy/README.md) with Caddy and automatic HTTPS, plus backup and restore.

New here? Start with the [guide](docs/guide.md) and the [deployment guide](deploy/README.md).

## Stability

Versions follow SemVer. Vukem Finances is in `0.x`: a minor release may break. See the [changelog](CHANGELOG.md) for what changed in each release.

## Contributing

To build from source you need Go (see `go.mod`) and Bun. Run `bun install`, `go test ./...`, `bun run typecheck` and `bun run lint:js`. Read [AGENTS.md](AGENTS.md) for the repository rules. Commits follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Security vulnerabilities

If you discover a security vulnerability, please contact the maintainer through their [GitHub profile](https://github.com/AlexGalhardo) instead of opening a public issue.

## License

Vukem Finances is open-source software licensed under the [MIT license](LICENSE).

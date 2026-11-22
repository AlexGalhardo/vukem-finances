# Security policy

## Reporting a vulnerability

Please report a vulnerability privately, not in a public issue or pull request.

Use GitHub private vulnerability reporting: open the Security tab of
[AlexGalhardo/vukem-finances](https://github.com/AlexGalhardo/vukem-finances/security) and choose "Report a vulnerability".

Include what you found, the steps to reproduce it, the version (`about` command or `config.Version`), the edition
(`personal` or `saas`) and what an attacker gains. Do not include real financial data, and do not test against an
installation you do not own.

## What to expect

- An acknowledgement within 5 working days.
- A first assessment, with a severity (Critical, High, Medium or Low), within 14 days.
- A fix in the next release for High and Critical findings, and a note in the release notes under Security once the fix is
  out. We credit the reporter unless asked not to.

This is a project run by one maintainer: these are goals and not a contract. There is no bug bounty.

## Supported versions

Only the latest release receives security fixes.

## Scope

In scope: the application in this repository (server, web app, public API, MCP server, sync protocol, Telegram bot,
webhooks, desktop app, mobile app, Docker images and the deployment files).

Out of scope: third-party services (Stripe, AbacatePay, Telegram, AI providers), a host you configured weakly, social
engineering, denial of service by volume, and findings that need physical access to an unlocked device.

The threat model, with the controls and the tests that prove them, is in [docs/security.md](docs/security.md).

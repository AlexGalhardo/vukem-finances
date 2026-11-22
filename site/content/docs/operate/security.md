---
title: "Security notes"
description: "How vukem-finances protects your data, the rules that never bend, and how to report a vulnerability."
---

## What protects your data

- **Isolation.** Every domain table carries a workspace id and every query goes through a workspace scope. Isolation is tested, not trusted: a test runs every operation in another workspace with the first one's ids and expects a 404.
- **Roles on the server.** The roles `owner`, `admin`, `member` and `viewer` are enforced where the work happens. Hiding a button is never the check. See [roles and workspaces](/docs/operate/roles-workspaces/).
- **Tokens.** An API token is shown once, only a hash is stored, and it is bound to one workspace and to the person who made it. A scope is a ceiling that the role of the person cuts at every request. Removing a member revokes their tokens.
- **Money.** Amounts are integers in cents and are never converted to floating point numbers.
- **Secrets.** `APP_KEY` encrypts stored secrets such as an AI provider key. Provider error text, tokens and keys never reach a log, a response or an export.
- **Untrusted files.** Statement import bounds the size, rows and cells, refuses entity expansion in OFX, and stores amounts only through the exact parser. Exports guard text cells against spreadsheet formulas.
- **The browser.** The Content-Security-Policy is `default-src 'self'`. In the personal edition no third-party script, font or image is ever loaded.
- **The desktop app.** It listens on the loopback address behind a gate that accepts only requests from its own window.
- **AI and MCP.** Text from your data is passed to a model as data, never as an instruction, and a result never carries a secret. A destructive MCP tool asks you to confirm every call.

## Running it safely

- Keep `.env` out of git and readable only by you, and never publish `APP_KEY`.
- On a server, publish only what `compose.prod.yaml` publishes, and run one replica; see [a VPS with Caddy](/docs/run/vps/).
- Back up the database with `db:backup` and keep a copy off the server; see [backup and restore](/docs/run/backup-restore/).
- Give an API token or an MCP toolset the smallest scope that does the job.
- Keep the application up to date; see [upgrading](/docs/run/upgrading/).

## Report a vulnerability

Please do not open a public issue with the details of a vulnerability. Use the private vulnerability reporting of the repository, under the Security tab of [the repository](https://github.com/AlexGalhardo/vukem-finances), when it is available. If it is not, open an issue that says only that you have a security report and asks for a private way to send it, without any technical detail. Include the version, what you found, how to reproduce it and what you think the impact is. A report is handled before any public discussion, and a fix ships in a release with a note in the changelog.

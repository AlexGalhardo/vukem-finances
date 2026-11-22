---
title: "Documentation"
description: "Guides and references for running, using and integrating vukem-finances, an open-source personal finance app you run yourself."
---

vukem-finances is an open-source personal finance application: accounts, transactions, budgets, goals, reports and CSV import, with data that stays on your own computer or server. These pages explain how to run it, how to use it and how to connect other programs to it.

The documentation describes the latest release and is written from the code: the REST, CLI and MCP references are generated from the same sources as the product, and a check fails the site build when a page names a command, setting, route or tool that does not exist.

## Where to start

- New here: read [what vukem-finances is](/docs/start/what-is-it/), then [the two editions](/docs/start/editions/) and the [quick start](/docs/start/quick-start/).
- Run it on your computer: the [desktop app](/docs/run/desktop/) or [Docker on localhost](/docs/run/docker/). Both work with no internet.
- Run it for a household or a team: [a VPS with Caddy](/docs/run/vps/), then [backup and restore](/docs/run/backup-restore/).
- Use it every day: the [Use it guides](/docs/use/accounts-transfers/) and the [keyboard shortcuts](/docs/use/shortcuts/).
- Connect a script or an AI client: the [REST API](/docs/integrate/rest-api/), the [command line](/docs/integrate/cli/) and [MCP over stdio](/docs/integrate/mcp-stdio/).
- Change the code: [how it is built](/docs/contribute/architecture/) and [how to run the tests](/docs/contribute/tests/).

## The three ways in

The REST API, the command line and the MCP server are three adapters over one registry of operations. A command, an endpoint and a tool of the same name take the same input, check the same permissions and give the same answer. Each has a reference generated from that registry.

## Offline by design

The personal edition, on the desktop and in Docker on your own computer, is a standalone application: it needs no internet to run. Only the SaaS edition needs the network, and no hosted service is currently offered; see [offline use](/docs/run/offline/).

## Every page also as Markdown

Each page has a Markdown twin at the same address with `.md`, and `llms.txt` at the root of the site lists them all, so tools and agents can read the documentation without scraping HTML.

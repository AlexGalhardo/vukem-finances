---
title: "Roles and workspaces"
description: "Workspaces hold your data, four roles decide who can do what, and the editions differ in how workspaces are created and joined."
---

## Workspaces

A workspace is a set of accounts, transactions and settings shared by its members. Everything in the app belongs to exactly one workspace, and nothing crosses between them: a request that names another workspace's row gets the same 404 as a row that does not exist.

- In the personal edition there is one implicit workspace, created at `/setup`, and people join it through an invite link.
- In the SaaS edition each person has a workspace and can create more, switch between them, and invite others.

Each workspace has a currency and a time zone. "Today" and "this month" follow its time zone.

## Roles

A workspace has exactly one owner. Only an ownership transfer moves it.

| Role | Can |
| --- | --- |
| owner | everything an admin can, plus billing in the SaaS edition, AI settings and transferring ownership |
| admin | manage accounts, categories, tags, budgets, cards, goals and imports, invite and remove people, rename the workspace, and take the JSON export |
| member | record and edit transactions, transfers, recurrences and statement payments |
| viewer | read everything and download the CSV; never write |

The role is read on the server at every request. A person demoted to viewer loses writing at once, and removing a member revokes their API tokens, sync devices and MCP grants for that workspace.

## Tokens and roles

An API token, a Telegram chat and an MCP grant each hold a set of scopes, and each is cut to the role of the person behind it at every call, so they can never do more than the person could. See [the REST API scopes](/docs/integrate/rest-api/).

## Audit

Sensitive actions on a workspace, such as changing roles, ownership, members, invitations and settings, are written to an audit log. Operations that write or destroy data through the API, the command line or MCP write a log line with the name, kind, surface, workspace, user and outcome, and never an amount, a description or a name.

## Locked out

If the owner cannot sign in, see the recovery command in [a VPS with Caddy](/docs/run/vps/).

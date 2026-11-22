---
title: "MCP over stdio"
description: "Connect an AI client to your own finances on your own machine with the local MCP server: setup, toolsets, destructive tools and safety."
---

A person can connect an AI client to their own finances on their own machine, and ask it to record, find and analyze transactions. There is no account, no network service and no model in the server: the client brings the model. The server speaks the Model Context Protocol, revision `2026-07-28`, and works offline. The tools, resources and prompts are listed in the generated [tools reference](/docs/integrate/mcp/tools/) and [resources and prompts](/docs/integrate/mcp/resources-prompts/).

## Run it

```sh
APP_EDITION=personal go run ./cmd/app mcp:serve --stdio
```

- It runs only with `APP_EDITION=personal`, as the local owner of the installation. With `saas` it exits non-zero before it serves, and without `--stdio` it is a usage error. For the SaaS edition see [MCP remote with OAuth](/docs/integrate/mcp-remote/).
- Standard output carries protocol messages and nothing else. Logs go to standard error.
- The owner must exist: finish `/setup` first.

## Configure a client

A client configuration looks like this. The path of the binary and of the database are yours:

```json
{
  "mcpServers": {
    "finances": {
      "command": "/path/to/app",
      "args": ["mcp:serve", "--stdio"],
      "env": {
        "APP_EDITION": "personal",
        "DATABASE_URL": "file:/path/to/storage/database.sqlite",
        "APP_KEY": "<the APP_KEY of the installation>",
        "MCP_TOOLSETS": "read"
      }
    }
  }
}
```

## What the agent may do

| Setting | Meaning |
| --- | --- |
| `MCP_TOOLSETS` | comma separated groups of tools: `read` (the default), `transactions`, `planning` and `all` |
| `MCP_ALLOW_DESTRUCTIVE` | `true` offers the tools that remove or archive data; the default is `false` |

- **Read only by default.** Write access is a choice. The default actor does not even hold a write scope.
- **Context budget.** A full tool list is large, and the sizes of each toolset are in the [tools reference](/docs/integrate/mcp/tools/). Pick the smallest toolset that does the job.
- **Destructive tools** are off unless `MCP_ALLOW_DESTRUCTIVE=true`. When on, each call asks you to confirm through the protocol, showing the tool and its input. A client that cannot ask gets a refusal, and a declined answer changes nothing. The annotations are hints for the client; the server enforces.
- **Downloads** have no tool: a file does not fit a model context. The read tools cover the same data.

## Safety

- Every tool runs through the operation registry, so edition, scope, role, validation and audit are the registry's.
- A result is your own text, such as descriptions, names and imported notes, and it may carry a prompt injection. The server says so in its instructions and in every tool description, returns the text as data, and no tool acts on what it finds there.
- A result never holds a password hash, an API token or the application key. An internal failure is a generic message and its cause stays in the log.
- Resources are read only and pass through the same operations as the tools.

## Resources, prompts and completion

Resources are `finances://workspace`, `finances://accounts`, `finances://categories`, `finances://budgets/{month}` and `finances://reports/{kind}/{period}`. Prompts are `monthly-review`, `categorize-uncategorized`, `budget-plan`, `find-unusual-spending` and `import-statement`. Completion suggests account and category names, periods and months, at most 100 values.

# MCP server

A person can connect an AI client to their own finances, on their own machine, and ask it to record, find and
analyze transactions. There is no account, no network service and no model in the server: the client brings the
model. The tools, resources and prompts are listed in the generated [reference](reference/mcp.md).

The target is the MCP specification revision `2026-07-28` (stateless, no session). Older revisions are answered
per request by the Vukem `mcp` package. The code is `app/mcp/`; the protocol is the framework's.

## Run it on stdio (personal edition)

```sh
APP_EDITION=personal go run ./cmd/app mcp:serve --stdio
```

- It runs only with `APP_EDITION=personal`, as the local owner of the installation. With `saas` it exits non-zero
  before it serves, and with `--stdio` missing it is a usage error.
- Standard output carries protocol messages and nothing else. Logs go to standard error.
- The owner must exist: finish `/setup` first.

A client configuration (the path of the binary and the database are yours):

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
| `MCP_TOOLSETS` | Comma separated groups: `read` (the default), `transactions`, `planning`, `all`. |
| `MCP_ALLOW_DESTRUCTIVE` | `true` offers the tools that remove or archive data. Default `false`. |

- **Read only by default.** Write access is a choice. The default actor does not even hold a write scope.
- **Context budget.** A full tool list is large (see the sizes in the reference). Pick the smallest toolset;
  `TestToolsets` fails when a toolset outgrows its budget.
- **Destructive tools** are off unless `MCP_ALLOW_DESTRUCTIVE=true`. When on, each call asks the person to
  confirm through the protocol (the input-required flow), showing the tool and its input. A client that cannot ask
  gets a refusal and a declined answer changes nothing. Annotations are hints for the client; the server enforces.
- **Downloads** (`export.*`) have no tool: a file does not fit a model context. The reads cover the same data.

## Safety

- Every tool runs through `operations.Invoke`: edition, scope, role, validation and audit are the registry's.
- A result is the person's own text (descriptions, names, imported notes) and may carry a prompt injection. The
  server says so in its instructions and in every tool description, returns it as data fields, and no tool acts on
  what it finds there. A prompt puts a typed value in one quoted string.
- A result never holds a password hash, an API token or the app key. An internal failure is a generic message; its
  cause stays in the server log.
- Resources are read only and read through the same operations, so they have the scope and isolation of the tool.

## Resources, prompts and completion

Resources: `finances://workspace`, `finances://accounts`, `finances://categories`, `finances://budgets/{month}` and
`finances://reports/{kind}/{period}` (a preset such as `this_month`, or `2026-01-01_2026-03-31`). Prompts:
`monthly-review`, `categorize-uncategorized`, `budget-plan`, `find-unusual-spending`, `import-statement`.
Completion suggests account and category names, periods and months, at most 100 values. Lists are paged with
opaque cursors, reports send progress when the client gives a progress token, and cancelling a request cancels the
context of the operation.

## Check it with the MCP Inspector

```sh
npx @modelcontextprotocol/inspector -- go run ./cmd/app mcp:serve --stdio
```

List the tools, resources and prompts and run one of each. The Inspector is a manual check and is not part of CI.

---
title: "MCP remote with OAuth"
description: "The authenticated MCP endpoint of the SaaS edition: how a client connects with OAuth, which scopes it gets, how consent works and how workspaces stay isolated."
---

The remote MCP endpoint is part of the SaaS edition, which an operator runs on their own server; no hosted service is currently offered. A person connects an AI client to one workspace by signing in and approving what the client may do. The personal edition has none of this: `/mcp`, the metadata and the authorization endpoints answer 404, and the local server is [MCP over stdio](/docs/integrate/mcp-stdio/).

## Turn it on

The endpoints are registered only when the edition is `saas` and `APP_URL` is an `https` address; otherwise remote MCP stays off with a warning. `MCP_TOOLSETS` and `MCP_ALLOW_DESTRUCTIVE` apply to the remote endpoint too. `MCP_OAUTH_CLIENTS` is a JSON list of public clients known ahead of time, each with an `id`, a `name` and `redirect_uris`:

```json
[{"id":"my-client","name":"My client","redirect_uris":["https://client.example/callback"]}]
```

Any other client is identified by a Client ID Metadata Document.

## Endpoint

`POST /mcp` only; `GET` and `DELETE` answer 405. There is no session id and no state between requests. The `Origin` is checked (403), and the `MCP-Protocol-Version`, `Mcp-Method` and `Mcp-Name` headers must agree with the body (400).

## The authorization flow

1. The client calls `/mcp` without a token and gets a 401 whose `WWW-Authenticate` header carries the address of the protected resource metadata.
2. The client reads `/.well-known/oauth-protected-resource` (RFC 9728) and then `/.well-known/oauth-authorization-server` (RFC 8414).
3. The client starts an authorization code flow with PKCE `S256` only, an exact redirect match and the `resource` parameter (RFC 8707). The response carries `iss` (RFC 9207).
4. The person signs in and sees a consent screen: the client, where it returns to, the workspace and the scopes. They can change the workspace there. Scopes the role cannot grant are marked, and the server refuses a decision that asks for them.
5. The client exchanges the code at `/token`. Access tokens live 15 minutes and name this server as their only audience. Refresh tokens rotate and are stored hashed; using one twice revokes the grant.

A client identified by a metadata document is shown by its verified host, never by the name it claims; that name is shown apart as unverified. The document is fetched only over HTTPS, from public addresses, with no redirects, in 5 seconds and 5 KiB.

## Scopes and roles

The scopes are the same as the API's; see [the REST API scopes](/docs/integrate/rest-api/). The approved scopes are stored with the grant, and on every call they are cut to the role the person holds now. A promotion never widens a grant, and a downgrade narrows it at once. Tools that remove data are for owners and admins and still ask for confirmation.

## Workspace isolation

A grant is bound to one person and one workspace. The membership and the role are read again on every request through the operation registry. Removing a member, resetting a password or signing out everywhere ends the grants. Resources, completion and tool calls all pass the same checks, so a client never sees another workspace.

## Limits and audit

120 reads and 60 writes a minute per person, workspace and client; 300 requests a minute per address on `/mcp`; 30 a minute per address on each OAuth endpoint. Every call writes a log line with the client, the tool and the outcome, never an input, an amount or a description.

## Connected apps

The page `/connected-apps` lists a person's grants with the client, scopes and last use, and revokes them: the next call gets 401 and the refresh token stops.

## Behind a proxy

The `deploy/Caddyfile` routes `/mcp` with `flush_interval -1` and no compression, so streamed progress arrives as it happens.

## Check it

Run the MCP Inspector against `https://<your domain>/mcp` and use its authentication panel. The tools, resources and prompts are the ones in the [reference](/docs/integrate/mcp/tools/).

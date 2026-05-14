# MCPBox Developer Guide

English documentation. Russian version: [DEVELOPER-ru.md](./DEVELOPER-ru.md)

## Overview

MCPBox is a Go-based control plane for MCP servers.

It groups MCP servers by project, stores configuration in SQLite, launches local `STDIO` MCP servers, proxies remote `HTTP streaming` MCP servers, provides an embedded React admin UI, and exposes one MCP URL per project.

At the current stage MCPBox is:
- a control plane for MCP servers;
- a project-based organizer for local and remote MCP endpoints;
- a single MCP endpoint per project;
- an operator UI for project lifecycle, server lifecycle, logging, and inspection.

At the current stage MCPBox is not:
- a multi-user platform with authentication and roles;
- a full observability stack;
- an aggregated multi-server MCP router that merges multiple backends behind one intelligent session.

## Core Model

- `Project`: a logical workspace group of MCP servers for one client, team, or environment.
- `MCP Server`: either a local `STDIO` server launched by MCPBox or a remote `HTTP streaming` server.
- `Primary Server`: the server inside a project that backs the project's MCP endpoint.

Important behavior:
- each project has exactly one MCP URL;
- that MCP URL always uses the explicitly selected primary server;
- if no primary server is selected, the MCP endpoint is not ready;
- if a project is paused, MCP connections are blocked;
- if a server is disabled, it cannot be started or used as the primary server.

## Current Features

### Backend

- Go HTTP API
- SQLite storage through GORM
- local `STDIO` MCP process orchestration
- remote `HTTP streaming` MCP proxy support
- project-level `/mcp/{project_token}` endpoint
- synchronous JSON-RPC request/response bridge for HTTP MCP clients
- legacy SSE compatibility mode for older MCP clients
- explicit primary server selection
- audit logging for control actions and MCP traffic
- project pause/resume
- server enable/disable
- `STDIO` server inspection

### Frontend

- embedded React admin UI
- project list and project overview
- modal create-project flow
- modal add-server flow for `STDIO` and `HTTP streaming`
- project MCP URL display
- start/stop controls for local servers
- primary server selection
- audit log console with project filtering
- auto-refreshing log view
- `Info` modal for `STDIO` servers
- English and Russian localization

## Transport Model

The main project endpoint is:

```http
GET /mcp/{project_token}
POST /mcp/{project_token}
```

There is also a backward-compatible alias:

```http
GET /connect/{project_token}
POST /connect/{project_token}
```

Current transport behavior:
- main mode: `POST /mcp/{project_token}` without `sessionId` works as synchronous HTTP JSON-RPC;
- legacy mode: `GET /mcp/{project_token}` opens an SSE stream and returns an `endpoint` event;
- legacy follow-up requests: `POST /mcp/{project_token}?sessionId=...`;
- remote `HTTP streaming` primary servers are proxied to upstream;
- paused projects and disabled primary servers are blocked before transport is opened.

Why this matters:
- LM Studio expects synchronous HTTP request/response for calls like `initialize` and `tools/list`;
- older SSE-based MCP clients still need the legacy session flow;
- MCPBox supports both without requiring separate project URLs.

## `STDIO` Server Inspection

For local `STDIO` servers MCPBox can inspect live MCP capabilities.

The `Info` action in the UI is available only for `STDIO` servers. It can show:
- server metadata from `initialize`;
- negotiated MCP capabilities;
- exposed `tools`;
- exposed `resources`;
- exposed `prompts`;
- nearby `README.md` if MCPBox finds one next to the configured local path.

Remote `HTTP streaming` servers intentionally do not expose this UI action.

## Logging and Control

MCPBox includes an audit trail intended for operational control.

Currently it logs events such as:
- project creation;
- server creation;
- primary server changes;
- project pause/resume;
- server start/stop;
- server enable/disable;
- MCP connection attempts;
- forwarded JSON-RPC payloads.

SQL query logging from GORM is disabled by default so normal MCP traffic does not flood the console.

## Requirements

- Go `1.25+`
- Node.js and npm for building the embedded UI
- Windows, Linux, or macOS

No external database is required. MCPBox creates a local SQLite file by default.

## Project Structure

```text
main.go                      Application entry point
internal/models              GORM models
internal/storage             SQLite storage and queries
internal/orchestrator        MCP process lifecycle, inspection, stdio bridge
internal/httpapi             HTTP API, MCP endpoint, embedded UI
html                         React + Vite source for the embedded admin UI
```

## Build

Build the embedded UI first, then build the Go binary:

```bash
npm --prefix html install
npm --prefix html run build
go build -o MCPBox .
```

On Windows:

```powershell
npm --prefix html install
npm --prefix html run build
go build -o MCPBox.exe .
```

The frontend build output is written to `internal/httpapi/ui/dist` and embedded into the Go application.

## Run

Default port: `38180`

Run from source:

```bash
go run .
```

On Windows:

```powershell
.\MCPBox.exe
```

When MCPBox starts successfully, it opens the local UI in the browser automatically and prints:
- `http://127.0.0.1:<port>/`
- local IPv4 URLs such as `http://192.168.x.x:<port>/`

## Port Configuration

Priority order:
1. CLI flag `-port`
2. Environment variable `MCPBOX_PORT`
3. Default `38180`

Examples:

```bash
go run . -port 39000
```

```powershell
$env:MCPBOX_PORT=39000
.\MCPBox.exe
```

## Local Data

By default MCPBox creates:

```text
mcpbox.db
```

This file is local runtime data and should not be committed.

## Client Integration

Typical flow:

1. Start MCPBox.
2. Create a project in the UI.
3. Add at least one MCP server to the project.
4. Select a primary server.
5. Copy the project's MCP URL from the UI.

Example MCP URL:

```text
http://127.0.0.1:38180/mcp/<project_token>
```

Important:
- the project must have a primary server before the endpoint is ready;
- if the project is paused, client access is blocked;
- if the primary server is disabled, the MCP path will fail;
- `/mcp/{project_token}` is the main endpoint;
- `/connect/{project_token}` is a backward-compatible alias only.

### Codex

CLI example:

```bash
codex mcp add mcpbox --url http://127.0.0.1:38180/mcp/<project_token>
```

Direct config example:

```toml
[mcp_servers.mcpbox]
url = "http://127.0.0.1:38180/mcp/<project_token>"
```

### Claude Code

CLI example:

```bash
claude mcp add --transport sse mcpbox http://127.0.0.1:38180/mcp/<project_token>
```

Project config example:

```json
{
  "mcpServers": {
    "mcpbox": {
      "type": "sse",
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

### LM Studio

LM Studio works with MCPBox through the main HTTP MCP URL.

Example config:

```json
{
  "mcpServers": {
    "mcpbox": {
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

Important notes for LM Studio:
- no `Authorization` header is required unless MCPBox is placed behind a separate auth layer;
- `POST /mcp/{project_token}` returns the actual JSON-RPC result synchronously;
- this is what allows `initialize`, `tools/list`, and similar calls to work correctly.

### Generic MCP Clients

For clients that accept JSON config, the typical pattern is:

```json
{
  "mcpServers": {
    "mcpbox": {
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

## HTTP API

### Health

```http
GET /healthz
```

### Projects

Create project:

```http
POST /api/projects
Content-Type: application/json
```

```json
{
  "name": "My Project",
  "description": "My MCP environment"
}
```

List projects:

```http
GET /api/projects
```

Get project status:

```http
GET /api/projects/{id}/status
```

Set primary server:

```http
POST /api/projects/{id}/primary-server
Content-Type: application/json
```

```json
{
  "server_id": 2
}
```

Pause project:

```http
POST /api/projects/{id}/pause
```

Resume project:

```http
POST /api/projects/{id}/resume
```

### MCP Servers

Add `STDIO` server:

```http
POST /api/projects/{id}/servers
Content-Type: application/json
```

```json
{
  "name": "Filesystem Server",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"],
  "env_vars": [],
  "env_passthrough": ["OPENAI_API_KEY"],
  "working_dir": "",
  "auto_start": true
}
```

Add remote `HTTP streaming` server:

```json
{
  "name": "Remote MCP",
  "transport": "http_stream",
  "url": "https://mcp.example.com/mcp",
  "bearer_token_env_var": "MCP_BEARER_TOKEN",
  "headers": [],
  "header_env_vars": []
}
```

Start server:

```http
POST /api/servers/{id}/start
```

Stop server:

```http
POST /api/servers/{id}/stop
```

Disable server:

```http
POST /api/servers/{id}/disable
```

Enable server:

```http
POST /api/servers/{id}/enable
```

Inspect local `STDIO` server:

```http
GET /api/servers/{id}/inspect
```

### Logs

List audit logs:

```http
GET /api/logs
```

Filter logs by project:

```http
GET /api/logs?project_id={id}
```

## Known Limitations

At the current stage MCPBox does not yet provide:
- authentication and authorization;
- user and role management;
- automatic multi-server routing inside one project;
- advanced metrics dashboards;
- historical analytics beyond the stored audit log;
- service installers for OS-level background execution.

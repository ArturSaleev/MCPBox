# MCPBox

English documentation. Russian version: [README-ru.md](./README-ru.md)

MCPBox is a Go-based control center for managing MCP servers in one place.

It groups servers by project, stores configuration in SQLite, launches local `STDIO` MCP servers, proxies per-project MCP connections, provides an embedded web UI, keeps an audit log of actions and requests, and lets operators pause projects or disable individual servers when needed.

## What MCPBox Is

MCPBox is not just a raw MCP proxy.

At the current stage it is:
- a control plane for MCP servers;
- a project-based organizer for local and remote MCP endpoints;
- a single connect endpoint per project;
- an operator UI for project lifecycle, server lifecycle, logging, and inspection.

At the current stage it is not:
- a multi-user platform with authentication and roles;
- a full observability stack;
- an aggregated multi-server MCP router that merges multiple backends behind one intelligent session.

## Core Model

The main domain objects are:

- `Project`: a logical workspace group of MCP servers for one client, team, or environment.
- `MCP Server`: either a local `STDIO` server launched by MCPBox or a remote `HTTP streaming` server.
- `Primary Server`: the one server inside a project that backs the project's `/connect/{token}` endpoint.

Important behavior:
- each project has exactly one connect URL;
- that connect URL always uses the explicitly selected primary server;
- if no primary server is selected, the connect endpoint is not ready;
- if a project is paused, MCP connections are blocked;
- if a server is disabled, it cannot be started or used as the primary server.

## Current Features

### Backend

- Go HTTP API
- SQLite storage through GORM
- local `STDIO` MCP process orchestration
- remote `HTTP streaming` MCP proxy support
- project-level `/connect/{project_token}` endpoint
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
- project connect URL display
- start/stop controls for local servers
- primary server selection
- audit log console with project filtering
- activity summary for most active project/server
- `Info` modal for `STDIO` servers
- English and Russian localization
- theme that follows system light/dark preference

## `STDIO` Server Inspection

For local `STDIO` servers MCPBox can inspect live MCP capabilities.

The `Info` action in the UI is available only for `STDIO` servers. It opens a modal that can show:
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
- connect attempts;
- forwarded JSON-RPC payloads.

This is designed to help operators understand who is using which MCP surface and to react quickly if a project or server should be stopped.

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
internal/httpapi             HTTP API, connect endpoint, embedded UI
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

Run a built binary:

```bash
./MCPBox
```

On Windows:

```powershell
.\MCPBox.exe
```

When MCPBox starts successfully, it opens the local UI in the browser automatically.

## Port Configuration

Priority order:
1. CLI flag `-port`
2. Environment variable `MCPBOX_PORT`
3. Default `38180`

Examples:

```bash
go run . -port 39000
```

```bash
MCPBOX_PORT=39000 go run .
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

## UI Overview

The embedded UI is organized around two primary views:

- `Projects`: create projects, add servers, choose a primary server, start/stop local servers, pause a project, disable a server, inspect local `STDIO` servers.
- `Logs`: compact audit console, current-project filter, and activity summary for most active projects and servers.

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

## Connect Endpoint

Each project has its own token and connect URL:

```http
GET /connect/{project_token}
POST /connect/{project_token}
```

Behavior:
- the endpoint always routes through the project's selected primary server;
- for local `STDIO` servers, `POST` forwards JSON-RPC into process `stdin`;
- for local `STDIO` servers, `GET` streams process `stdout` frames over SSE;
- for remote `HTTP streaming` servers, MCPBox proxies the request to the remote upstream;
- if the project is paused, access is blocked;
- if the primary server is missing or disabled, the connect path is not ready.

Example JSON-RPC request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

## Development Workflow

Frontend only:

```bash
cd html
npm install
npm run dev
```

Full verification:

```bash
npm --prefix html run build
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go build ./...
```

## Known Limitations

At the current stage MCPBox does not yet provide:
- authentication and authorization;
- user and role management;
- automatic multi-server routing inside one project;
- advanced metrics dashboards;
- historical analytics beyond the stored audit log;
- service installers for OS-level background execution.

## What Is Probably Enough For Now

For the current phase, the project already covers the core MCPBox value proposition:
- configure projects;
- connect local and remote MCP servers;
- select a primary server;
- inspect local `STDIO` capabilities;
- monitor usage through logs;
- block risky behavior by pausing a project or disabling a server.

That is a solid MVP for a company-operated MCP control center.

The next steps would be improvements, not missing fundamentals.

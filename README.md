# MCPBox

English documentation. Russian version: [README-ru.md](./README-ru.md)

MCPBox is a single-binary Go gateway for managing local MCP servers by project.

At the current stage, MCPBox can:
- store projects and MCP server definitions in SQLite;
- start local MCP servers as child processes;
- expose a per-project SSE endpoint for external AI clients;
- forward JSON-RPC messages from HTTP to MCP stdio;
- serve an embedded admin UI placeholder.

## Current Status

This repository is in the early foundation stage.

Implemented now:
- Go backend with HTTP API;
- SQLite storage through GORM;
- process orchestration in `internal/orchestrator`;
- SSE bridge on `/connect/{project_token}`;
- basic project/server management API;
- embedded placeholder admin page.

Not implemented yet:
- full React + Tailwind admin UI;
- advanced session routing for multiple MCP servers per project;
- production-grade auth and access control;
- structured logging and metrics;
- install/service scripts.

## Requirements

- Go 1.25+
- Windows, Linux, or macOS

No external database is required. SQLite file is created locally.

## Project Structure

```text
main.go                      Application entry point
internal/models              GORM models
internal/storage             SQLite storage and queries
internal/orchestrator        MCP process lifecycle and stdio bridge
internal/httpapi             HTTP API, SSE endpoint, embedded UI
```

## Build

```bash
go build -o MCPBox .
```

On Windows:

```powershell
go build -o MCPBox.exe .
```

## Run

Default port is `38180`.

Run with default settings:

```bash
go run .
```

On Windows binary:

```powershell
.\MCPBox.exe
```

## Port Configuration

Port can be configured in two ways.

Priority order:
1. CLI flag `-port`
2. Environment variable `MCPBOX_PORT`
3. Default value `38180`

Examples:

```bash
go run . -port 39000
```

```powershell
.\MCPBox.exe -port 39000
```

```powershell
$env:MCPBOX_PORT=39000
.\MCPBox.exe
```

Recommendation:
- use `-port` for manual local runs and shortcuts;
- use `MCPBOX_PORT` for scripts, CI, or service wrappers.

## Data Storage

By default MCPBox creates a local SQLite database file:

```text
mcpbox.db
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

### MCP Servers

Add server to a project:

```http
POST /api/projects/{id}/servers
Content-Type: application/json
```

```json
{
  "name": "Everything Server",
  "launch_command": "npx @modelcontextprotocol/server-everything",
  "working_dir": "",
  "auto_start": true
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

## SSE / JSON-RPC Bridge

Each project has a unique token.

Open SSE stream:

```http
GET /connect/{project_token}
```

Forward JSON-RPC request to the MCP process:

```http
POST /connect/{project_token}
Content-Type: application/json
```

Example request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

Behavior:
- `POST /connect/{project_token}` sends JSON-RPC payload to child process `stdin`;
- `GET /connect/{project_token}` receives child process `stdout` frames through SSE;
- current Stage 1 implementation uses the first configured server in the project as the active server for the connection.

## Development Notes

- Process management is isolated in `internal/orchestrator`.
- Lifecycle control uses `context.Context`.
- Shutdown attempts graceful stop before forcing process termination.
- SQLite driver is pure Go to avoid `gcc/cgo` dependency for local builds.

## Important Maintenance Rule

When major or behavior-changing updates are made, documentation should be updated in the same change set:
- `README.md` for English;
- `README-ru.md` for Russian.

This repository should stay runnable from documentation alone.

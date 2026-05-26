# MCPBox Developer Guide

Russian version: [DEVELOPER-ru.md](./DEVELOPER-ru.md)

## Overview

MCPBox is a Go-based control plane for MCP servers.

It groups MCP servers by project, stores configuration in SQLite, launches local `stdio` MCP processes, proxies remote `HTTP streaming` MCP servers, serves an embedded React admin UI, and exposes one MCP URL per project.

Today MCPBox is:
- a control plane for MCP servers
- a project-based organizer for local and remote MCP backends
- an aggregation endpoint for enabled servers inside a project
- an operator UI for project lifecycle, server lifecycle, inspection, health checks, and logging

Today MCPBox is not:
- a multi-user platform with auth and roles
- a full observability platform, even though it now includes basic built-in latency, error, and traffic metrics
- a distributed MCP orchestration system across many hosts

## Core Model

- `Project`: a logical workspace grouping MCP servers for one client, team, or environment
- `MCP Server`: either a local `stdio` server launched by MCPBox or a remote `HTTP streaming` server
- `Catalog Item`: an integration definition synchronized from an external JSON manifest
- `Installed Integration`: a project-level record that links a catalog item to a concrete `MCPServer`
- `Installed Package`: a reusable installed runtime package that can be linked into one or more projects
- `Project Package Instance`: a project-level attachment between an installed package and the concrete managed `MCPServer`
- `Performance Metric`: a lightweight per-request record used for latency, error, and traffic summaries in the logs UI

Important behavior:
- each project has exactly one MCP URL
- the project URL is `/mcp/{project_token}`
- the project endpoint aggregates all enabled servers in that project
- if a project is paused, new MCP connections are blocked
- disabled servers are excluded from the project endpoint
- local `stdio` servers can be auto-started at application boot

## Current Features

### Backend

- Go HTTP API
- SQLite storage through GORM
- orchestration for local `stdio` MCP processes
- proxy support for remote `HTTP streaming` MCP servers
- local Knowledge Base / RAG collections backed by on-disk Bleve full-text indexes
- project-level `/mcp/{project_token}` endpoint
- synchronous JSON-RPC request/response bridge for HTTP MCP clients
- legacy SSE compatibility mode for older MCP clients
- aggregated `tools`, `resources`, and `prompts` listing across enabled project servers
- forwarding of tool, prompt, and resource calls to the correct backing server
- audit logging for control actions and MCP traffic
- per-method aggregation audit logs for `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list`, and `prompts/get`
- project pause/resume
- server enable/disable
- local server inspection
- health verification on create, update, start, and manual check
- deep health probing after `initialize`, including safe follow-up tool probes for database-style and filesystem-style servers
- sanitized health-check trace logging so diagnostics are preserved without leaking secrets
- catalog sync from external JSON manifests or local uploaded manifest files
- installed integrations stored alongside regular MCP servers
- package install and uninstall lifecycle with package reuse across projects
- system dependency checks before package install
- manifest-driven secret handling so sensitive config can be moved into environment variables instead of visible CLI args
- manifest-driven post-install health checks
- Docker runtime MVP for stdio-oriented container-backed catalog items
- `go_install` support for Go-based stdio integrations, with managed binary placement inside the package install directory
- lightweight performance metrics for request count, error rate, latency, and traffic
- embedded Ollama launcher powered by `github.com/mark3labs/mcphost/sdk`
- LM Studio deeplink launcher through `POST /api/projects/{id}/launch-lmstudio`

Knowledge Base note:
- the current RAG layer uses classic local full-text search, not embedding indexes
- collections are stored on disk and searched through Bleve
- no external vector database is required
- no embedding generation pipeline is required at this stage
- future `Pro` capabilities are documented separately in [PRO-ROADMAP.md](./PRO-ROADMAP.md)

### Frontend

- embedded React admin UI
- project list and project overview
- modal create-project flow
- project duplication flow with rename-before-create
- modal add-server flow for `stdio` and `HTTP streaming`
- Market tab for catalog sync, package install, uninstall, and add-to-project flows
- automatic catalog sync when the Market view is opened
- project MCP URL display
- start/stop controls for local servers
- health-check status and manual `Check` action
- immediate health-check success or failure feedback after manual checks
- audit log console with project filtering
- built-in performance dashboard inside the logs screen
- auto-refreshing log view
- `Info` modal for `stdio` servers
- local Ollama status detection and model picker
- one-click `Launch Ollama` action for eligible projects
- `Launch Project` dialog for choosing local launch flows
- `Add to LM Studio` action from the project UI
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
- `POST /mcp/{project_token}` without `sessionId` works as synchronous HTTP JSON-RPC
- `GET /mcp/{project_token}` opens the legacy SSE flow and returns an `endpoint` event
- follow-up SSE requests use `POST /mcp/{project_token}?sessionId=...`
- paused projects are blocked before transport is opened
- only enabled servers participate in the project endpoint
- MCPBox answers top-level capability discovery itself, then fans out list and call operations to project servers

Why this matters:
- LM Studio expects synchronous HTTP request/response for calls such as `initialize` and `tools/list`
- older SSE-based clients still need the legacy session model
- MCPBox supports both without requiring separate project URLs

## Aggregation Behavior

`/mcp/{project_token}` is an aggregation endpoint, not a single-primary-server route.

Current behavior:
- `initialize` is answered by MCPBox itself
- `tools/list`, `resources/list`, and `prompts/list` combine results from enabled project servers
- MCPBox adds stable aliases when needed so duplicate names from different servers can coexist
- tool calls, prompt fetches, and resource reads are routed back to the originating server
- aggregated calls also write normalized audit entries so downstream activity can be traced per method and per backing server

This is why one project can present several MCP backends through one URL while keeping client configuration stable.

## Local `stdio` Inspection

For local `stdio` servers MCPBox can inspect live MCP capabilities.

The `Info` action in the UI is available only for `stdio` servers. It can show:
- server metadata from `initialize`
- negotiated MCP capabilities
- exposed `tools`
- exposed `resources`
- exposed `prompts`
- nearby `README.md` if MCPBox finds one next to the configured local path

Remote `HTTP streaming` servers intentionally do not expose this UI action.

## Server Health Checks

MCPBox verifies MCP server operability before users discover problems through an AI client.

Current behavior:
- when a server is created, MCPBox runs a health check and stores the result
- when a server is edited, MCPBox re-checks the updated configuration
- when a local `stdio` server is started manually, start is treated as successful only if MCP health verification passes
- the UI exposes a manual `Check` action for both local and remote servers
- the `Check` API response now returns the refreshed health status, error text, and timestamp immediately

Current verification strategy:
- local `stdio` servers are checked through a real MCP handshake: `initialize`, `notifications/initialized`, and follow-up list calls
- remote `HTTP streaming` servers are checked through the same JSON-RPC flow over HTTP, not only a shallow connectivity test
- when tools are exposed, MCPBox may run a safe probe such as `query`, `read_query`, `list_database`, or filesystem listing to catch broken runtime configuration earlier
- unsupported list methods such as `resources/list` or `prompts/list` are tolerated when the upstream server clearly does not implement them
- the last health state, error text, and timestamp are persisted in SQLite and shown in the admin UI

Trace behavior:
- each health run can emit `server_health_trace` audit entries for request, response, and failure diagnostics
- secrets are masked in command arguments, URLs, headers, and JSON payloads before trace details are written

## Catalog And Integrations

MCPBox can synchronize an external catalog manifest into SQLite and let users install those entries into projects.

Relevant API routes:

```http
GET /api/catalog/items
GET /api/catalog/items?enabled_only=1
POST /api/catalog/sync
GET /api/packages
DELETE /api/packages/{id}
POST /api/projects/{id}/integrations
```

Installed catalog items create regular project-linked `MCPServer` records, so the project endpoint remains `/mcp/{project_token}` with no special transport case.

Catalog sync notes:
- `POST /api/catalog/sync` accepts either a remote `url` or `manifest_content` plus `file_name`
- legacy catalog URLs such as `https://webeasy.kz/mcpbox/catalog.json` are normalized to `https://mcpbox.sh/catalog.json`
- sync failures are written both to the UI and to the audit log

Manifest notes:
- `icon_url` is supported for catalog cards and dialogs
- `system_dependencies` can block package install when required binaries such as `git`, `psql`, or `docker` are missing
- `default_env` can be sent as an object or as an array of `{ key, value }` pairs
- `env_schema` describes environment variables for the install/add-to-project dialog, including `secret: true`
- config fields can also declare `secret: true` and `env_var` so MCPBox stores the secret in `server.EnvJSON` instead of leaving it in visible CLI args
- `health_check` can force post-install verification and optionally block add-to-project when the integration does not come up cleanly
- Docker catalog items currently target stdio-style `docker run --rm -i ...` flows and `docker_pull` installation
- Go catalog items can now use `install.strategy: "go_install"` together with `source.package` and an entry point so the installed binary is copied into the managed package directory

Package lifecycle notes:
- package install happens once per catalog item version
- package uninstall is blocked while any project still references that package
- `Add to project` creates a fresh project package instance and a regular managed `MCPServer`

Project lifecycle note:
- `POST /api/projects/{id}/duplicate` clones project metadata, servers, package instances, installed integrations, and connected knowledge-base links while generating a fresh token for the new project

## Ollama Integration

MCPBox includes an embedded local Ollama launcher for one-click MCP testing.

Current behavior:
- the UI checks `GET /api/ollama/status`
- the button is shown only when `ollama` is installed
- the status response includes discovered local models and a default model
- project launch uses `POST /api/projects/{id}/launch-ollama`
- MCPBox writes a temporary `mcphost` config pointing back to the current project endpoint
- MCPBox opens a new terminal session and runs its own `ollama-chat` subcommand
- the `ollama-chat` subcommand uses `github.com/mark3labs/mcphost/sdk`, so no separate `mcphost` binary is required

Practical requirement:
- `ollama` must still be installed locally, because MCPBox launches a local Ollama-backed session rather than bundling the model runtime itself

## LM Studio Integration

MCPBox can prepare a local project for LM Studio without asking the user to build the config manually.

Current behavior:
- the UI calls `POST /api/projects/{id}/launch-lmstudio`
- MCPBox validates that the project is not paused and has at least one enabled MCP server or connected knowledge base
- the backend builds an `lmstudio://add_mcp` deeplink that points back to the project endpoint
- the generated server name is normalized from the project name and prefixed with `mcpbox_{project_id}_...`
- MCPBox asks the local OS to open the deeplink and returns the deeplink payload in the API response

## Startup Behavior

At application startup MCPBox loads all projects from storage and decides which local servers should be started automatically.

Current rule:
- if a project is not paused and has at least one enabled `stdio` server with `auto_start=true`, MCPBox starts all enabled `stdio` servers in that project

This is intentionally project-oriented behavior, not first-server-only behavior.

## Logging And Control

MCPBox includes an audit trail intended for operational control.

Currently it logs events such as:
- project creation
- server creation
- project pause/resume
- server start/stop
- server enable/disable
- MCP connection attempts
- forwarded JSON-RPC payloads
- health-check activity
- per-method aggregated MCP traffic
- Ollama launch actions
- LM Studio launch preparation and deeplink open actions

Operational note:
- common informational stderr lines from Filesystem-style MCP servers are filtered so they do not appear as misleading access errors
- SQL query logging from GORM is disabled by default so ordinary MCP traffic does not flood the console

In addition to audit logs, MCPBox now stores lightweight performance metrics.

Relevant API route:

```http
GET /api/logs/metrics
GET /api/logs/metrics?project_id={id}&window=5m
GET /api/logs/metrics?project_id={id}&window=1h
GET /api/logs/metrics?project_id={id}&window=24h
```

Current metrics behavior:
- metrics are recorded around proxied JSON-RPC calls and managed server method calls
- each metric records project, server, transport, operation, latency, request bytes, response bytes, and success/failure
- the logs screen uses these records to render summary cards, trend charts, top slow servers, top error servers, top traffic servers, and recent failures
- this is intentionally a lightweight in-app operational view, not a Prometheus or OpenTelemetry-style external observability system

## Requirements

- Go `1.26+`
- Node.js and npm for building the embedded UI
- Windows, Linux, or macOS

No external database is required. MCPBox creates a local SQLite file by default.

## Project Structure

```text
main.go                      application entry point and startup orchestration
internal/models              GORM models
internal/storage             SQLite storage and queries
internal/orchestrator        MCP process lifecycle, inspection, stdio bridge
internal/httpapi             HTTP API, MCP endpoint, embedded UI, Ollama and LM Studio launch APIs
internal/ollamahost          embedded Ollama chat host built on mcphost/sdk
internal/installer           local package installation service
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

Run the embedded local Ollama host directly:

```bash
go run . ollama-chat --config /path/to/project.yml --model llama3.2
```

When MCPBox starts successfully, it opens the local UI automatically and prints:
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

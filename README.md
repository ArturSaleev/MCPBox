# MCPBox

MCPBox is a Go-based control plane for MCP servers. It lets you group local and remote MCP backends by project, inspect what they expose, monitor traffic, and publish a single project URL for AI clients such as Claude, Cursor, Codex, LM Studio, and local Ollama-based workflows.

<p align="center">
  <a href="images/main.png">
    <img src="images/main.png" alt="MCPBox main view" width="32%" height="150px" />
  </a>
  <a href="images/market.png">
    <img src="images/market.png" alt="MCPBox market view" width="32%" height="150px" />
  </a>
  <a href="images/log.png">
    <img src="images/log.png" alt="MCPBox logs view" width="32%" height="150px" />
  </a>
</p>

<p align="center">
  <sub>Click any screenshot to open it in full size.</sub>
</p>

## Why MCPBox

Managing several MCP servers quickly turns into a mix of JSON snippets, terminal tabs, and half-documented local setup. MCPBox gives you one embedded UI and one local binary for the operational side of that work.

Key capabilities:
- Project-based organization for local `stdio` and remote `HTTP streaming` MCP servers
- One project MCP endpoint at `/mcp/{project_token}` that aggregates all enabled servers in the project
- Live inspection of `tools`, `resources`, `prompts`, and nearby README files for local `stdio` servers
- Health checks on create, update, manual check, and local server start
- Audit log for MCP traffic and control actions
- Project pause/resume and per-server enable/disable controls
- Embedded Market / Catalog flow for installing integrations into projects
- One-click Ollama launcher for local MCP testing through embedded `mcphost/sdk`
- Single-binary deployment with embedded React UI and local SQLite storage

## What Is New In 1.1.0

- Embedded Ollama support: launch a local `ollama + MCPBox` chat flow from the project UI without installing `mcphost` separately
- Aggregated project endpoint: enabled project servers are exposed through the same project MCP URL
- Better project startup: if a project has any enabled `stdio` server with `auto_start=true`, MCPBox starts all enabled `stdio` servers in that project
- Cleaner Filesystem MCP handling: common informational stderr lines are no longer treated like access failures
- New Market page and catalog flow: integration discovery, compact sync action, single-column cards, and tighter action placement

## Quick Start

1. Download the binary for your OS from the latest release, or build from source.
2. Run `MCPBox`.
3. Open `http://127.0.0.1:38180/` if the browser does not open automatically.
4. Create a project.
5. Add local `stdio` servers or remote `HTTP streaming` servers.
6. Copy the project endpoint from the UI and connect your AI client to `/mcp/{project_token}`.

For local Ollama testing:
1. Make sure `ollama` is installed.
2. Open a project that has at least one enabled MCP server.
3. Choose a local Ollama model in the UI.
4. Click `Launch Ollama`.

## Build From Source

Requirements:
- Go `1.26+`
- Node.js and npm

Build steps:

```bash
npm --prefix html install
npm --prefix html run build
go build -o MCPBox .
```

Run from source:

```bash
go run .
```

Default port: `38180`

## Documentation

- [README-ru.md](./README-ru.md) - Russian user guide
- [DEVELOPER.md](./DEVELOPER.md) - developer guide, API notes, and architecture
- [DEVELOPER-ru.md](./DEVELOPER-ru.md) - Russian developer guide
- [RELEASE-1.1.0.md](./RELEASE-1.1.0.md) - release notes draft for version `1.1.0`

## Notes

- MCPBox uses a local SQLite database by default.
- The Ollama launcher is shown only when `ollama` is installed on the machine.
- Local server inspection is available for `stdio` servers only.

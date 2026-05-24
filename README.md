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
- Built-in Knowledge Base / RAG collections with local indexing for code, text, CSV, XLSX, DOCX, PPTX, and text-based PDF
- Live inspection of `tools`, `resources`, `prompts`, and nearby README files for local `stdio` servers
- Health checks on create, update, manual check, and local server start
- Audit log for MCP traffic and control actions
- Project pause/resume and per-server enable/disable controls
- Embedded Market / Catalog flow for syncing, installing, verifying, and uninstalling integrations
- One-click Ollama launcher for local MCP testing through embedded `mcphost/sdk`
- Single-binary deployment with embedded React UI and local SQLite storage

## What Is New In 1.2.1

- Market page cleanup and refactor so the catalog UI is no longer a large monolithic block inside `App.tsx`
- Catalog source switcher with support for syncing from a remote URL or a local JSON file
- Default catalog source migrated from `webeasy.kz` to [mcpbox.sh](https://mcpbox.sh)
- Better Market UX with improved install flow, clearer installed state, and package usage visibility
- Installed package lifecycle controls including safe uninstall when a package is not used by any project
- `Add to project` is shown only for packages that are already installed
- Project duplication flow with a new name and full cloning of servers, integrations, package links, and connected knowledge bases
- Project creation/edit UI is simplified by removing the root path field from the modal
- Manifest support for `icon_url`
- Manifest support for `system_dependencies` with pre-install runtime checks
- Manifest support for `default_env`, `env_schema`, `secret: true`, and `env_var`
- Secrets from catalog install forms now flow into environment variables instead of staying in visible command arguments
- Project UI masks sensitive command arguments and secret environment/header values
- Catalog sync failures are now written to the audit log
- Manifest support for `health_check` to verify integrations after adding them to a project
- Docker runtime MVP with `runtime.type: "docker"` and `install.strategy: "docker_pull"` for stdio-oriented Docker-backed MCP servers
- Better Python runtime fallback so installations work on systems with `python3` but no `python`
- Logs view now includes built-in performance monitoring with latency, error, traffic, top-server breakdowns, and compact trend charts

## What Is New In 1.2.0

- Built-in Knowledge Base / RAG workflow with a dedicated `Knowledge Base` page in the UI
- Global knowledge collections that can be connected to one or many projects
- Local indexing and search for code, text, CSV, XLSX, DOCX, PPTX, and text-based PDF
- Project-level MCP tool `search_project_knowledge` for querying connected knowledge bases through the project endpoint
- Better audit visibility for Knowledge Base tool calls and search activity
- Improved local document context with section metadata such as `Sheet`, `Slide`, and `Page`

## What Was New In 1.1.1

- Windows Ollama launch fix: the embedded `Launch Ollama` flow now starts correctly on Windows through a dedicated PowerShell path
- Safer command execution on Windows: quoted paths and paths with spaces are handled more reliably
- Better Windows terminal startup: the launched chat session now respects the project working directory
- Automatic Windows recovery: if the Ollama daemon is not ready, MCPBox starts `ollama serve` before opening chat

## What Was New In 1.1.0

- Embedded Ollama support for one-click local MCP testing
- New Market page and catalog-based integration flow
- Better project auto-start so all enabled local servers come up correctly
- Cleaner Filesystem MCP logs during normal operation

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

## Notes

- MCPBox uses a local SQLite database by default.
- The current Knowledge Base / RAG implementation uses local full-text indexing with Bleve. It does not build or require embedding indexes at this stage.
- The Docker runtime support in `1.2.1` is intentionally an MVP. Advanced container features such as compose-style orchestration, custom networks, and volume presets are not fully implemented yet.
- Secret masking is applied in the UI and catalog install flow. Existing manually configured servers that still pass secrets directly in command arguments should be migrated to environment variables for full process-list safety.
- Future `Pro` capabilities are documented separately in `PRO-ROADMAP.md` and are not part of the current `1.2.1` implementation.
- The Ollama launcher is shown only when `ollama` is installed on the machine.
- Local server inspection is available for `stdio` servers only.

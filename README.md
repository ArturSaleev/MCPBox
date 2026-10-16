# MCPBox

MCPBox is one open-source application under Apache-2.0. It includes the former MCPBoxPro features: local accounts, roles, sessions, agent tokens, OIDC SSO, PolicyBox and RagBox integrations, and storage configuration. No commercial license or activation is required.

Build independently from this repository (Go 1.26+, Node.js/npm):

```bash
cd html
npm ci
npm run build
cd ..
GOWORK=off go build -o MCPBox .
./MCPBox admin create --email admin@example.com --password 'replace-with-a-unique-password'
./MCPBox
```

Admin UI: `http://127.0.0.1:38181/`; project MCP: `http://localhost:38182/mcp/<project-token>`. Override listeners with `MCPBOX_HOST`, `MCPBOX_PORT`, `MCPBOX_MCP_HOST`, and `MCPBOX_MCP_PORT`. Sign in with the account created above. Interactive startup can also create the first administrator.

See [migration and compatibility](docs/MERGE.md) before replacing an existing installation. All code and frontend sources are contained in this repository; the MCPBoxPro directory is no longer required. Release builds: `./build-release.sh host` or `./build-release.sh all`.


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
- One-click Ollama and llama.cpp launchers for local MCP testing
- Single-binary deployment with embedded React UI and local SQLite storage

## Release 1.3.0

See [release notes](release/RELEASE-1.3.0.md) and [upgrade instructions](docs/MERGE.md).

## Quick Start

1. Download the binary for your OS from the latest release, or build from source.
2. Run `MCPBox`.
3. Open `http://127.0.0.1:38181/` if the browser does not open automatically.
4. Sign in with your local administrator account, then create a project.
5. Add local `stdio` servers or remote `HTTP streaming` servers.
6. Copy the project endpoint from the UI and connect your AI client to `/mcp/{project_token}`.

For local Ollama testing:
1. Make sure `ollama` is installed.
2. Open a project that has at least one enabled MCP server.
3. Choose a local Ollama model in the UI.
4. Click `Launch Ollama`.

For LM Studio:
1. Open a project that has at least one enabled MCP server or connected knowledge base.
2. Click `Launch Project`.
3. Choose `Add to LM Studio`.
4. Confirm the deeplink in LM Studio when your OS opens it.

For local llama.cpp Web UI testing:
1. Make sure `llama-server` is installed and available in `PATH`.
2. Set `MCPBOX_LLAMACPP_MODEL` or choose a local `.gguf` model in the launch dialog.
3. Open a project with an enabled MCP server or connected knowledge base.
4. Click `Launch Project` and choose the llama.cpp action.
5. MCPBox starts `llama-server`, applies the project prompt as a system prompt, and opens the Web UI.

## Build From Source

Requirements:
- Go `1.26+`
- Node.js and npm

Build steps:

```bash
npm --prefix html ci
npm --prefix html run build
GOWORK=off go build -o MCPBox .
```

Run from source:

```bash
GOWORK=off go run .
```

Default port: `38181`

## Documentation

- [README-ru.md](./README-ru.md) - Russian user guide
- [DEVELOPER.md](./DEVELOPER.md) - developer guide, API notes, and architecture
- [DEVELOPER-ru.md](./DEVELOPER-ru.md) - Russian developer guide

## Notes

- MCPBox uses a local SQLite database by default.
- Local Knowledge Base / RAG uses full-text indexing with Bleve without mandatory embedding indexes. Optional RagBox integration adds external ingestion and retrieval.
- The Docker runtime support in `1.2.1` is intentionally an MVP. Advanced container features such as compose-style orchestration, custom networks, and volume presets are not fully implemented yet.
- The `go_install` package flow in `1.2.2` assumes the target integration can be installed through `go install module/path@version` and exposed through a concrete binary entry point.
- Secret masking is applied in the UI and catalog install flow. Existing manually configured servers that still pass secrets directly in command arguments should be migrated to environment variables for full process-list safety.
- The Ollama launcher is shown only when `ollama` is installed on the machine.
- The llama.cpp launcher requires `llama-server`; project prompts are passed through `--system-prompt-file` and are also exposed through MCP `prompts/list` as `project_prompt`.
- Local server inspection is available for `stdio` servers only.

[CLI Registry: configuration, API, MCP and execution boundaries](docs/CLI.md).

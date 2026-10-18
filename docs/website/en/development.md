# Architecture and development

[Documentation](README.md) | [Русский](../ru/development.md)

MCPBox is one Go module with an embedded React UI. `main.go` wires accounts, SSO, integrations and authorization into the `app` runtime. No sibling MCPBoxPro directory is needed to build it.

## Source map

| Path | Responsibility |
| --- | --- |
| `app/`, `main.go` | Runtime, listeners, startup, admin commands |
| `internal/httpapi/` | Core HTTP API, project MCP aggregation, local launchers |
| `internal/orchestrator/` | Local processes, inspection and health |
| `internal/storage/` | Core models and SQLite persistence |
| `internal/installer/` | Marketplace packages |
| `internal/cli/` | CLI registry, catalog and bounded execution |
| `internal/rag/` | Local extraction, Bleve indexes and search |
| `internal/proauth/`, `internal/prosso/` | Users, roles, sessions, tokens, OIDC |
| `internal/prohttp/` | Authenticated administration and integration routes |
| `internal/propolicy/`, `internal/proprojects/` | PolicyBox bindings and project shadows |
| `internal/proragbox/` | External RagBox integration |
| `internal/prostorage/` | Storage configuration and operational connections |
| `connectruntime/` | Shared connection runtime |
| `html/src/` | Frontend sources |
| `internal/httpapi/ui/dist/` | Embedded frontend build output |

## Build and validation

Use Go 1.26+ and Node.js/npm compatible with Vite in `html/package.json`. From the repository root:

```bash
npm --prefix html ci
npm --prefix html run build
GOWORK=off go build -o MCPBox .
GOWORK=off go test ./...
```

After UI changes, rebuild both frontend and Go binary. `npm --prefix html run dev` starts Vite, but its current configuration has no API proxy; use the embedded build to validate the complete application. For persistent data use the built executable: `go run` places the binary in a temporary directory.

## Request flow and boundaries

The UI calls the protected Admin API. MCP clients call `/mcp/{project-token}` on a separate listener. Project and authorization checks precede calls to available backends. Local `stdio` runs on the MCPBox machine; remote `http_stream` uses the network. RagBox and PolicyBox run separately. A project prompt does not replace permissions in the target system.

SQLite remains the bootstrap layer; PostgreSQL does not imply migration of every core table. Verify table ownership and runtime/shadow status when changing storage. Internal `pro*` names remain for compatibility.

## Releases

Install frontend dependencies first (`npm ci` above), then run `./build-release.sh host` or `./build-release.sh all`. The script rebuilds the UI and produces `release/MCPBox-<os>-<arch>` for macOS, Linux and Windows on amd64/arm64. Windows binaries use `.exe`. The script also generates `release/SHA256SUMS` for the current run; documentation and LICENSE remain in the repository. Use `macos` for macOS asset names.

Details: [developer guide](../../../DEVELOPER.md), [HTTP API](api.md), [CLI reference](../../CLI.md). Update both documentation languages when behavior changes. Backend tests do not establish connectivity to external OAuth, PolicyBox/RagBox or real MCP clients; validate those integrations separately.

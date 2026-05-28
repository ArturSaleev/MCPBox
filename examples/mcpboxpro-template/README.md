# MCPBox Pro Template

This folder is a starter scaffold for the private `mcpboxpro` repository.

## Suggested Local Layout

```text
workspace/
  MCPBox/
  mcpboxpro/
  go.work
```

## Files To Copy Into The Private Repository

- `main.go`
- `go.mod`

Optional:
- `go.work.example` as a starting point for your local shared workspace

## First Local Setup

1. Clone both repositories side by side.
2. Copy this template into the `mcpboxpro` repository root.
3. Create a top-level `go.work` one directory above both repos.
4. Run:

```bash
export MCPBOX_PRO_BOOTSTRAP_TOKEN=change-me
go work use ./MCPBox ./mcpboxpro
cd mcpboxpro
go mod tidy
go run .
```

## How Development Flows

Shared changes:
- go into `MCPBox`

Pro-only changes:
- go into `mcpboxpro`

The `replace` directive in `go.mod` makes local Pro development use the sibling `MCPBox` checkout, so private code sees shared changes immediately.

## Next Expected Pro Steps

- the template already includes a first real Pro MVP:
  - `internal/proauth`
  - `internal/prohttp`
  - `GET /api/pro/auth/me`
  - `GET /api/pro/tokens`
  - `POST /api/pro/tokens`
  - `DELETE /api/pro/tokens/{id}`
- set `MCPBOX_PRO_BOOTSTRAP_TOKEN` before first run
- then build the UI around these endpoints

# Configuration reference

[Documentation](README.md) | [Русский](../ru/configuration.md)

## Listeners and launchers

| Variable | Default | Purpose |
| --- | --- | --- |
| `MCPBOX_HOST` | `127.0.0.1` | Admin UI/API bind address |
| `MCPBOX_PORT` | `38181` | Admin UI/API TCP port |
| `MCPBOX_MCP_HOST` | `127.0.0.1` | MCP bind address (local machine) |
| `MCPBOX_MCP_PORT` | `38182` | MCP TCP port |
| `MCPBOX_PUBLIC_MCP_URL` | unset | Public HTTPS origin for MCP/OAuth; no path |
| `MCPBOX_PRO_BOOTSTRAP_TOKEN` | unset | Emergency admin bearer credential |
| `MCPBOX_OLLAMA_MODEL` | unset | Preferred local Ollama model |
| `MCPBOX_LLAMACPP_MODEL` | unset | Local GGUF model path |
| `MCPBOX_LLAMACPP_MODEL_NAME` | unset | Model name override |
| `MCPBOX_LLAMACPP_PORT` | `39200` | llama.cpp listener port |
| `MCPBOX_LLAMACPP_CHAT_TEMPLATE_FILE` | unset | Chat template file |
| `MCPBOX_LLAMACPP_CTX_SIZE` | unset | Passed as context size to llama-server |
| `MCPBOX_LLAMACPP_N_GPU_LAYERS` | unset | Passed as GPU layer count to llama-server |

Listener precedence: command-line flag, environment, edition default. Flags: `--host`, `--port`, `--mcp-host`, `--mcp-port`. Invalid ports are rejected. For a local-only installation:

```bash
MCPBOX_MCP_HOST=127.0.0.1 ./MCPBox
```

Set variables in the service environment or shell. No `.env` loader is configured by the entry point. Child processes need their runtimes and binaries in the service account's PATH. Ollama and llama.cpp are optional external installations.

## Public MCP URL and OAuth

Set `MCPBOX_PUBLIC_MCP_URL=https://mcp.example.com` in the MCPBox process environment and restart the service. Use an HTTPS origin only: no credentials, path, query or fragment. An optional trailing slash is accepted. Invalid values are ignored; without a valid value, MCPBox uses request/proxy-based address detection.

This origin is used for OAuth discovery, protected-resource metadata and the preferred project connection URL (`https://mcp.example.com/mcp/<project-token>`). The UI displays and copies the URL supplied by the backend, even when the admin panel uses another domain or port. This setting does not change listener bind addresses or configure a proxy.

With separate listeners, `/.well-known/` and `/oauth/` are served on the MCP listener (`38182` by default); the admin listener returns 404 for these routes. Forward these paths and `/mcp/` to the MCP listener. OIDC sign-in to the administration panel still uses `/api/pro/sso/callback` on the admin listener. See [ChatGPT connection](chatgpt.md) and [network access](network-access.md).

## OIDC SSO

Configure `MCPBOX_PRO_OIDC_ISSUER_URL`, `MCPBOX_PRO_OIDC_CLIENT_ID`, `MCPBOX_PRO_OIDC_CLIENT_SECRET`, and `MCPBOX_PRO_OIDC_REDIRECT_URL`. The callback path is `/api/pro/sso/callback`; register its exact public URL with your identity provider.

| Optional variable | Default |
| --- | --- |
| `MCPBOX_PRO_OIDC_PROVIDER_NAME` | `OIDC` |
| `MCPBOX_PRO_OIDC_SCOPES` | `openid profile email` (space-separated) |
| `MCPBOX_PRO_OIDC_DEFAULT_ROLE` | `reader` |
| `MCPBOX_PRO_OIDC_SESSION_DAYS` | `30` |
| `MCPBOX_PRO_OIDC_AUTO_CREATE_USERS` | `true` |
| `MCPBOX_PRO_OIDC_ALLOWED_HD` | empty |

The `MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET` setting is no longer used. See [migration](policybox-migration.md).

## Data and integrations

The default `mcpbox.db` is next to the resolved executable, independently of the current working directory. Runtime data includes `package_store/`, `knowledge_base/managed/`, `knowledge_base/indexes/`, and compatibility directory `.mcpboxpro/rag-shadow/` under the data root. Keep the installation writable for its service account.

The Go embedding API accepts `app.Options.StoreDSN`; it is not a documented executable `--db` flag or environment variable. Relative DSNs are resolved beside the executable; absolute and special `:` / `file:` DSNs are preserved. `go run .` uses a temporary executable: build a persistent binary for a persistent installation.

Configure RagBox, PostgreSQL and log backends in Settings. SQLite remains the bootstrap database. Selecting PostgreSQL does not migrate every core table. Inspect storage runtime and shadow status after a change; see [integrations](knowledge-integrations.md).

Compatibility names `MCPBOX_PRO_*`, `/api/pro/*`, `pro:*`, and `.mcpboxpro` remain in the unified application and do not require a paid edition.

Sources: [runtime](../../../app/runtime.go), [entry point](../../../main.go), [SSO](../../../internal/prosso/service.go), [launcher](../../../internal/httpapi/llamacpp.go).

Defaults above describe current source. Published v1.3.0 uses the earlier MCP bind `0.0.0.0`; explicitly set `--mcp-host 127.0.0.1` until updating. Enable direct remote access with `--mcp-host 0.0.0.0` or `MCPBOX_MCP_HOST=0.0.0.0`. [Network access and project isolation](network-access.md).

## Appearance

Use the sun, moon or monitor button at the bottom of the navigation to choose **Light**, **Dark** or **System**. The same control is available on the sign-in screen. System follows the operating system appearance, including changes while MCPBox is open. An explicit Light/Dark choice overrides the system preference. The choice is saved in this browser under `mcpbox-theme` and applies to pages, forms, menus, dialogs and notifications; it does not change server or project settings.

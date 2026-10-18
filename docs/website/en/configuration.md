# Configuration reference

[Documentation](README.md) | [Русский](../ru/configuration.md)

## Listeners and launchers

| Variable | Default | Purpose |
| --- | --- | --- |
| `MCPBOX_HOST` | `127.0.0.1` | Admin UI/API bind address |
| `MCPBOX_PORT` | `38181` | Admin UI/API TCP port |
| `MCPBOX_MCP_HOST` | `0.0.0.0` | MCP bind address (all IPv4 interfaces) |
| `MCPBOX_MCP_PORT` | `38182` | MCP TCP port |
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

`MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET` configures the signed runtime-context integration; see the [PolicyBox protocol notes](../../POLICYBOX_RUNTIME_INTEGRATION_RU.md) before enabling that flow. It is separate from the admin bootstrap credential.

## Data and integrations

The default `mcpbox.db` is next to the resolved executable, independently of the current working directory. Runtime data includes `package_store/`, `knowledge_base/managed/`, `knowledge_base/indexes/`, and compatibility directory `.mcpboxpro/rag-shadow/` under the data root. Keep the installation writable for its service account.

The Go embedding API accepts `app.Options.StoreDSN`; it is not a documented executable `--db` flag or environment variable. Relative DSNs are resolved beside the executable; absolute and special `:` / `file:` DSNs are preserved. `go run .` uses a temporary executable: build a persistent binary for a persistent installation.

Configure RagBox, PolicyBox, PostgreSQL and log backends in Settings. SQLite remains the bootstrap database. Selecting PostgreSQL does not migrate every core table. Inspect storage runtime and shadow status after a change; see [integrations](knowledge-integrations.md).

Compatibility names `MCPBOX_PRO_*`, `/api/pro/*`, `pro:*`, and `.mcpboxpro` remain in the unified application and do not require a paid edition.

Sources: [runtime](../../../app/runtime.go), [entry point](../../../main.go), [SSO](../../../internal/prosso/service.go), [launcher](../../../internal/httpapi/llamacpp.go).

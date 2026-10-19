# API overview

[Documentation](README.md) | [Русский](../ru/api.md)

MCPBox has a web UI and HTTP API. Every administration endpoint, including the shared `/api/projects`, `/api/servers`, `/api/logs`, and `/api/projects/{id}/oauth-clients` routes, requires an authenticated admin session or token. Reads enforce `pro:read`, mutations enforce `pro:write`, and deletion or secret-token reveal requires `pro:admin`. Health/meta, login and SSO routes, and MCP/OAuth protocol endpoints remain public. Use HTTPS and do not expose administrative API endpoints publicly without access controls.

## Core API

| Area | Main endpoints |
| --- | --- |
| Service status | `GET /healthz`, `GET /api/meta` |
| Projects | `GET, POST /api/projects`; `GET, PUT, DELETE /api/projects/{id}` |
| Project actions | `POST /api/projects/{id}/servers`; project token, bearer-token, pause/resume, and other project actions use `POST /api/projects/{id}/…` |
| Project OAuth connections | `GET, POST /api/projects/{id}/oauth-clients`; reveal token, enable, disable, or delete under `/api/projects/{id}/oauth-clients/{clientID}/…` |
| MCP servers | `GET, PUT, DELETE /api/servers/{id}`; `POST /api/servers/{id}/start` and `/stop` |
| Marketplace | `GET /api/catalog/items`; `POST /api/catalog/sync`; `POST /api/catalog/items/{id}/install` |
| Packages | `GET /api/packages`; `DELETE /api/packages/{id}` |
| Knowledge bases | `GET, POST /api/rag/collections`; `PUT, DELETE /api/rag/collections/{id}` |
| Logs and metrics | `GET /api/logs`, `GET /api/logs/metrics` |
| Project MCP | `/mcp/{project-token}` |

## Admin API

| Area | Main endpoints |
| --- | --- |
| Authentication | `POST /api/pro/login`, `GET /api/pro/auth/me`, OIDC routes under `/api/pro/sso/…` |
| Users, sessions, tokens | `/api/pro/users`, `/api/pro/sessions`, `/api/pro/tokens` |
| RagBox configuration | `GET, PUT /api/pro/ragbox/config` |
| RagBox administration | `/api/pro/ragbox/admin/vector-connections`, `/api/pro/ragbox/admin/workspaces` |
| RagBox knowledge bases | `GET /api/pro/ragbox/knowledge-bases`, `POST /api/pro/ragbox/knowledge-bases/sync` |
| Upload to knowledge base | `POST /api/pro/ragbox/collections/{collectionID}/files:upload` with multipart fields `file` and `storage_mode=keep|transient` |
| Storage | `GET, PUT /api/pro/storage/config`; `GET /api/pro/storage/runtime-status` and `/shadow-status` |

The API evolves with MCPBox releases. Treat this page as an operational overview and use the running service’s UI and release documentation for the exact payload fields. Do not automate destructive operations until they have been tested in a non-production environment.

## Administrative request example

Create an API token in the UI with the required scope and set it in the environment as `MCPBOX_ADMIN_TOKEN`. A project token does not replace an admin credential.

```bash
curl --fail-with-body http://127.0.0.1:38181/healthz
curl --fail-with-body \
  -H "Authorization: Bearer ${MCPBOX_ADMIN_TOKEN}" \
  http://127.0.0.1:38181/api/projects
```

`MCPBOX_ADMIN_TOKEN` is a curl example variable, not a server setting. `POST /api/pro/login` accepts JSON `email` and `password` and returns `token`, `user`, `session`; use that token in the Bearer header. Do not publish login responses in logs.

General read/write scopes do not cover every privileged route: Storage updates and CLI mutations, for example, require `pro:admin`. Exact scope checks are defined alongside route registration.

For `401`, check credentials; for `403`, check permissions. Send MCP JSON-RPC to the MCP listener, not administrative requests. An HTTP health check does not establish that an MCP call succeeds.

## Contract sources

- [Core routes](../../../internal/httpapi/server.go)
- [Login and SSO](../../../internal/prohttp/routes_auth.go)
- [Storage routes](../../../internal/prohttp/routes.go)
- [CLI schema and API](../../CLI.md)
- [Authentication middleware](../../../internal/proauth/middleware.go)

`POST /api/projects/{id}/migrate-policybox` requires `pro:admin`: rotates the bearer credential, clears the legacy flag, and pauses the project. Retired `/api/pro/policybox/*`, `/connect/runtime`, and `/api/agentbox/bootstrap` return `410 Gone`. Direct bootstrap remains: `POST /api/agentbox/direct-bootstrap` accepts `mcp_url`, `page_url`, `page_title` and uses project bearer/OAuth authentication, pause and migration gates. See [migration](policybox-migration.md).

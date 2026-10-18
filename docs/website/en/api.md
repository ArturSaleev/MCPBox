# API overview

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
| PolicyBox configuration | `GET, PUT /api/pro/policybox/config` |
| PolicyBox test flow | `POST /api/pro/policybox/runtime/ticket`, `/exchange`, `/validate` |
| Storage | `GET, PUT /api/pro/storage/config`; `GET /api/pro/storage/runtime-status` and `/shadow-status` |

The API evolves with MCPBox releases. Treat this page as an operational overview and use the running service’s UI and release documentation for the exact payload fields. Do not automate destructive operations until they have been tested in a non-production environment.

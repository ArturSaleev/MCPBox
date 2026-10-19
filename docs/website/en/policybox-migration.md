# Migrating from PolicyBox

[Documentation](README.md) | [Русский](../ru/policybox-migration.md)

This guide describes current source for the planned v1.3.1 release. Published v1.3.0 still contains the old integration.

The PolicyBox client, settings tab, runtime playground, external audit fetch, startup migrations and `admin policybox` commands have been removed. MCPBox no longer calls PolicyBox. RagBox, local accounts, roles, sessions, project bearer/OAuth authentication, MCP session isolation, server/tool permissions and project pause checks remain.

## Before upgrading

1. Stop MCPBox and back up its installation directory, SQLite database, knowledge-base files, and any configured operational PostgreSQL database. Keep the corresponding old binary for rollback.
2. Identify projects with `identity_verification_enabled: true` in the authenticated `GET /api/projects` response. A PolicyBox configuration or binding alone does not make a project use runtime validation; the legacy project flag did. Do not share this response: it contains credentials.
3. Review the identity, role, scope and tenant/resource rules enforced by your old PolicyBox deployment. MCPBox bearer credentials authorize access to a project; they do not reproduce those per-user rules. Enforce required business rules in the upstream MCP server/target system, or use separate projects and restricted tools before enabling access.

## Migrate each legacy project

After upgrading, a project with the legacy flag returns `403` for MCP connections and AgentBox direct bootstrap, even if the caller supplies a valid project bearer, OAuth credential, or arbitrary user ID. Editing or resuming the project does not clear that gate. Other projects retain their existing authentication settings.

1. Sign in as an MCPBox administrator and select the project. Its migration notice replaces the old identity-verification feature.
2. Choose **Migrate to bearer and pause**, then confirm. The operation atomically clears the legacy flag, enables bearer authentication, generates a new bearer credential and pauses the project. It preserves the project URL/token, servers, prompts and knowledge-base links.
3. Review server/tool permissions and all named OAuth clients; disable any client that should no longer have access. Existing OAuth client credentials are preserved. Configure clients with the new project bearer or an approved project OAuth credential. Admin sessions/API tokens and old PolicyBox runtime tokens cannot substitute for these credentials.
4. Resume the project only after access review. Test allowed and denied calls using real clients and verify the audit trail.

For automation, call `POST /api/projects/{id}/migrate-policybox` on the Admin listener using an admin credential with `pro:admin`. Success returns project status; repeating migration returns `409` without changing a migrated project. The audit event is `project_policybox_migrated`. Treat responses containing bearer tokens as secrets.

## Routes and configuration

- `/connect/runtime`, `/api/agentbox/bootstrap` and `/api/pro/policybox/*` return `410 Gone`; ticket/exchange/validate and published manifests are no longer available.
- Use `/mcp/{project-token}` or `/connect/{project-token}` for direct MCP calls.
- AgentBox direct bootstrap remains at `POST /api/agentbox/direct-bootstrap`, with JSON `mcp_url` and optional `page_url`/`page_title`. It shares project bearer/OAuth checks, pause and migration gates with MCP connections. It does not mint runtime identities, roles, scopes, signed context headers or PolicyBox manifests. Configure the embedding client for direct mode and verify compatibility separately.
- `MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET` is no longer consumed. Saved PolicyBox URLs, tokens, bindings and manifest records are inactive.

## Data and rollback

No PolicyBox tables, stored credentials, project flags or external service data are automatically deleted. The legacy project flag remains in the schema solely as a migration gate. Preserve both bootstrap SQLite and operational storage backups if PostgreSQL was configured. Existing audit records remain viewable in the ordinary log feed; remote PolicyBox audit records are no longer fetched.

For rollback, stop MCPBox and restore the matching old binary and pre-upgrade data backups together. Replacing only the binary does not reverse an explicit migration. Reconfigure and test the old PolicyBox deployment before exposing endpoints. This change does not stop or uninstall a separately deployed PolicyBox service; handle its retirement separately after checking its other consumers.

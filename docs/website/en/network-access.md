# Network access and project isolation

[Documentation](README.md) | [Русский](../ru/network-access.md)

## Choose how clients reach MCPBox

| Scenario | Configuration |
| --- | --- |
| Qwen Studio installed on the same computer | Use `http://127.0.0.1:38182/mcp/<project-token>` directly; no tunnel |
| Cloud-hosted ChatGPT | Publish the MCP listener with `tailscale funnel 38182` or an HTTPS reverse proxy; keep project authentication enabled |
| Direct LAN / remote client | Explicitly bind to an appropriate interface with `--mcp-host <address>`; configure network access and HTTPS/authentication as appropriate |

Current source defaults to private loopback listeners: admin `127.0.0.1:38181`, MCP `127.0.0.1:38182`. Published v1.3.0 binaries predate this change and default MCP to `0.0.0.0:38182`; keep `--mcp-host 127.0.0.1` explicit when using them. An existing network deployment updating from v1.3.0 must retain its intended bind address explicitly:

```bash
./MCPBox --mcp-host 0.0.0.0
```

`MCPBOX_MCP_HOST=0.0.0.0` is the environment equivalent. CLI flags take precedence. Binding all interfaces makes the endpoint reachable wherever firewall/routing allows; it does not add authentication. Funnel works with a loopback listener on the same machine, so it does not require this remote bind.

Keep admin and MCP on separate addresses/ports. By default the MCP listener serves MCP/OAuth protocol routes and health, not `/api/projects`, `/api/pro/*` or the admin UI. Deliberately configuring identical admin/MCP listener addresses combines them and exposes the administrative surface on that listener too.

## Credentials and isolation

- The project URL token selects a project and contains 128 random bits. Without project bearer authentication, knowledge of that URL grants access to its enabled tools. Treat it as a capability secret, not a public identifier.
- With bearer authentication enabled, the URL token alone is insufficient. Each request needs that project's bearer credential or an enabled named OAuth client's credential for the same project.
- Bearer and OAuth client credentials are generated from 256 random bits. Project bearer comparison is constant time. Randomness is not a substitute for rate limiting; MCPBox does not claim exhaustive brute-force protection. Use an appropriate reverse proxy/network limit for public deployments.
- Lists and execution resolve tools, resources and prompts from enabled servers attached to the requested project. A credential or legacy SSE session belonging to another project is rejected. Disabling an OAuth client, rotating the project bearer, or pausing a project affects subsequent requests; it does not undo an already executed tool action.
- OAuth discovery and authorization/token protocol endpoints need to be reachable before login. That does not make administrative APIs public. OAuth tests cover PKCE/client validation and protected-project metadata.
- Project grouping and CLI allowlists are not OS sandboxes. Local programs run with the MCPBox OS account's permissions; target services must enforce their own data permissions. Do not treat project separation as a filesystem security boundary.

## Audit and verification

Known-project bearer denials and mismatched legacy sessions are recorded as `connect_blocked_bearer_auth` and `connect_blocked_session`, with static details rather than submitted secrets. Unknown project URLs return 404. Other operational logs can contain request metadata: protect log access and avoid sending sensitive data as names, prompts or arguments.

Regression tests exercise cross-project bearer/OAuth rejection, capability lists and attempted foreign tool/resource/prompt execution, mismatched sessions, credential revocation/rotation, pause, denial auditing and admin-route exclusion from the MCP listener. Listener tests cover the local default and explicit network overrides. These are automated local checks, not a penetration test or a live test of an external ChatGPT/Funnel connection.

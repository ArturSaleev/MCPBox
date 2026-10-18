# Run MCPBox safely in production

[Documentation](README.md) | [Русский](../ru/production-security.md)

This page is for the person who runs the server. You do not need to be an MCP expert: follow the checklist in order.

## The simple production setup

Keep MCPBox on a server inside your company network. Leave the admin interface private, and publish only the project MCP endpoint through HTTPS when external clients need it.

```text
Administrators → private admin UI → MCPBox
AI clients      → HTTPS reverse proxy → project MCP endpoint
```

## Before you go live

1. Create a dedicated operating-system account for MCPBox. Do not run it as root or Administrator.
2. Create the first administrator and sign in.
3. Make the admin UI available only to administrators: keep it on `127.0.0.1`, use a VPN, or use an SSH tunnel.
4. Place a reverse proxy such as Nginx or Caddy in front of any public MCP endpoint and use a valid HTTPS certificate.
5. Open only required firewall ports. Usually this is port `443` on the reverse proxy; do not expose the raw admin port `38181` publicly.
6. Enable bearer authentication for every project that is reachable outside a trusted network.
7. Test one MCP call and check **Logs** before inviting users.

MCPBox enforces the admin session/token on both `/api/pro/*` and the shared project, server, log, and OAuth-connection administration routes. Revealing a complete OAuth client token requires `pro:admin`. Do not publish the admin port as part of the public MCP endpoint.

## A simple reverse-proxy idea

Use a public HTTPS name such as `https://mcp.example.com`. The proxy forwards requests to MCPBox on its internal MCP port. Keep the exact proxy configuration in your infrastructure repository; it depends on your network and certificate provider.

The important rules are simple: HTTPS outside, HTTP only inside the protected network if needed, and no public access to the Administration port.

## Security rules that matter

- Give each administrator their own account. Do not share an admin password or bootstrap token.
- Use SSO when your organization requires centrally managed access.
- Store passwords, project bearer tokens, API tokens, and RagBox/PolicyBox tokens in a secret manager where possible.
- Use the smallest permissions needed for each user, service account, and MCP server.
- Review every Marketplace package and MCP tool before enabling it. A tool may read, change, or send data.
- Use a separate project for each independent business area or data-access boundary.
- Keep the server, installed packages, and operating system patched.
- Review audit logs regularly and revoke unused sessions and tokens.

## Back up and restore

Before an update or a configuration change, make a protected backup of:

- the MCPBox database, normally `mcpbox.db` next to the executable;
- managed knowledge-base upload folders and local indexes;
- reverse-proxy and service configuration; and

Do not store backups in a public repository. Test restoring on a non-production server: stop MCPBox, restore the database and knowledge-base data together, then start it and verify a project, a server, and a knowledge-base search.

## Updating

1. Back up the items above.
2. Stop MCPBox.
3. Replace the executable with the approved release.
4. Start the service.
5. Check admin login, one MCP project, and the logs.

If something fails, stop the new executable, restore the previous executable and the matching database backup, then investigate in a test environment.

[Complete backup and recovery procedure](operations.md).

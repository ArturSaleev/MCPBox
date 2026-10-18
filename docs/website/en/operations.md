# Backup, upgrade and recovery

[Documentation](README.md) | [Русский](../ru/operations.md)

## Persistent installation

Keep a built `MCPBox` executable in a dedicated writable directory. Run it as a dedicated OS account and retain its environment in your service manager. Configure restart behavior and logs there. The repository does not provide a universal systemd, launchd, Windows service or Docker Compose installation.

Current source defaults both listeners to localhost. Published v1.3.0 binaries still bind MCP to all IPv4 interfaces unless explicitly configured. For remote administration, an SSH tunnel can keep the admin port private:

```bash
ssh -N -L 38181:127.0.0.1:38181 user@server
```

Open `http://127.0.0.1:38181/` locally. For externally reachable MCP endpoints, use HTTPS through your reverse proxy and configure project authentication. See [production security](production-security.md).

## Consistent backup

1. Record the executable version, listener settings, integration configuration and external storage locations.
2. Stop MCPBox with your service manager and confirm it has exited. Stop writers before copying SQLite.
3. Copy the whole installation/data directory to protected backup storage, including hidden directories: `mcpbox.db`, any remaining `mcpbox.db-wal` and `mcpbox.db-shm`, `package_store/`, `knowledge_base/`, and `.mcpboxpro/`.
4. Back up source folders outside the installation separately. Retain service environment/secrets and reverse-proxy configuration securely.
5. If PostgreSQL, external log storage or RagBox is used, take matching backups with those services' own tools. Copying SQLite alone does not back up these systems.
6. Restart MCPBox and verify login, projects, server health and knowledge search. Protect and periodically test the backup.

## Upgrade and migration

Stop the old service, take the backup above, then replace only the executable in the existing installation. Preserve data and environment. Never run two versions against the same SQLite database. Startup performs additive schema migrations; retain a matching old executable and pre-upgrade backup for rollback.

The former MCPBoxPro features are included in MCPBox without activation. Existing accounts and compatibility names remain. For an older installation without users, create the administrator as shown in [installation](installation.md). Old standalone MCPBox deployments using port `38180` must explicitly set `MCPBOX_PORT=38180` if that address must be retained. Two independent databases are not automatically merged.

## Restore and rollback

Stop the service. Restore the matching executable, entire data snapshot, external stores and configuration into a separate test installation first. Keep test listeners private and prevent unintended access to production integrations. Check ownership and permissions. Start only one process per data directory.

Verify login, project count/settings, one local server, one remote server if used, a client MCP tool call, and a representative knowledge search. Check storage runtime/shadow status and errors. Rebuildable indexes do not replace original documents: transient uploads may no longer have a retained original.

For rollback after an unsuccessful upgrade, stop the new executable and restore the previous binary together with its matching pre-upgrade data and external state. Replacing only the executable does not reverse database changes.

## Diagnose before changing data

`GET /healthz` confirms HTTP service health, not the health of every integration. Use Logs/Analytics and individual server checks. A `401` usually needs the correct credential; `403` needs the required scope or project authorization. A paused project, disabled server, missing executable or unavailable dependency needs its own fix. See [troubleshooting](troubleshooting.md).

Defaults above describe current source. Published v1.3.0 uses the earlier MCP bind `0.0.0.0`; explicitly set `--mcp-host 127.0.0.1` until updating. Enable direct remote access with `--mcp-host 0.0.0.0` or `MCPBOX_MCP_HOST=0.0.0.0`. [Network access and project isolation](network-access.md).

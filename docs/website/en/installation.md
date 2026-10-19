# Install and configure MCPBox

[Documentation](README.md) | [Русский](../ru/installation.md)

This guide covers a standard on-premises installation of MCPBox.

## Prerequisites

- A supported macOS, Linux, or Windows server
- Permission to run the application and open the selected TCP ports
- For optional SSO: an OIDC application registered with your identity provider
- For optional knowledge-base integration: a reachable RagBox instance

## 1. Download and verify the binary

Download the binary for your platform from the [official release](https://github.com/ArturSaleev/MCPBox/releases/tag/v1.3.0) into a dedicated writable directory. Rename it to `MCPBox` (macOS/Linux) or `MCPBox.exe` (Windows) after verifying its original filename.


The executable is named:

- macOS / Linux: `MCPBox`
- Windows: `MCPBox.exe`


Before running it, [verify SHA-256 and review OS security prompts](release-verification.md). Release binaries do not require Go or Node.js. Choose amd64 for Intel/AMD or arm64 for ARM; install additional runtimes only for connected MCP/CLI integrations that need them.

## 2. Open-source license

MCPBox uses Apache-2.0. No activation is required.

## 3. Create the first administrator

Run the following command from the application directory before the first startup:

```bash
./MCPBox admin create \
  --email admin@example.com \
  --password 'change-this-password' \
  --name 'System Administrator'
```

The password must contain at least eight characters. Store it in a secure password manager and replace the example value with a unique password.

On Windows PowerShell:

```powershell
.\MCPBox.exe admin create --email admin@example.com --password 'change-this-password' --name 'System Administrator'
```

## 4. Start MCPBox

Start the executable from its installation directory:

```bash
./MCPBox
```

Open the administration interface at [http://127.0.0.1:38181/](http://127.0.0.1:38181/) and sign in with the account created in the previous step.

For a server accessed remotely, use an SSH tunnel, reverse proxy, or change the admin listener host deliberately. Do not expose the administrative interface directly to the public internet.

## Listener configuration

MCPBox uses these defaults:

| Setting | Default | Purpose |
| --- | --- | --- |
| Admin host | `127.0.0.1` | Web administration UI and Admin API |
| Admin port | `38181` | Web administration UI and Admin API |
| MCP host | `127.0.0.1` | Project MCP listener |
| MCP port | `38182` | Project MCP listener |

For direct remote MCP access, explicitly override them using environment variables:

```bash
MCPBOX_HOST=127.0.0.1 \
MCPBOX_PORT=39181 \
MCPBOX_MCP_HOST=0.0.0.0 \
MCPBOX_MCP_PORT=39182 \
./MCPBox
```

The same values may also be passed as command-line flags:

```bash
./MCPBox \
  --host 127.0.0.1 --port 39181 \
  --mcp-host 0.0.0.0 --mcp-port 39182
```

The project MCP URL has the following form:

```text
http://<server>:<mcp-port>/mcp/<project-token>
```

Retrieve the project token from the MCPBox administration interface. In production, publish the endpoint through HTTPS and restrict network access as required by your security policy.

## Bootstrap token

You may configure an emergency/bootstrap token for the administration interface with `MCPBOX_PRO_BOOTSTRAP_TOKEN`:

```bash
MCPBOX_PRO_BOOTSTRAP_TOKEN='replace-with-a-long-random-secret' ./MCPBox
```

Use a long, unique secret and protect it as a credential. Prefer named local users or SSO for daily administration.

## Optional: OIDC single sign-on

MCPBox enables generic OIDC SSO when the following variables are set:

```bash
MCPBOX_PRO_OIDC_ISSUER_URL=https://id.example.com \
MCPBOX_PRO_OIDC_CLIENT_ID=your-client-id \
MCPBOX_PRO_OIDC_CLIENT_SECRET=your-client-secret \
MCPBOX_PRO_OIDC_REDIRECT_URL=https://mcpbox.example.com/api/pro/sso/callback \
./MCPBox
```

Optional settings:

- `MCPBOX_PRO_OIDC_PROVIDER_NAME`
- `MCPBOX_PRO_OIDC_SCOPES`
- `MCPBOX_PRO_OIDC_ALLOWED_HD`
- `MCPBOX_PRO_OIDC_DEFAULT_ROLE`
- `MCPBOX_PRO_OIDC_SESSION_DAYS`
- `MCPBOX_PRO_OIDC_AUTO_CREATE_USERS`

Register the redirect URL exactly as shown in your identity provider. For a local installation, the callback is normally `http://127.0.0.1:38181/api/pro/sso/callback`.

## Optional integrations

Configure integrations from the Administration interface after signing in.

- **RagBox** provides knowledge-base ingestion, vector connections, and retrieval. MCPBox checks the configured RagBox connection before treating it as active.

Keep service credentials separate and grant only the permissions each integration needs.

## Updating MCPBox

1. Back up the installation directory and database before updating.
2. Stop the running MCPBox process.
3. Replace the executable with the new version.
4. Start MCPBox and verify that the administration interface and required project endpoints work.

If the new release changes your deployment topology, update firewall rules, reverse-proxy configuration, and OIDC redirect URLs accordingly.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| The UI is unavailable | Confirm that the process is running and that `MCPBOX_HOST` / `MCPBOX_PORT` are correct. |
| Cannot sign in | Confirm the local admin credentials, or verify the OIDC variables and redirect URL. |
| MCP clients cannot connect | Check the MCP host/port, firewall, reverse proxy, project token, and project bearer/OAuth credentials. |

Defaults above describe current source. Published v1.3.0 uses the earlier MCP bind `0.0.0.0`; explicitly set `--mcp-host 127.0.0.1` until updating. Enable direct remote access with `--mcp-host 0.0.0.0` or `MCPBOX_MCP_HOST=0.0.0.0`. [Network access and project isolation](network-access.md).

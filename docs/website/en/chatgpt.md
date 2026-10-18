# Connect an MCPBox project to ChatGPT

[Documentation](README.md) | [Русский](../ru/chatgpt.md)

This guide lets ChatGPT use the tools that you configured inside one MCPBox project.

## Local clients: connect directly without Tailscale

When MCPBox and an installed AI client application run on the same computer, use the local project URL:

```text
http://127.0.0.1:38182/mcp/<project-token>
```

The project owner has verified this scenario with the **locally installed Qwen Studio application**. This verification does not cover the Qwen Studio web version. Configure the URL and the project's authentication method in the application's MCP settings.

On another computer, `127.0.0.1` refers to that computer itself: use a reachable network address of the MCPBox host and the appropriate listener/firewall settings. This is separate from the cloud-hosted ChatGPT connection described below.

## Before you start

You need:

- a working MCPBox project with at least one tested MCP server;
- a public HTTPS address for its MCP endpoint, for example `https://mcp.example.com/mcp/<project-token>`;
- project **Bearer authentication** enabled; and
- access to ChatGPT developer mode or custom apps in your ChatGPT plan/workspace.

ChatGPT connects to remote MCP servers, not directly to a private localhost or on-premises address. If your MCPBox server is private, use an approved secure tunnel or network design rather than opening the admin interface to the internet. OpenAI’s current guidance on developer mode and remote MCP apps is available in the [ChatGPT Help Center](https://help.openai.com/en/articles/12584461-developer-mode-and-full-mcp-connectors-in-chatgpt-beta).

## Tailscale Funnel: one way for cloud-hosted ChatGPT to reach local MCPBox

**Tailscale is not required to use MCPBox.** It is one way to make a local MCP endpoint reachable by cloud-hosted ChatGPT. If you already have a public HTTPS endpoint, use it directly. Local MCP clients do not need a tunnel.

For the “cloud-hosted ChatGPT → local MCPBox” scenario, we recommend **Tailscale Funnel**. It provides a public HTTPS URL and proxies requests to the MCP listener without a separate domain or reverse proxy setup.

1. Install Tailscale on the MCPBox machine and sign in to your tailnet. Funnel requires MagicDNS, HTTPS certificates and permission in your tailnet policy; the CLI may prompt you to enable the required settings on first use. See [Tailscale Funnel requirements](https://tailscale.com/docs/features/tailscale-funnel).
2. Start MCPBox and enable project **Bearer authentication**. Confirm that the MCP listener is running on local port `38182`.
3. On the same machine, run:

   ```bash
   tailscale funnel 38182
   ```

4. Copy the HTTPS address printed by the command and append the project path. For example, if the address is `https://my-machine.example.ts.net`, enter this in ChatGPT:

   ```text
   https://my-machine.example.ts.net/mcp/<project-token>
   ```

`38182` is the local MCPBox port, not the public HTTPS port: do not append `:38182` to the public URL. Substitute your configured MCP port if different. Publish the MCP listener on `38182` and keep administration on `38181` private.

Without `--bg`, keep the command running; for background operation use `tailscale funnel --bg 38182`. Check with `tailscale funnel status`. Disable this publication with `tailscale funnel 38182 off`. Both MCPBox and Tailscale must remain running.

Funnel exposes the address to the internet; MCPBox supplies authentication. Ordinary `tailscale serve` restricts access to the tailnet and does not replace Funnel for cloud-hosted ChatGPT. Once you have the URL, continue with OAuth setup below.

[Local-server publishing example](https://tailscale.com/docs/reference/examples/funnel) · [Commands and background mode](https://tailscale.com/docs/reference/tailscale-cli/funnel).

## Step 1: prepare the MCPBox project

1. Open the project in MCPBox.
2. Enable **Bearer authentication**.
3. Confirm that the project MCP endpoint is available through public HTTPS.
4. Keep the project token and OAuth client tokens private. Do not put them into a prompt or screenshot.

MCPBox includes OAuth discovery and authorization endpoints for this flow. Create a separate named OAuth client for each ChatGPT user or app. Each client has its own callback URL and token, and its name is shown as the actor in MCPBox **Logs**.
Protected-resource metadata is only advertised for a project with **Bearer authentication** enabled. A public MCP project does not advertise OAuth, so local clients such as LibreChat can connect without a false authentication prompt.

## Step 2: create the ChatGPT app

In ChatGPT, open **Settings → Apps → Create**. In a Business, Enterprise, or Edu workspace, an administrator may first need to enable developer mode and grant you permission.

Enter the public MCP endpoint URL, choose OAuth authentication if ChatGPT asks for it, and start the tool scan. ChatGPT will show the callback/redirect URL that it intends to use.

## Step 3: create an OAuth client in MCPBox

Keep the current ChatGPT setup open, then:

1. Open the target project in MCPBox.
2. Select the users icon next to the connection information.
3. Enter a unique client name and paste the exact callback URL shown by ChatGPT.
4. Create the client, reveal its token with the dedicated action, and copy it.
5. Return to the same ChatGPT setup: use the client name as the OAuth client ID and its token as the OAuth client secret.
6. Keep the public MCP endpoint as the resource/server URL and wait for tool scanning to finish.

The callback URL is stored in the database but is not displayed after creation. The token is also hidden until explicitly revealed. The table only shows the client name, status, creation date, and management actions.

If ChatGPT reports `redirect_uri_mismatch`, the current callback differs from the stored value. Delete and recreate that OAuth client with the callback from the current ChatGPT window. Do not restart the app-creation flow afterward, because a new flow may issue a different callback URL.

## Step 4: test safely

Create the app as a draft first. In a new ChatGPT conversation, select the app with `@` or from the tools menu and ask for a harmless read-only action. Check MCPBox **Logs** to confirm the request arrived. Only then allow more tools or publish the app to other workspace users.

## Safety checklist

- Connect only an MCPBox server that your organization owns and has reviewed.
- Start with read-only tools; review write tools carefully.
- Use a separate project for the ChatGPT connection if it needs a smaller tool set than internal users.
- Disable or delete the individual OAuth client if its token was exposed or an employee leaves.
- Review tools again after an MCP package or server is updated.

ChatGPT app features, plan availability, and screens can change. For the latest workspace rules, see [Apps in ChatGPT](https://help.openai.com/en/articles/11487775-connectors-in-chatgpt).

Funnel on the same machine works with `127.0.0.1:38182`; it does not require binding to `0.0.0.0`. See [network configuration](network-access.md).

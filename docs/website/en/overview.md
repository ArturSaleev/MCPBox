# MCPBox

[Documentation](README.md) | [Русский](../ru/overview.md)

MCPBox is the administrative control plane for AI projects and MCP integrations. It helps teams configure projects, manage access, publish tool connections, and operate enterprise integrations from one interface.

It is designed for private and on-premises deployments. MCPBox runs as a standalone service and includes a web-based Administration interface and project MCP endpoints.

## What MCPBox provides

- AI project and MCP connection management
- Administration: local users, roles, sessions, and access tokens
- A separate MCP endpoint for each project
- Optional single sign-on through any OpenID Connect (OIDC) provider
- Integrations with RagBox for knowledge bases and retrieval
- Integrations with PolicyBox for runtime identity and access validation
- Audit logs, analytics, and storage configuration

MCPBox stores project configuration and published tool manifests. It is part of the access-control architecture, but it does not replace authorization in the target business system: the system that owns the data or action must still make the final domain authorization decision.

## Downloads

Download the release package for your operating system from the [MCPBox download page](https://github.com/ArturSaleev/MCPBox).


Supported release targets include:

- macOS: Apple Silicon and Intel
- Linux: x86_64
- Windows: x86_64

## Documentation

- [Installation and configuration](./installation.md)
- [Open-source license](./license.md)
- [Projects, MCP servers, Marketplace, and prompts](./projects-marketplace-prompts.md)
- [Knowledge bases, logging, and integrations](./knowledge-integrations.md)
- [API overview](./api.md)
- [Run MCPBox safely in production](./production-security.md)
- [Troubleshooting](./troubleshooting.md)
- [Connect an MCPBox project to ChatGPT](./chatgpt.md)

## Default addresses

| Surface | Default address |
| --- | --- |
| Administration interface | `http://127.0.0.1:38181/` |
| Project MCP endpoint | `http://<server>:38182/mcp/<project-token>` |

The administration interface listens on localhost by default. The MCP listener binds to all network interfaces by default; expose it externally only behind the appropriate network controls, TLS termination, and authentication.

## Typical deployment flow

1. Download and unpack MCPBox on the server that will run it.
2. Create the first local administrator.
3. Start the service and sign in to the administration interface.
4. Create projects and configure the required MCP servers and optional integrations.

For detailed instructions, see [Installation and configuration](./installation.md).

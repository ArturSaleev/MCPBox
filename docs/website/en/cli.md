# CLI

[Documentation](README.md) | [Русский](../ru/cli.md)

The CLI registry exposes selected operations of local programs to a project through MCP. It is separate from Marketplace installation of MCP servers.

1. Install a trusted program and make it available to the MCPBox service account.
2. Create an entry on the CLI page or synchronize the catalog. Sync discovers binaries through PATH; it neither installs nor executes programs.
3. Review exact operations and allowed argument sets. Additional arguments are denied by default.
4. Set an existing project `root_path` and attach the entry in the CLI tab.
5. Verify `cli_list`, `cli_describe`, then an approved `cli_execute` call through an MCP client.

For example, allow Git `status` and `log`; for `log`, explicitly allow the complete argument vector `["-n", "5"]`. This does not allow arbitrary arguments or a shell string.

Programs run without a shell, in the project directory, with a restricted environment, timeout and output limits. Defaults are 30 seconds and 1 MiB combined stdout/stderr. This is an administrative allowlist, not an OS sandbox; an approved operation may modify files or use the network.

Disabling an entry or unlinking it blocks subsequent calls but does not cancel an already running process. CLI-only projects can also publish an MCP endpoint.

[Exact schema, API, execution limits and catalog](../../CLI.md).

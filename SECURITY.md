# Security policy

[Русский](SECURITY-ru.md)

## Supported versions

Security work targets the latest stable MCPBox release (currently v1.3.0) and the current development branch. Older 1.2.x releases and the former separate MCPBoxPro are not separate security-maintenance targets; upgrade using the [migration guide](docs/MERGE.md). Reports affecting an older version are still useful if the issue also affects current code.

Support is best effort. This policy does not promise a response deadline, a backport to every version, a bug bounty or an independent security audit.

## Report privately through GitHub

Do not publish suspected vulnerabilities, working exploits, tokens or affected users' data in public Issues, PRs or discussions.

Use **Security → Advisories → Report a vulnerability** in the [MCPBox repository](https://github.com/ArturSaleev/MCPBox/security/advisories). Reports submitted through GitHub Private Vulnerability Reporting are private to the reporter and authorized maintainers/collaborators.

**Setup status:** private vulnerability reporting was disabled when this policy was prepared. Adding `SECURITY.md` does not enable the GitHub feature. Until the owner enables it, the private report button may be absent. If so, open only a public administrative request titled **Enable private vulnerability reporting**, without any vulnerability details, affected systems, proof of concept or credentials. Wait for the private channel before sending those details. No email reporting address is used by this project.

See [GitHub's private reporting instructions](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/report-privately).

## Useful report contents

Privately include the affected release/commit, OS and architecture, relevant configuration with secrets removed, impact, prerequisites, minimal reproduction and any suggested fix. Use synthetic examples. Do not send production databases, real access tokens or unrelated personal data. Coordinate disclosure with the maintainer and allow investigation before publishing exploitation details.

Test only systems you own or are authorized to assess. Do not access another user's data, interrupt services, or conduct bulk scanning against installations you do not control.

## Deployment boundaries

Project URLs and credentials are secrets. Projects and approved CLI operations are not OS sandboxes. Use least-privilege OS/service accounts, enable authentication for externally accessible projects, keep the admin UI private and protect backups/logs. Read the [network isolation guide](docs/website/en/network-access.md) and [production guidance](docs/website/en/production-security.md).

Current source defaults MCP to loopback; published v1.3.0 binaries retain the earlier all-interface default unless explicitly configured. Do not assume documentation changes modify an already downloaded executable.

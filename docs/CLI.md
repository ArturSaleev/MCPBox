# Project CLI registry

MCPBox owns the models, GORM migration, storage, HTTP API, executor, MCP tools and UI in this repository.

User guide: [English](website/en/cli.md) / [Русский](website/ru/cli.md).

## Administration

The global **CLI** page registers a name, stable lowercase ID (`slug`), binary,
description, enabled state and exact operations (one per line). Multiword
operations such as `mcp list` are supported. No source change is needed to add
another trusted binary. A project connects active entries under
**MCP | CLI | RAG | Prompt**. Links are stored in `project_clis` with a composite
primary key. Project duplication copies the links; deleting a registry entry or
project removes its links. Disabling a CLI preserves links but revokes execution.
The additive tables are migrated on the next application startup.

Example registry payload:

```json
{
  "name": "Git",
  "slug": "git",
  "binary": "git",
  "description": "Git inspection",
  "is_enabled": true,
  "operations": [
    {"command": "status"},
    {"command": "diff"},
    {"command": "log", "argument_sets": [["-n", "5"], ["--oneline", "-n", "10"]]}
  ],
  "timeout_seconds": 30,
  "max_output_bytes": 1048576
}
```

Additional arguments are denied by default. An `argument_sets` entry allows
only that complete ordered vector; it is not a flag/prefix allowlist. Empty
arguments are allowed for each registered operation. In the UI the optional
argument-set JSON uses operation names as keys, e.g. `{"log":[["-n","5"]]}`.
Operation words are split on whitespace, without shell quoting/interpolation.
Use argument sets for literal arguments containing spaces.

## API

All endpoints use the existing administrative API authentication boundary.
Reads require `pro:read`; CLI mutations and administrative execution
require `pro:admin`. The project MCP endpoint retains its existing bearer/OAuth
and project-authorization behavior.

| Method | Path | Purpose |
|---|---|---|
| GET / POST | `/api/cli` | List (`{"items": [...]}`) / create entries |
| GET / PUT / DELETE | `/api/cli/{id}` | Read / replace configuration / delete |
| GET | `/api/cli/{id}/commands` | Allowed operations (`{"items": [...]}`) |
| GET | `/api/projects/{id}/cli` | Project links, including disabled entries |
| PUT / DELETE | `/api/projects/{id}/cli/{cliID}` | Idempotently connect / disconnect |
| POST | `/api/projects/{id}/cli/execute` | Execute an approved operation |

Execution request: `{"cli":"git","command":"log","args":["-n","5"]}`.
Response includes `stdout`, `stderr`, `exit_code`, `timed_out`, `truncated`.
Nonzero process exit codes are execution results, not HTTP errors. Permission
failures are 403; nonexistent numeric resources are 404.

## MCP

Only three compact tools are advertised, and only when the project has an
active CLI link:

- `cli_list`: project-visible CLI IDs, names and descriptions.
- `cli_describe`: exact operations/argument sets and limits for one allowed ID.
- `cli_execute`: the same request and executor as HTTP. Execution failures use
  MCP `isError`; successful results include text and `structuredContent`.

No per-operation tools or global operation lists are placed in tools/list.
When CLI tools are published, an upstream MCP tool using a reserved `cli_*`
name gets a server-prefixed alias; projects without active CLI preserve their
existing upstream aliases.
CLI-only projects are valid MCP connections. Permissions are reloaded on every
execution, so unlinking/disabling blocks subsequent calls in an existing session.

## Execution boundary and extension points

The executor rejects paused projects, missing/disabled/unlinked CLI entries,
unknown operations and unapproved argument vectors before starting a process.
It uses `exec.CommandContext(binary, argv...)`, never shell command strings.
Common shell binaries are rejected. Executable resolution uses the server PATH;
relative binary paths are rejected. The working directory is the project's
existing `root_path` when set (a missing path or a file is rejected). When empty,
each invocation gets a fresh private temporary directory, removed after execution.
This avoids inheriting MCPBox's working directory or the operator's home; it does
not restrict filesystem access. Set the project directory in the project editor
for Git, Go, Composer and other workspace operations. MCP server `working_dir`
and FilesSystemMCP paths are separate settings and do not populate `root_path`.
Requests cannot override cwd or environment. Child processes receive PATH and
basic locale/terminal settings, not MCPBox's secret-bearing server environment.
Stdin is closed; interactive operations are unsupported.

Only an executable named `ssh` (or `ssh.exe`, including an absolute path) also
receives the service account's HOME and SSH_AUTH_SOCK, when available. Use a trusted
absolute executable path if PATH can be modified. OpenSSH uses the service
account's SSH config, keys and known_hosts; on Unix it can also resolve the home
from the account database. MCPBox does not copy keys into its database or disable
host key checking. Configure the host alias with `BatchMode yes` in that account's
`~/.ssh/config`, provision known_hosts and keys/agent ahead of time, and ensure
permissions allow the MCPBox account to read them. Windows OpenSSH may resolve
its profile from the Windows account; verify configuration under the service
identity on that platform. Other CLI programs do not receive this SSH profile.

For example, register binary `ssh`, operation `mcpbox-linux`, and exact argument
sets `[["hostname"], ["docker ps"]]`. Then call
`{"cli":"ssh-linux","command":"mcpbox-linux","args":["docker ps"]}`.
The remote command remains a single local argument, but SSH passes it to the
remote shell: approve only exact trusted strings. `["sudo reboot"]` is denied
unless explicitly registered. No wildcard or local shell is introduced.

Defaults are 30 seconds and 1 MiB combined stdout/stderr. Configurable bounds are
1–300 seconds and 1 KiB–4 MiB. Output continues draining after the storage cap.
Unix cancels the process group on timeout/cancellation; Windows currently cancels
the direct child, not its full descendant tree. A short pipe-drain deadline keeps
inherited output pipes from hanging the request.

Registry/link changes and execution outcomes (including denial, timeout and
truncation) use the existing audit log. Arguments/output are omitted from audit
because they may contain secrets. An already running process is not revoked by a
registry edit; changes apply to subsequent executions.

This is an administrator-managed allowlist, not an OS sandbox or a guarantee of
read-only behavior. Registered programs and their approved operations must be
trusted: `npm`, `go`, `php`, `git`, Docker and similar tools can run scripts,
plugins or code, read files outside cwd, access networks or modify state.
Registering a generic interpreter/eval/script operation grants its capabilities.
Use absolute binary paths when executable identity must be stable. No commands
are enabled or preinstalled automatically.

Per-operation policy lives in `CLIOperation`; future argument validators,
read/write classification and confirmation policies belong there. Enforce those
policies in the shared executor before process launch; do not expose a generic
shell fallback. Strong filesystem isolation, confirmation flows, execution
concurrency quotas and Windows process-tree isolation are future extensions.

## Starter catalog and binary discovery

The default catalog URL is **https://mcpbox.sh/cli.json**. Upload the supplied
`internal/cli/cli.json` there; this exact file is also embedded as the fallback.
It is separate from the Marketplace `catalog.json`. Extend `items`, use unique
stable `slug` IDs, and update `version`; `schema_version` remains `1`. Supported
`platforms` values are `linux`, `darwin`, `windows`. Catalog entries use PATH
binary names; configuration validation rejects shell binaries and invalid data.
The download is limited to 1 MiB with a five-second timeout. Network, HTTP and
schema errors use the built-in starter catalog without blocking startup.

On application startup, a background check imports only entries supported by the
server OS whose binaries are found through Go's `exec.LookPath`. This is a
filesystem/PATH check: it does not run commands, install executables or probe
services. For example, finding Docker does not establish that its daemon works.
No project links are created. Missing/unsupported entries remain visible in the
catalog availability list, and can be checked again after the user installs a
program. Registered entries also display when their binary is no longer found.

Automatic sync preserves all existing configuration (including operations,
user descriptions and enabled state). Deleting any CLI remembers its stable ID
in `cli_catalog_dismissals`, so automatic checks do not revive it. Explicit
manual creation or catalog restoration clears that dismissal. A fresh database
receives only binaries available on that machine.

The CLI page provides **Check and add found binaries** and an expandable list
with availability and differences. Existing entries can be updated only through
an explicit **Review / Apply changes** flow showing current and proposed
configuration. Applying updates preserves disabled state and project links;
review carefully because newly approved operations become usable by connected
projects. The server rechecks binary/OS availability and verifies the reviewed
catalog entry's hash before applying, rejecting a changed remote entry.

Additional administrative endpoints:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/cli/catalog` | Availability, differences, catalog source/version |
| POST | `/api/cli/catalog/sync` | Add absent available entries only |
| POST | `/api/cli/catalog/{slug}/apply` | Apply reviewed entry with `{"revision":"..."}` |

Pro catalog writes require `pro:admin`, like other CLI mutations. Catalog
administration and registry logic remain shared in MCPBox; Pro adds no duplicate
implementation. Imported definitions do not install Git, Docker, npm or any
other software, and binary presence does not imply an operation is harmless.

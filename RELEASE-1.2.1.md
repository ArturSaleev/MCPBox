# MCPBox 1.2.1

## Release Notes

MCPBox 1.2.1 is a large product polish release focused on the Market experience, package lifecycle, safer secret handling, project duplication, and built-in operational visibility on the logs screen.

### Highlights

- Market refactor and cleaner package UX
  - The Market UI is no longer a large monolithic block inside `App.tsx`.
  - Catalog source can now be synchronized from a remote URL or from a local JSON file.
  - Installed packages now support safer lifecycle controls, including uninstall protection while a package is still used by one or more projects.
  - `Add to project` is shown only after a package has actually been installed.

- Richer catalog manifest contract
  - Catalog items now support `icon_url`, `system_dependencies`, `default_env`, `env_schema`, `secret: true`, `env_var`, and `health_check`.
  - `default_env` can be provided as either an object or an array of `{ key, value }` pairs.
  - The default catalog source has been migrated from `webeasy.kz` to [mcpbox.sh](https://mcpbox.sh).

- Safer secret handling
  - Sensitive install-time values can now move into environment variables instead of remaining in visible CLI arguments.
  - The project UI masks secret values in launch commands and secret-like env/header values where possible.
  - Existing manually configured servers that still pass secrets directly in command arguments should still be migrated to env variables for full process-list safety.

- Docker runtime MVP
  - Catalog items can now declare `runtime.type: "docker"` with `install.strategy: "docker_pull"`.
  - This first version is intentionally optimized for stdio-oriented container-backed MCP servers.

- Better project lifecycle
  - Projects can now be duplicated with a new name.
  - Duplication clones servers, package links, installed integrations, and connected knowledge bases while generating a fresh project token.
  - The project form was simplified by removing the root path field from the create/edit modal.

- Built-in performance monitoring on the logs screen
  - The existing logs page now also shows latency, error, and traffic summaries.
  - It includes compact trend charts, top slow servers, top error servers, top traffic servers, and recent failures.
  - This is a lightweight embedded observability layer, not an external metrics platform.

### What Changed

- Refactored the Market page into dedicated frontend modules
- Added catalog sync from remote URL or local uploaded manifest
- Added package install, uninstall, and add-to-project lifecycle handling
- Added manifest parsing and storage for `icon_url`, `system_dependencies`, `default_env`, `env_schema`, `secret`, `env_var`, and `health_check`
- Added system dependency verification before package install
- Added post-install integration verification via manifest-driven health checks
- Added secret masking and env-based secret propagation for managed catalog installs
- Added Docker runtime MVP with `docker_pull`
- Added Python runtime fallback so `python3` works on systems without `python`
- Added project duplication API and UI flow
- Added built-in performance metric storage and `/api/logs/metrics`
- Reworked the logs screen into a combined audit + performance operations view
- Added audit log entries for catalog sync failures and package install/add failures
- Added MySQL catalog example based on `@benborla29/mcp-server-mysql`

### Upgrade Notes

- Older catalog manifests remain compatible, but `default_env` now also supports object syntax.
- If an older saved catalog source still points to `https://webeasy.kz/mcpbox/catalog.json`, MCPBox normalizes it to `https://mcpbox.sh/catalog.json`.
- Docker support in `1.2.1` is intentionally an MVP. Advanced features such as compose-style orchestration, custom networks, and volume presets are not fully implemented yet.
- Performance metrics are stored locally and used by the built-in logs UI. They are not intended to replace external observability stacks.

## Short Release Text

MCPBox 1.2.1 brings a major Market upgrade with package lifecycle controls, local-file catalog sync, richer manifest support, secret-safe env handling, Docker runtime MVP, project duplication, and built-in performance monitoring directly on the logs screen.

## Russian Release Text

MCPBox 1.2.1 приносит крупное обновление Market: lifecycle управления пакетами, sync каталога из локального JSON, расширенный manifest contract, более безопасную работу с секретами через env, Docker runtime MVP, дублирование проектов и встроенный performance monitoring прямо на экране логов.

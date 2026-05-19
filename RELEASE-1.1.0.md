# MCPBox 1.1.0

## Release Notes

MCPBox 1.1.0 is focused on local developer workflow, project startup behavior, and the introduction of the new Market experience.

### Highlights

- Embedded Ollama chat inside MCPBox
  - `mcphost` no longer needs to be installed separately for local Ollama testing.
  - MCPBox now starts an in-app Ollama chat flow through an embedded SDK integration.
  - Users can launch local MCP testing from the project view with model selection from installed Ollama models.

- Better project startup behavior
  - Project startup now launches all enabled local `STDIO` servers for projects that use auto-start, instead of only the first server in the list.

- Cleaner server logs
  - Known informational messages from the Filesystem MCP server are no longer surfaced as misleading errors.
  - Graceful shutdown signals no longer produce noisy runner shutdown errors during normal exit.

- New Market page and catalog flow
  - MCPBox now includes a dedicated Market page for catalog-based integrations.
  - Catalog sync action was compacted into an icon button with tooltip.
  - Catalog cards now display in a single-column layout for easier scanning.
  - Documentation and website links were moved into the card header.
  - Install actions were moved into the header area to reduce card height and improve visual structure.

### What Changed

- Added embedded Ollama host flow powered by `github.com/mark3labs/mcphost/sdk`
- Added Ollama model detection and selection in the UI
- Added internal `ollama-chat` subcommand to MCPBox
- Improved startup orchestration for multi-server projects
- Reduced noisy stderr reporting for Filesystem MCP
- Added the new Market page and refined its layout and controls

### Upgrade Notes

- MCPBox now targets Go `1.26.0` because the embedded `mcphost` SDK requires it.
- Existing project/server data remains compatible.
- Updated release binaries should be rebuilt before publishing.

## Short Release Text

MCPBox 1.1.0 adds embedded Ollama support for one-click local MCP testing, introduces the new Market page for catalog-based integrations, improves project auto-start so all enabled local servers come up correctly, and reduces noisy Filesystem server logs.

## Russian Release Text

MCPBox 1.1.0 добавляет встроенную поддержку Ollama для локального MCP-тестирования в один клик, вводит новую страницу Market для каталоговых интеграций, исправляет автозапуск проектов так, чтобы поднимались все включённые локальные серверы, и убирает шумные ложные ошибки от Filesystem MCP.

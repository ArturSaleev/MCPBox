# MCPBox 1.1.1

## Release Notes

MCPBox 1.1.1 is a focused patch release for the embedded Ollama launcher on Windows.

### Fixed

- Fixed a Windows-specific launch bug where `Launch Ollama` could open a terminal but fail to start the embedded `ollama-chat` flow.
- Fixed command construction for Windows so paths with spaces and nested quotes are handled safely through PowerShell encoded execution.
- Fixed terminal startup on Windows so the launched session uses the project working directory instead of trying to prepend a Unix-style `cd`.
- Improved Ollama startup probing on Windows: MCPBox now checks whether `ollama` is already available, starts `ollama serve` in the background when needed, waits briefly, and then launches chat.

### What Changed

- Added a Windows-specific PowerShell launch path for embedded Ollama sessions
- Switched Windows terminal startup to `Start-Process` with explicit `WorkingDirectory`
- Added UTF-16 Base64 PowerShell command encoding to avoid escaping issues in file paths and arguments
- Added audit logging for prepared Ollama launch commands to simplify diagnostics

### Impact

- `Launch Ollama` now works reliably on Windows setups where the previous `cmd /k` flow failed
- Projects located in paths with spaces should launch more consistently
- The fix does not change Linux or macOS launch behavior

## Short Release Text

MCPBox 1.1.1 fixes the Windows Ollama launcher. The embedded `Launch Ollama` flow now starts through a Windows-specific PowerShell path, preserves the correct working directory, handles quoted paths safely, and starts `ollama serve` automatically when needed.

## Russian Release Text

MCPBox 1.1.1 исправляет запуск Ollama на Windows. Встроенная кнопка `Launch Ollama` теперь использует отдельный PowerShell-сценарий для Windows, корректно сохраняет рабочую директорию проекта, безопасно обрабатывает пути с пробелами и при необходимости автоматически поднимает `ollama serve`.

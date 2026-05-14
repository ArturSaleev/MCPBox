# MCPBox Developer Guide

Русская документация. English version: [DEVELOPER.md](./DEVELOPER.md)

## Обзор

MCPBox — это Go-based control plane для MCP-серверов.

Он группирует MCP-серверы по проектам, хранит конфигурацию в SQLite, запускает локальные `STDIO` MCP-серверы, проксирует удалённые `HTTP streaming` MCP-серверы, отдаёт встроенный React UI и даёт один MCP URL на проект.

На текущем этапе MCPBox — это:
- control plane для MCP-серверов;
- project-based организатор локальных и удалённых MCP endpoint-ов;
- единый MCP endpoint на проект;
- операторский UI для жизненного цикла проектов, серверов, логирования и inspection.

На текущем этапе MCPBox ещё не:
- multi-user платформа с аутентификацией и ролями;
- полноценный observability stack;
- умный multi-server MCP router, который объединяет несколько backends в одну сессию.

## Основная Модель

- `Project`: логическая workspace-группа MCP-серверов для одного клиента, команды или окружения.
- `MCP Server`: либо локальный `STDIO` сервер, который запускает MCPBox, либо удалённый `HTTP streaming` сервер.
- `Primary Server`: сервер внутри проекта, через который работает MCP endpoint проекта.

Важное поведение:
- у каждого проекта ровно один MCP URL;
- этот MCP URL всегда использует явно выбранный primary server;
- если primary server не выбран, MCP endpoint не готов;
- если проект поставлен на паузу, MCP-подключения блокируются;
- если сервер отключён, его нельзя запускать и использовать как primary.

## Текущие Возможности

### Backend

- Go HTTP API
- SQLite через GORM
- orchestration локальных `STDIO` MCP-процессов
- proxy для удалённых `HTTP streaming` серверов
- project-level `/mcp/{project_token}` endpoint
- synchronous JSON-RPC request/response bridge для HTTP MCP-клиентов
- legacy SSE compatibility mode для старых MCP-клиентов
- явный выбор primary server
- audit logging для управляющих действий и MCP-трафика
- pause/resume проекта
- enable/disable сервера
- inspection локального `STDIO` сервера

### Frontend

- встроенный React admin UI
- список проектов и project overview
- модальное создание проекта
- модальное добавление сервера для `STDIO` и `HTTP streaming`
- показ project MCP URL
- start/stop локальных серверов
- выбор primary server
- audit log console с фильтрацией
- автообновляемая страница логов
- `Info` modal для `STDIO` серверов
- локализация English/Russian

## Transport Модель

Основной endpoint проекта:

```http
GET /mcp/{project_token}
POST /mcp/{project_token}
```

Есть и backward-compatible alias:

```http
GET /connect/{project_token}
POST /connect/{project_token}
```

Текущее поведение transport-слоя:
- основной режим: `POST /mcp/{project_token}` без `sessionId` работает как synchronous HTTP JSON-RPC;
- legacy режим: `GET /mcp/{project_token}` открывает SSE stream и возвращает `endpoint` event;
- дальнейшие legacy запросы идут через `POST /mcp/{project_token}?sessionId=...`;
- remote `HTTP streaming` primary servers проксируются в upstream;
- paused project и disabled primary server блокируются ещё до открытия transport-а.

Почему это важно:
- LM Studio ожидает synchronous HTTP request/response для вызовов вроде `initialize` и `tools/list`;
- старые SSE-ориентированные MCP-клиенты всё ещё используют legacy session flow;
- MCPBox поддерживает оба сценария без разделения URL проекта.

## Inspection Для `STDIO`

Для локальных `STDIO` серверов MCPBox умеет делать live inspection MCP-возможностей.

Кнопка `Info` в UI доступна только для `STDIO` серверов. Она может показать:
- server metadata из `initialize`;
- negotiated MCP capabilities;
- доступные `tools`;
- доступные `resources`;
- доступные `prompts`;
- соседний `README.md`, если MCPBox находит его рядом с локальным путём сервера.

Для удалённых `HTTP streaming` серверов эта кнопка специально не показывается.

## Логирование И Контроль

В MCPBox уже есть audit trail для операционного контроля.

Сейчас логируются, например:
- создание проекта;
- создание сервера;
- смена primary server;
- pause/resume проекта;
- start/stop сервера;
- enable/disable сервера;
- попытки MCP-подключения;
- forwarded JSON-RPC payloads.

SQL-логирование GORM по умолчанию выключено, чтобы обычный MCP-трафик не засорял консоль.

## Требования

- Go `1.25+`
- Node.js и npm для сборки встроенного UI
- Windows, Linux или macOS

Внешняя БД не нужна. MCPBox по умолчанию создаёт локальный SQLite-файл.

## Структура Проекта

```text
main.go                      Точка входа приложения
internal/models              GORM-модели
internal/storage             SQLite-хранилище и запросы
internal/orchestrator        Жизненный цикл MCP-процессов, inspection, stdio bridge
internal/httpapi             HTTP API, MCP endpoint, встроенный UI
html                         React + Vite исходники встроенной админки
```

## Сборка

Сначала нужно собрать встроенный UI, затем Go-бинарь:

```bash
npm --prefix html install
npm --prefix html run build
go build -o MCPBox .
```

Для Windows:

```powershell
npm --prefix html install
npm --prefix html run build
go build -o MCPBox.exe .
```

Результат сборки frontend-а пишется в `internal/httpapi/ui/dist` и затем встраивается в Go-приложение.

## Запуск

Порт по умолчанию: `38180`

Запуск из исходников:

```bash
go run .
```

Для Windows:

```powershell
.\MCPBox.exe
```

После успешного старта MCPBox автоматически открывает локальный UI в браузере и печатает:
- `http://127.0.0.1:<port>/`
- локальные IPv4 URL, например `http://192.168.x.x:<port>/`

## Настройка Порта

Порядок приоритета:
1. CLI-флаг `-port`
2. Переменная окружения `MCPBOX_PORT`
3. Значение по умолчанию `38180`

Примеры:

```bash
go run . -port 39000
```

```powershell
$env:MCPBOX_PORT=39000
.\MCPBox.exe
```

## Локальные Данные

По умолчанию MCPBox создаёт:

```text
mcpbox.db
```

Это локальные runtime-данные, их не нужно коммитить в git.

## Подключение Клиентов

Типичный сценарий:

1. Запустить MCPBox.
2. Создать проект в UI.
3. Добавить в проект хотя бы один MCP-сервер.
4. Назначить primary server.
5. Скопировать project MCP URL из UI.

Пример MCP URL:

```text
http://127.0.0.1:38180/mcp/<project_token>
```

Важно:
- у проекта должен быть выбран primary server, иначе endpoint не готов;
- если проект поставлен на паузу, доступ клиента будет заблокирован;
- если primary server отключён, MCP path не сработает;
- `/mcp/{project_token}` — основной endpoint;
- `/connect/{project_token}` — только backward-compatible alias.

### Codex

Пример через CLI:

```bash
codex mcp add mcpbox --url http://127.0.0.1:38180/mcp/<project_token>
```

Пример прямой конфигурации:

```toml
[mcp_servers.mcpbox]
url = "http://127.0.0.1:38180/mcp/<project_token>"
```

### Claude Code

Пример через CLI:

```bash
claude mcp add --transport sse mcpbox http://127.0.0.1:38180/mcp/<project_token>
```

Пример project config:

```json
{
  "mcpServers": {
    "mcpbox": {
      "type": "sse",
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

### LM Studio

LM Studio работает с MCPBox через основной HTTP MCP URL.

Пример конфигурации:

```json
{
  "mcpServers": {
    "mcpbox": {
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

Важные замечания для LM Studio:
- `Authorization` header не нужен, если только MCPBox не стоит за вашей отдельной auth-прослойкой;
- `POST /mcp/{project_token}` возвращает реальный JSON-RPC result синхронно;
- именно это позволяет LM Studio корректно выполнять `initialize`, `tools/list` и похожие вызовы.

### Универсальные MCP-Клиенты

Для клиентов, которые принимают JSON-конфигурацию, типичный шаблон такой:

```json
{
  "mcpServers": {
    "mcpbox": {
      "url": "http://127.0.0.1:38180/mcp/<project_token>"
    }
  }
}
```

## HTTP API

### Health

```http
GET /healthz
```

### Проекты

Создание проекта:

```http
POST /api/projects
Content-Type: application/json
```

```json
{
  "name": "My Project",
  "description": "My MCP environment"
}
```

Список проектов:

```http
GET /api/projects
```

Статус проекта:

```http
GET /api/projects/{id}/status
```

Назначение primary server:

```http
POST /api/projects/{id}/primary-server
Content-Type: application/json
```

```json
{
  "server_id": 2
}
```

Поставить проект на паузу:

```http
POST /api/projects/{id}/pause
```

Возобновить проект:

```http
POST /api/projects/{id}/resume
```

### MCP-Серверы

Добавление `STDIO` сервера:

```http
POST /api/projects/{id}/servers
Content-Type: application/json
```

```json
{
  "name": "Filesystem Server",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"],
  "env_vars": [],
  "env_passthrough": ["OPENAI_API_KEY"],
  "working_dir": "",
  "auto_start": true
}
```

Добавление удалённого `HTTP streaming` сервера:

```json
{
  "name": "Remote MCP",
  "transport": "http_stream",
  "url": "https://mcp.example.com/mcp",
  "bearer_token_env_var": "MCP_BEARER_TOKEN",
  "headers": [],
  "header_env_vars": []
}
```

Запуск сервера:

```http
POST /api/servers/{id}/start
```

Остановка сервера:

```http
POST /api/servers/{id}/stop
```

Отключение сервера:

```http
POST /api/servers/{id}/disable
```

Включение сервера:

```http
POST /api/servers/{id}/enable
```

Inspection локального `STDIO` сервера:

```http
GET /api/servers/{id}/inspect
```

### Логи

Получить audit logs:

```http
GET /api/logs
```

Фильтрация логов по проекту:

```http
GET /api/logs?project_id={id}
```

## Текущие Ограничения

На этом этапе в MCPBox ещё нет:
- аутентификации и авторизации;
- управления пользователями и ролями;
- автоматической multi-server routing логики внутри одного проекта;
- продвинутых metrics dashboards;
- исторической аналитики сверх audit log;
- service installers для фонового запуска на уровне ОС.

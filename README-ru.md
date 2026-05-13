# MCPBox

Русская документация. English version: [README.md](./README.md)

MCPBox — это Go-based control center для управления MCP-серверами в одном месте.

Он группирует серверы по проектам, хранит конфигурацию в SQLite, запускает локальные `STDIO` MCP-серверы, проксирует project-level MCP подключения, отдаёт встроенный web UI, ведёт аудит действий и запросов, а также позволяет быстро приостанавливать проекты или отключать отдельные серверы.

## Что Такое MCPBox

MCPBox на текущем этапе — это не просто сырой MCP proxy.

Сейчас это:
- control plane для MCP-серверов;
- project-based органайзер локальных и удалённых MCP endpoint-ов;
- единый connect endpoint на проект;
- операторский UI для жизненного цикла проектов, серверов, логирования и inspection.

Сейчас это ещё не:
- multi-user платформа с аутентификацией и ролями;
- полноценный observability stack;
- умный multi-server MCP router, который объединяет несколько backends в одну сессию.

## Основная Модель

Главные сущности:

- `Project`: логическая workspace-группа MCP-серверов для одного клиента, команды или окружения.
- `MCP Server`: либо локальный `STDIO` сервер, который запускает MCPBox, либо удалённый `HTTP streaming` сервер.
- `Primary Server`: единственный сервер внутри проекта, через который работает `/connect/{token}`.

Важное поведение:
- у каждого проекта один connect URL;
- этот connect URL всегда использует явно выбранный primary server;
- если primary server не выбран, connect endpoint не готов;
- если проект поставлен на паузу, MCP-подключения блокируются;
- если сервер отключён, его нельзя запускать и использовать как primary.

## Текущие Возможности

### Backend

- Go HTTP API
- SQLite через GORM
- orchestration локальных `STDIO` MCP-процессов
- поддержка proxy для удалённых `HTTP streaming` серверов
- project-level `/connect/{project_token}` endpoint
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
- показ project connect URL
- start/stop для локальных серверов
- выбор primary server
- audit log console с фильтрацией
- сводка по самым активным проектам и серверам
- `Info` modal для `STDIO` серверов
- локализация English/Russian
- тема, следующая системному light/dark режиму

## Inspection Для `STDIO`

Для локальных `STDIO` серверов MCPBox умеет делать live inspection MCP-возможностей.

Кнопка `Info` в UI доступна только для `STDIO` серверов. Она открывает модалку, где можно увидеть:
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
- попытки подключения;
- forwarded JSON-RPC payloads.

Это нужно для того, чтобы оператор понимал, кто использует MCP surface, и мог быстро остановить проект или сервер при подозрительном поведении.

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
internal/httpapi             HTTP API, connect endpoint, встроенный UI
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

Запуск собранного бинаря:

```bash
./MCPBox
```

Для Windows:

```powershell
.\MCPBox.exe
```

После успешного старта MCPBox автоматически открывает локальный UI в браузере.

## Настройка Порта

Порядок приоритета:
1. CLI-флаг `-port`
2. Переменная окружения `MCPBOX_PORT`
3. Значение по умолчанию `38180`

Примеры:

```bash
go run . -port 39000
```

```bash
MCPBOX_PORT=39000 go run .
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

## Обзор UI

Во встроенном UI сейчас есть два основных режима:

- `Projects`: создание проектов, добавление серверов, выбор primary server, start/stop локальных серверов, pause проекта, disable сервера, inspection локальных `STDIO` серверов.
- `Logs`: компактная audit console, фильтр по текущему проекту и сводка по активности самых популярных проектов и серверов.

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

### MCP-серверы

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

## Connect Endpoint

У каждого проекта есть свой token и connect URL:

```http
GET /connect/{project_token}
POST /connect/{project_token}
```

Поведение:
- endpoint всегда маршрутизируется через выбранный primary server проекта;
- для локальных `STDIO` серверов `POST` отправляет JSON-RPC в `stdin` процесса;
- для локальных `STDIO` серверов `GET` стримит `stdout` кадры через SSE;
- для удалённых `HTTP streaming` серверов MCPBox проксирует запрос на upstream;
- если проект на паузе, доступ блокируется;
- если primary server отсутствует или отключён, connect path не готов.

Пример JSON-RPC запроса:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

## Development Workflow

Только frontend:

```bash
cd html
npm install
npm run dev
```

Полная проверка:

```bash
npm --prefix html run build
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go build ./...
```

## Текущие Ограничения

На этом этапе в MCPBox ещё нет:
- аутентификации и авторизации;
- управления пользователями и ролями;
- автоматической multi-server routing логики внутри одного проекта;
- продвинутых metrics dashboards;
- исторической аналитики сверх audit log;
- service installers для фонового запуска на уровне ОС.

## Что Уже Можно Считать Достаточным

Для текущего этапа проект уже закрывает основную ценность MCPBox:
- конфигурация проектов;
- подключение локальных и удалённых MCP-серверов;
- выбор primary server;
- inspection локальных `STDIO` возможностей;
- мониторинг использования через логи;
- быстрое ограничение риска через pause проекта или disable сервера.

То есть как MVP для company-operated MCP control center проект уже выглядит цельным.

Дальнейшие шаги — это уже улучшения, а не отсутствие фундамента.

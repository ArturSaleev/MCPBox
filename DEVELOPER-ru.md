# MCPBox Developer Guide
## Market / Catalog Update

Первый этап миграции на модель `Market / Integrations + Custom Servers` уже внедрён.

- Основной сценарий для remote integrations теперь начинается с внешнего JSON catalog manifest.
- MCPBox умеет сделать `POST /api/catalog/sync`, сохранить catalog items в SQLite и показать их во вкладке `Market`.
- Установка элемента каталога в проект выполняется через `POST /api/projects/{id}/integrations`.
- При установке создаются `Installed Integration` и связанный обычный `MCPServer`, поэтому `/mcp/{project_token}` не меняется.
- Custom server flows для `STDIO` и manual `HTTP streaming` остаются рабочими.
- OAuth не удалён из backend, но больше не является главным UX для добавления remote integration.

Новые API первого этапа:

```http
GET /api/catalog/items
GET /api/catalog/items?enabled_only=1
POST /api/catalog/sync
POST /api/projects/{id}/integrations
```

Базовая форма manifest:

```json
{
  "schema_version": "2026-05-18",
  "generated_at": "2026-05-18T10:00:00Z",
  "items": [
    {
      "id": "notion",
      "name": "Notion MCP",
      "category": "productivity",
      "description": "Remote Notion integration",
      "transport": "http_stream",
      "mcp_url": "https://api.example.com/mcp/notion",
      "auth_type": "none",
      "auth_provider": "",
      "config_schema": {},
      "capabilities": ["tools", "resources"],
      "tags": ["docs", "notes"],
      "website": "https://example.com",
      "docs_url": "https://example.com/docs",
      "enabled": true,
      "version": "1.0.0"
    }
  ]
}
```

Русская документация. English version: [DEVELOPER.md](./DEVELOPER.md)

## Обзор

MCPBox — это control plane на Go для MCP-серверов.

Он группирует MCP-серверы по проектам, хранит конфигурацию в SQLite, запускает локальные `STDIO` MCP-серверы, проксирует удалённые `HTTP streaming` MCP-серверы, отдаёт встроенный React UI и предоставляет один MCP URL на проект.

Сейчас MCPBox — это:
- control plane для MCP-серверов;
- проектный организатор локальных и удалённых MCP endpoint-ов;
- единая точка входа MCP на проект;
- операторский UI для управления проектами, серверами, логами, health-check и inspection.

Сейчас MCPBox ещё не:
- multi-user платформой с аутентификацией и ролями;
- полноценным observability stack;
- интеллектуальным multi-server router, который объединяет несколько backend-ов в одну сессию.

## Основная модель

- `Project`: логическая workspace-группа MCP-серверов для клиента, команды или окружения.
- `MCP Server`: либо локальный `STDIO` сервер, который запускает MCPBox, либо удалённый `HTTP streaming` сервер.
- `Primary Server`: сервер внутри проекта, через который работает MCP endpoint проекта.

Важное поведение:
- у каждого проекта ровно один MCP URL;
- этот MCP URL всегда использует явно выбранный primary server;
- если primary server не выбран, MCP endpoint не готов;
- если проект поставлен на паузу, MCP-подключения блокируются;
- если сервер отключён, его нельзя запускать или использовать как primary.

## Текущие возможности

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
- health-проверка сервера при create, update, start и manual check
- OAuth 2 flow для remote MCP servers

### Frontend

- встроенный React admin UI
- список проектов и project overview
- модальные окна создания проекта и добавления сервера
- поддержка `STDIO` и `HTTP streaming`
- отображение project MCP URL
- start/stop локальных серверов
- выбор primary server
- ручной `Check` для health-проверки
- OAuth connect/disconnect для remote MCP servers
- audit log console с фильтрацией
- автообновляемая страница логов
- `Info` modal для `STDIO` серверов
- локализация English/Russian

## Transport-модель

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
- MCPBox поддерживает оба сценария без разделения project URL.

## Inspection для `STDIO`

Для локальных `STDIO` серверов MCPBox умеет делать live inspection MCP-возможностей.

Кнопка `Info` в UI доступна только для `STDIO` серверов. Она может показать:
- server metadata из `initialize`;
- negotiated MCP capabilities;
- доступные `tools`;
- доступные `resources`;
- доступные `prompts`;
- соседний `README.md`, если MCPBox находит его рядом с локальным путём сервера.

Для удалённых `HTTP streaming` серверов эта кнопка специально не показывается.

## Health-проверка серверов

MCPBox проверяет работоспособность MCP-сервера до того, как проблема всплывёт уже внутри AI-клиента.

Текущее поведение:
- при создании сервера MCPBox сразу выполняет health-check и сохраняет результат;
- при редактировании сервера MCPBox повторно проверяет обновлённую конфигурацию;
- ручной запуск локального `STDIO` сервера считается успешным только если MCP health-check реально прошёл;
- UI показывает последнее health-состояние, текст ошибки и время проверки;
- для локальных и удалённых серверов есть отдельное действие `Check`.

Текущая стратегия проверки:
- локальные `STDIO` серверы проверяются через настоящий MCP handshake: `initialize`, `notifications/initialized`, а также `tools/list`, `resources/list` и `prompts/list`, если сервер их поддерживает;
- удалённые `HTTP streaming` серверы проверяются через HTTP-запрос `initialize` к настроенному MCP URL;
- результат сохраняется в SQLite и возвращается через API.

## OAuth для удалённых MCP-серверов

MCPBox умеет работать как OAuth 2 client для remote `HTTP streaming` MCP servers.

Текущее поведение:
- OAuth настраивается на уровне конкретного remote MCP server;
- MCPBox открывает системный браузер для логина и consent;
- callback принимает сам MCPBox по `GET /oauth/callback`;
- access token и refresh token хранятся локально в SQLite;
- при proxy-запросах MCPBox автоматически подставляет bearer token в upstream;
- access token обновляется по refresh token, когда срок действия подходит к концу;
- при изменении OAuth-конфига старые токены очищаются специально.

Почему не WebView:
- Figma прямо требует обычный браузер и не поддерживает embedded WebView для OAuth;
- browser-based flow лучше работает с MFA, SSO, passkeys и корпоративной авторизацией.

Что важно для провайдеров вроде Figma:
- MCPBox не убирает необходимость создать свой OAuth app у провайдера;
- redirect URL должен указывать обратно на MCPBox, например `http://127.0.0.1:38180/oauth/callback`;
- MCPBox выступает OAuth client-ом, а не просто пробрасывает URL.

Типовой сценарий для Figma:
1. Создать remote server с URL `https://mcp.figma.com/mcp`
2. Выбрать auth type `oauth2`
3. Использовать preset `figma`
4. Указать `Client ID` и `Client Secret` своего Figma OAuth app
5. Зарегистрировать callback URL в Figma app
6. Нажать `Connect OAuth`

## Логирование и контроль

В MCPBox есть audit trail для операционного контроля.

Сейчас логируются, например:
- создание проекта;
- создание сервера;
- смена primary server;
- pause/resume проекта;
- start/stop сервера;
- enable/disable сервера;
- попытки MCP-подключения;
- forwarded JSON-RPC payloads;
- OAuth connect/disconnect;
- health-check результаты.

SQL-логирование GORM по умолчанию выключено, чтобы обычный MCP-трафик не засорял консоль.

## Требования

- Go `1.25+`
- Node.js и npm для сборки встроенного UI
- Windows, Linux или macOS

Внешняя БД не нужна. MCPBox по умолчанию создаёт локальный SQLite-файл.

## Структура проекта

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

## Настройка порта

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

## Локальные данные

По умолчанию MCPBox создаёт:

```text
mcpbox.db
```

Это локальные runtime-данные, их не нужно коммитить в git.

## Подключение клиентов

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

## HTTP API

### Health

```http
GET /healthz
```

### Проекты

```http
POST /api/projects
GET /api/projects
GET /api/projects/{id}/status
POST /api/projects/{id}/primary-server
POST /api/projects/{id}/pause
POST /api/projects/{id}/resume
```

### MCP-серверы

```http
POST /api/projects/{id}/servers
PUT /api/servers/{id}
DELETE /api/servers/{id}
POST /api/servers/{id}/start
POST /api/servers/{id}/stop
POST /api/servers/{id}/enable
POST /api/servers/{id}/disable
POST /api/servers/{id}/check
POST /api/servers/{id}/oauth-start
POST /api/servers/{id}/oauth-disconnect
GET /api/servers/{id}/inspect
```

### OAuth callback

```http
GET /oauth/callback
```

### Логи

```http
GET /api/logs
GET /api/logs?project_id={id}
```

## Текущие ограничения

На этом этапе в MCPBox ещё нет:
- полной встроенной аутентификации и авторизации пользователей MCPBox;
- управления пользователями и ролями;
- автоматической multi-server routing логики внутри одного проекта;
- продвинутых metrics dashboards;
- исторической аналитики сверх audit log;
- service installers для фонового запуска на уровне ОС.

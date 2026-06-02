# MCPBox Developer Guide

English version: [DEVELOPER.md](./DEVELOPER.md)

## Обзор

MCPBox - это control plane на Go для MCP-серверов.

Он группирует MCP-серверы по проектам, хранит конфигурацию в SQLite, запускает локальные `stdio` MCP-процессы, проксирует удаленные `HTTP streaming` MCP-серверы, отдает встроенный React admin UI и предоставляет один MCP URL на проект.

Сейчас MCPBox - это:
- control plane для MCP-серверов
- проектный организатор локальных и удаленных MCP-бэкендов
- aggregation endpoint для включенных серверов внутри проекта
- операторский UI для управления проектами, серверами, inspection, health-check и логированием

Сейчас MCPBox еще не:
- multi-user платформой с auth и ролями
- полноценной observability-платформой, хотя теперь в нем уже есть базовые встроенные метрики по задержкам, ошибкам и трафику
- распределенной системой оркестрации MCP между несколькими хостами

## Основная Модель

- `Project`: логическая workspace-группа MCP-серверов для клиента, команды или окружения
- `MCP Server`: либо локальный `stdio` сервер, который запускает MCPBox, либо удаленный `HTTP streaming` сервер
- `Catalog Item`: описание интеграции, синхронизированное из внешнего JSON manifest
- `Installed Integration`: запись уровня проекта, которая связывает catalog item с конкретным `MCPServer`
- `Installed Package`: переиспользуемый установленный runtime-пакет, который можно привязать к одному или нескольким проектам
- `Project Package Instance`: связь уровня проекта между installed package и конкретным managed `MCPServer`
- `Performance Metric`: легкая запись на один запрос, из которой строятся сводки по задержкам, ошибкам и трафику в экране логов

Важное поведение:
- у каждого проекта ровно один MCP URL
- project URL имеет вид `/mcp/{project_token}`
- endpoint проекта агрегирует все включенные серверы проекта
- если проект поставлен на паузу, новые MCP-подключения блокируются
- отключенные серверы исключаются из project endpoint
- локальные `stdio` серверы могут автоматически запускаться при старте приложения

## Текущие Возможности

### Backend

- Go HTTP API
- SQLite через GORM
- orchestration локальных `stdio` MCP-процессов
- proxy для удаленных `HTTP streaming` MCP-серверов
- локальные Базы знаний / RAG-коллекции на on-disk Bleve полнотекстовых индексах
- project-level endpoint `/mcp/{project_token}`
- synchronous JSON-RPC request/response bridge для HTTP MCP-клиентов
- legacy SSE compatibility mode для старых MCP-клиентов
- агрегированный список `tools`, `resources` и `prompts` по включенным серверам проекта
- маршрутизация tool, prompt и resource вызовов обратно в правильный backend
- audit logging для управляющих действий и MCP-трафика
- per-method aggregation audit logs для `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list` и `prompts/get`
- pause/resume проекта
- enable/disable сервера
- inspection локального сервера
- проверка health при create, update, start и manual check
- углубленный health probing после `initialize`, включая безопасные follow-up probe-вызовы для database-style и filesystem-style серверов
- sanitized health-check trace logging, чтобы диагностика сохранялась без утечки секретов
- sync каталога из внешнего JSON manifest или локально загруженного manifest-файла
- хранение installed integrations рядом с обычными MCP servers
- install/uninstall lifecycle пакетов с переиспользованием пакета между проектами
- проверка системных зависимостей перед установкой пакета
- manifest-driven обработка секретов, чтобы чувствительные значения уходили в env, а не оставались в видимых CLI args
- manifest-driven post-install health checks
- Docker runtime MVP для stdio-ориентированных container-backed catalog items
- поддержка `go_install` для Go-based stdio интеграций с размещением managed binary внутри директории установленного пакета
- легкие performance metrics по числу запросов, ошибкам, задержкам и трафику
- встроенный запуск Ollama через `github.com/mark3labs/mcphost/sdk`
- LM Studio deeplink launcher через `POST /api/projects/{id}/launch-lmstudio`

Примечание по Базе знаний:
- текущий RAG-слой использует классический локальный полнотекстовый поиск, а не embedding-индексы
- коллекции хранятся на диске и ищутся через Bleve
- внешняя векторная база не требуется
- pipeline генерации embeddings на этом этапе тоже не нужен

### Frontend

- встроенный React admin UI
- список проектов и project overview
- модальный create-project flow
- flow дублирования проекта с переименованием до создания
- модальный add-server flow для `stdio` и `HTTP streaming`
- вкладка Market для sync каталога, установки пакетов, удаления пакетов и add-to-project flow
- автоматический sync каталога при открытии Market view
- отображение project MCP URL
- start/stop локальных серверов
- health status и ручное действие `Check`
- немедленная обратная связь об успешном или неуспешном health-check после ручной проверки
- audit log console с фильтрацией по проекту
- встроенный performance dashboard внутри экрана логов
- автообновляемый просмотр логов
- `Info` modal для `stdio` серверов
- определение статуса Ollama и выбор локальной модели
- one-click действие `Launch Ollama` для подходящих проектов
- диалог `Запустить` для выбора локального сценария запуска
- действие `Add to LM Studio` из project UI
- локализация English и Russian

## Transport-Модель

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
- `POST /mcp/{project_token}` без `sessionId` работает как synchronous HTTP JSON-RPC
- `GET /mcp/{project_token}` открывает legacy SSE flow и возвращает `endpoint` event
- дальнейшие SSE-запросы идут через `POST /mcp/{project_token}?sessionId=...`
- paused project блокируется еще до открытия transport-а
- в project endpoint участвуют только включенные серверы
- верхнеуровневый capability discovery MCPBox отвечает сам, после чего fan-out делает в серверы проекта

Почему это важно:
- LM Studio ожидает synchronous HTTP request/response для вызовов вроде `initialize` и `tools/list`
- старые SSE-клиенты все еще используют legacy session model
- MCPBox поддерживает оба сценария без отдельных project URL

## Поведение Агрегатора

`/mcp/{project_token}` - это aggregation endpoint, а не route на один primary server.

Текущее поведение:
- `initialize` обрабатывается самим MCPBox
- `tools/list`, `resources/list` и `prompts/list` объединяют результаты всех включенных серверов проекта
- MCPBox добавляет project-level helpers в агрегированные MCP-ответы: `get_project_env_and_rules` как tool, `search_project_knowledge` при подключенных RAG-коллекциях и `project_prompt` как стандартный MCP prompt, если prompt проекта не пустой
- при необходимости MCPBox добавляет стабильные alias'ы, чтобы одинаковые имена из разных серверов не конфликтовали
- tool calls, prompt fetch и resource read маршрутизируются обратно в сервер-источник
- неподдерживаемые upstream-ответы для `resources/list` и `prompts/list` игнорируются, если сервер явно сообщает, что optional capability не реализована
- агрегированные вызовы также пишут нормализованные audit entries, чтобы downstream activity можно было отследить по методу и по backing server

Благодаря этому один project URL может представлять несколько MCP-бэкендов без смены конфигурации клиента.

## Inspection Для Локальных `stdio`

Для локальных `stdio` серверов MCPBox умеет делать live inspection MCP-возможностей.

Действие `Info` в UI доступно только для `stdio` серверов. Оно может показать:
- server metadata из `initialize`
- negotiated MCP capabilities
- доступные `tools`
- доступные `resources`
- доступные `prompts`
- соседний `README.md`, если MCPBox находит его рядом с локальным путем сервера

Для удаленных `HTTP streaming` серверов это UI-действие специально не показывается.

## Health-Проверка Серверов

MCPBox проверяет работоспособность MCP-сервера до того, как проблема всплывет уже внутри AI-клиента.

Текущее поведение:
- при создании сервера MCPBox сразу выполняет health-check и сохраняет результат
- при редактировании сервера MCPBox повторно проверяет обновленную конфигурацию
- ручной запуск локального `stdio` сервера считается успешным только если MCP health-check реально прошел
- UI показывает ручное действие `Check` для локальных и удаленных серверов
- API ответа `Check` теперь сразу возвращает обновленные health status, error text и timestamp

Текущая стратегия проверки:
- локальные `stdio` серверы проверяются через реальный MCP handshake: `initialize`, `notifications/initialized` и follow-up list вызовы
- удаленные `HTTP streaming` серверы проверяются через тот же JSON-RPC flow поверх HTTP, а не только через поверхностную проверку доступности
- если сервер публикует tools, MCPBox может сделать безопасный probe вроде `query`, `read_query`, `list_database` или filesystem listing, чтобы раньше поймать поломанную runtime-конфигурацию
- неподдерживаемые list-методы вроде `resources/list` или `prompts/list` считаются допустимыми, если upstream сервер явно их не реализует
- последнее состояние, текст ошибки и время проверки сохраняются в SQLite и показываются в UI

Trace behavior:
- каждый health run может писать `server_health_trace` audit entries с request, response и failure diagnostics
- секреты маскируются в аргументах команд, URL, заголовках и JSON payload до записи trace details

## Каталог И Интеграции

MCPBox умеет синхронизировать внешний catalog manifest в SQLite и устанавливать его элементы в проекты.

Основные API routes:

```http
GET /api/catalog/items
GET /api/catalog/items?enabled_only=1
POST /api/catalog/sync
GET /api/packages
DELETE /api/packages/{id}
POST /api/projects/{id}/integrations
```

Установленный catalog item создает обычный project-linked `MCPServer`, поэтому основной project endpoint остается `/mcp/{project_token}` без отдельной transport-модели.

Нюансы sync каталога:
- `POST /api/catalog/sync` принимает либо удаленный `url`, либо `manifest_content` вместе с `file_name`
- legacy URL каталога вроде `https://webeasy.kz/mcpbox/catalog.json` нормализуются в `https://mcpbox.sh/catalog.json`
- ошибки sync пишутся и в UI, и в audit log

Нюансы manifest-контракта:
- `icon_url` поддерживается в catalog cards и dialog'ах
- `system_dependencies` могут блокировать install, если не хватает бинарников вроде `git`, `psql` или `docker`
- `default_env` можно передавать как объект или как массив пар `{ key, value }`
- `env_schema` описывает env-переменные для install/add-to-project dialog, включая `secret: true`
- config fields тоже могут объявлять `secret: true` и `env_var`, чтобы MCPBox сохранял секрет в `server.EnvJSON`, а не оставлял его в видимых CLI args
- `health_check` может требовать post-install verification и при необходимости блокировать add-to-project
- Docker catalog items сейчас ориентированы на stdio-style `docker run --rm -i ...` и установку через `docker_pull`
- Go catalog items теперь могут использовать `install.strategy: "go_install"` вместе с `source.package` и entry point, чтобы установленный бинарник копировался в managed package directory

Нюансы жизненного цикла пакета:
- install пакета выполняется один раз на конкретную версию catalog item
- uninstall пакета блокируется, пока хотя бы один проект все еще использует этот пакет
- `Add to project` создает новый project package instance и обычный managed `MCPServer`

Нюанс жизненного цикла проекта:
- `POST /api/projects/{id}/duplicate` клонирует метаданные проекта, серверы, package instances, installed integrations и подключенные knowledge-base links, создавая при этом новый token

## Интеграция С Ollama

В MCPBox есть встроенный launcher для локального Ollama-сценария и тестирования MCP в один клик.

Текущее поведение:
- UI запрашивает `GET /api/ollama/status`
- кнопка показывается только если `ollama` установлена
- status endpoint возвращает список найденных локальных моделей и default model
- запуск проекта выполняется через `POST /api/projects/{id}/launch-ollama`
- MCPBox пишет временный `mcphost` config, который смотрит обратно на project endpoint
- затем MCPBox открывает новый terminal session и запускает собственный subcommand `ollama-chat`
- `ollama-chat` использует `github.com/mark3labs/mcphost/sdk`, поэтому отдельный бинарник `mcphost` пользователю не нужен

Практический нюанс:
- сама `ollama` все равно должна быть установлена локально, потому что MCPBox не встраивает runtime модели, а запускает локальный Ollama-backed session

## Интеграция С llama.cpp

MCPBox умеет запускать локальный `llama-server` и открывать его Web UI для выбранного проекта.

Текущее поведение:
- UI запрашивает `GET /api/llamacpp/status`
- запуск проекта выполняется через `POST /api/projects/{id}/launch-llamacpp`
- путь к модели берется из request, сохраненных настроек проекта или `MCPBOX_LLAMACPP_MODEL`
- MCPBox запускает `llama-server` с host `127.0.0.1`, настроенным портом, `--jinja`, опциональными настройками context/GPU/chat-template и сгенерированным `--system-prompt-file`, если prompt проекта не пустой
- managed state llama.cpp хранит PID, активную модель, URL сервера и digest project prompt, чтобы MCPBox перезапускал сервер при смене модели или prompt

Практический нюанс:
- `llama-server` должен быть установлен локально и доступен в `PATH`

## Интеграция С LM Studio

MCPBox умеет подготовить локальный проект для LM Studio без ручной сборки конфигурации.

Текущее поведение:
- UI вызывает `POST /api/projects/{id}/launch-lmstudio`
- MCPBox проверяет, что проект не на паузе и в нем есть хотя бы один включенный MCP-сервер или подключенная база знаний
- backend строит `lmstudio://add_mcp` deeplink, который указывает обратно на project endpoint
- имя сервера нормализуется из имени проекта и получает префикс `mcpbox_{project_id}_...`
- MCPBox просит локальную ОС открыть deeplink и возвращает deeplink payload в API response

## Поведение При Старте Приложения

При старте MCPBox загружает проекты из хранилища и решает, какие локальные серверы нужно поднять автоматически.

Текущее правило:
- если проект не на паузе и в нем есть хотя бы один включенный `stdio` сервер с `auto_start=true`, MCPBox запускает все включенные `stdio` серверы этого проекта

Это намеренно project-oriented поведение, а не запуск только первого сервера в списке.

## Логирование И Контроль

В MCPBox есть audit trail для операционного контроля.

Сейчас логируются, например:
- создание проекта
- создание сервера
- pause/resume проекта
- start/stop сервера
- enable/disable сервера
- попытки MCP-подключения
- forwarded JSON-RPC payloads
- health-check активность
- per-method aggregated MCP traffic
- действия запуска Ollama
- подготовка и открытие deeplink для LM Studio

Операционные детали:
- типовые informational stderr-строки от Filesystem-подобных MCP-серверов фильтруются и не выглядят как ложные access errors
- SQL-логирование GORM по умолчанию выключено, чтобы обычный MCP-трафик не засорял консоль

Помимо audit logs MCPBox теперь хранит и легкие performance metrics.

Основной API route:

```http
GET /api/logs/metrics
GET /api/logs/metrics?project_id={id}&window=5m
GET /api/logs/metrics?project_id={id}&window=1h
GET /api/logs/metrics?project_id={id}&window=24h
```

Текущее поведение metrics-слоя:
- метрики записываются вокруг proxied JSON-RPC calls и managed server method calls
- каждая запись хранит project, server, transport, operation, latency, request bytes, response bytes и success/failure
- экран логов использует эти данные для summary cards, trend charts, top slow servers, top error servers, top traffic servers и recent failures
- это намеренно легкий встроенный operational view, а не внешний Prometheus/OpenTelemetry-level observability stack

## Требования

- Go `1.26+`
- Node.js и npm для сборки встроенного UI
- Windows, Linux или macOS

Внешняя БД не нужна. MCPBox по умолчанию создает локальный SQLite-файл.

## Структура Проекта

```text
main.go                      точка входа приложения и startup orchestration
internal/models              GORM-модели
internal/storage             SQLite-хранилище и запросы
internal/orchestrator        жизненный цикл MCP-процессов, inspection, stdio bridge
internal/httpapi             HTTP API, MCP endpoint, встроенный UI, API запуска Ollama и LM Studio
internal/ollamahost          встроенный Ollama chat host на базе mcphost/sdk
internal/installer           сервис локальной установки пакетов
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

Прямой запуск встроенного локального Ollama host:

```bash
go run . ollama-chat --config /path/to/project.yml --model llama3.2
```

После успешного старта MCPBox автоматически открывает локальный UI и печатает:
- `http://127.0.0.1:<port>/`
- локальные IPv4 URL, например `http://192.168.x.x:<port>/`

## Настройка Порта

Порядок приоритета:
1. CLI-флаг `-port`
2. Переменная окружения `MCPBOX_PORT`
3. Значение по умолчанию `38180`

Пример:

```bash
go run . -port 39000
```

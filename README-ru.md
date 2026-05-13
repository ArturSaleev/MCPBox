# MCPBox

Русская документация. English version: [README.md](./README.md)

MCPBox — это Go-шлюз в виде одного бинарного файла для управления локальными MCP-серверами по проектам.

На текущем этапе MCPBox умеет:
- хранить проекты и конфигурации MCP-серверов в SQLite;
- запускать локальные MCP-серверы как дочерние процессы;
- отдавать SSE endpoint для каждого проекта;
- проксировать JSON-RPC сообщения из HTTP в stdio MCP-процесса;
- отдавать встроенный placeholder интерфейса админки.

## Текущий статус

Репозиторий находится на раннем этапе закладки фундамента.

Уже реализовано:
- Go backend с HTTP API;
- SQLite-хранилище через GORM;
- оркестрация процессов в `internal/orchestrator`;
- SSE-мост на `/connect/{project_token}`;
- базовое API для управления проектами и серверами;
- встроенная placeholder-страница админки.

Пока не реализовано:
- полноценная React + Tailwind админка;
- продвинутая маршрутизация сессий для нескольких MCP-серверов в одном проекте;
- production-ready аутентификация и контроль доступа;
- структурированные логи и метрики;
- install/service скрипты.

## Требования

- Go 1.25+
- Windows, Linux или macOS

Внешняя база данных не требуется. SQLite-файл создаётся локально.

## Структура проекта

```text
main.go                      Точка входа приложения
internal/models              GORM-модели
internal/storage             SQLite-хранилище и запросы
internal/orchestrator        Жизненный цикл MCP-процессов и stdio bridge
internal/httpapi             HTTP API, SSE endpoint, встроенный UI
```

## Сборка

```bash
go build -o MCPBox .
```

Для Windows:

```powershell
go build -o MCPBox.exe .
```

## Запуск

Порт по умолчанию: `38180`.

Запуск со стандартными настройками:

```bash
go run .
```

Для Windows-бинарника:

```powershell
.\MCPBox.exe
```

## Настройка порта

Порт можно задать двумя способами.

Порядок приоритета:
1. Флаг командной строки `-port`
2. Переменная окружения `MCPBOX_PORT`
3. Значение по умолчанию `38180`

Примеры:

```bash
go run . -port 39000
```

```powershell
.\MCPBox.exe -port 39000
```

```powershell
$env:MCPBOX_PORT=39000
.\MCPBox.exe
```

Рекомендация:
- `-port` использовать для ручного локального запуска и ярлыков;
- `MCPBOX_PORT` использовать для скриптов, CI и сервисной обвязки.

## Хранение данных

По умолчанию MCPBox создаёт локальный SQLite-файл:

```text
mcpbox.db
```

## HTTP API

### Проверка здоровья

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

Получение списка проектов:

```http
GET /api/projects
```

Получение статуса проекта:

```http
GET /api/projects/{id}/status
```

### MCP-серверы

Добавление сервера в проект:

```http
POST /api/projects/{id}/servers
Content-Type: application/json
```

```json
{
  "name": "Everything Server",
  "launch_command": "npx @modelcontextprotocol/server-everything",
  "working_dir": "",
  "auto_start": true
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

## SSE / JSON-RPC мост

У каждого проекта есть уникальный токен.

Открыть SSE stream:

```http
GET /connect/{project_token}
```

Отправить JSON-RPC запрос в MCP-процесс:

```http
POST /connect/{project_token}
Content-Type: application/json
```

Пример запроса:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

Поведение:
- `POST /connect/{project_token}` отправляет JSON-RPC payload в `stdin` дочернего процесса;
- `GET /connect/{project_token}` получает кадры из `stdout` дочернего процесса через SSE;
- в текущей реализации Stage 1 для подключения используется первый сервер проекта как активный сервер.

## Заметки по разработке

- Управление процессами вынесено в `internal/orchestrator`.
- Контроль жизненного цикла построен на `context.Context`.
- При остановке сначала выполняется graceful shutdown, затем принудительное завершение при необходимости.
- SQLite-драйвер используется pure-Go, чтобы локальная сборка не зависела от `gcc/cgo`.

## Важное правило сопровождения

Если в проект вносятся крупные изменения или изменения поведения, документация должна обновляться в том же наборе изменений:
- `README.md` для английской версии;
- `README-ru.md` для русской версии.

Репозиторий должен оставаться понятным для запуска только по документации.

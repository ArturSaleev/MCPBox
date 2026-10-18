# Архитектура и разработка

[Оглавление](README.md) | [English](../en/development.md)

MCPBox — один Go-модуль со встроенным React UI. `main.go` подключает учётные записи, SSO, интеграции и авторизацию к runtime из `app`. Каталог соседнего MCPBoxPro для сборки не нужен.

## Карта исходников

| Path | Responsibility |
| --- | --- |
| `app/`, `main.go` | Runtime, listeners, startup, admin commands |
| `internal/httpapi/` | Core HTTP API, project MCP aggregation, local launchers |
| `internal/orchestrator/` | Local processes, inspection and health |
| `internal/storage/` | Core models and SQLite persistence |
| `internal/installer/` | Marketplace packages |
| `internal/cli/` | CLI registry, catalog and bounded execution |
| `internal/rag/` | Local extraction, Bleve indexes and search |
| `internal/proauth/`, `internal/prosso/` | Users, roles, sessions, tokens, OIDC |
| `internal/prohttp/` | Authenticated administration and integration routes |
| `internal/propolicy/`, `internal/proprojects/` | PolicyBox bindings and project shadows |
| `internal/proragbox/` | External RagBox integration |
| `internal/prostorage/` | Storage configuration and operational connections |
| `connectruntime/` | Shared connection runtime |
| `html/src/` | Frontend sources |
| `internal/httpapi/ui/dist/` | Embedded frontend build output |

## Сборка и проверка

Нужны Go 1.26+ и Node.js/npm, совместимые с Vite из `html/package.json`. Из корня репозитория:

```bash
npm --prefix html ci
npm --prefix html run build
GOWORK=off go build -o MCPBox .
GOWORK=off go test ./...
```

После изменений UI пересоберите frontend и Go-бинарник. `npm --prefix html run dev` запускает Vite, но текущая конфигурация Vite не задаёт proxy к API; для проверки полного приложения используйте встроенную сборку. Для установки с постоянными данными используйте собранный бинарник: `go run` размещает исполняемый файл во временном каталоге.

## Поток запроса и границы

Админка обращается к защищённому Admin API. MCP-клиент обращается к `/mcp/{project-token}` на отдельном listener. Проверки проекта и авторизации выполняются перед вызовом доступных backend. Локальный `stdio` запускается на машине MCPBox, удалённый `http_stream` вызывается по сети. RagBox и PolicyBox работают отдельно. Промпт проекта не заменяет права целевой системы.

SQLite остаётся bootstrap-слоем; PostgreSQL не означает перенос всех core-таблиц. При изменениях хранилища проверяйте фактическое владение таблицами и runtime/shadow status. Внутренние имена `pro*` сохранены для совместимости.

## Релизы

Сначала установите frontend-зависимости (`npm ci` выше), затем выполните `./build-release.sh host` или `./build-release.sh all`. Скрипт пересобирает UI и создаёт `release/<os>/<arch>/MCPBox/` для macOS, Linux и Windows на amd64/arm64. Windows-бинарник имеет расширение `.exe`. Релизный скрипт копирует README и LICENSE; полный комплект документации находится в репозитории.

Подробности: [руководство разработчика](../../../DEVELOPER-ru.md), [HTTP API](api.md), [CLI reference](../../CLI.md). При изменениях поведения обновляйте обе языковые версии документации. Проверки backend не подтверждают работу внешних OAuth, PolicyBox/RagBox и реальных MCP-клиентов — для них нужны отдельные интеграционные проверки.

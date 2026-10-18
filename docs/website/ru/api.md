# Обзор API

[Оглавление](README.md) | [English](../en/api.md)

У MCPBox есть web UI и HTTP API. Все административные endpoints, включая общие `/api/projects`, `/api/servers`, `/api/logs` и `/api/projects/{id}/oauth-clients`, требуют аутентифицированную admin session или token. Чтение проверяет `pro:read`, изменения — `pro:write`, а удаление и раскрытие секретных токенов — `pro:admin`. Публичными остаются health/meta, маршруты входа и SSO, а также MCP/OAuth protocol endpoints. Используйте HTTPS и не публикуйте административное API без ограничений доступа.

## Основное API

| Раздел | Основные endpoints |
| --- | --- |
| Статус сервиса | `GET /healthz`, `GET /api/meta` |
| Проекты | `GET, POST /api/projects`; `GET, PUT, DELETE /api/projects/{id}` |
| Действия проекта | `POST /api/projects/{id}/servers`; токены, пауза/возобновление и прочие действия — `POST /api/projects/{id}/…` |
| OAuth-подключения проекта | `GET, POST /api/projects/{id}/oauth-clients`; показать token, включить, отключить или удалить — `/api/projects/{id}/oauth-clients/{clientID}/…` |
| MCP-серверы | `GET, PUT, DELETE /api/servers/{id}`; `POST /api/servers/{id}/start` и `/stop` |
| Marketplace | `GET /api/catalog/items`; `POST /api/catalog/sync`; `POST /api/catalog/items/{id}/install` |
| Пакеты | `GET /api/packages`; `DELETE /api/packages/{id}` |
| Базы знаний | `GET, POST /api/rag/collections`; `PUT, DELETE /api/rag/collections/{id}` |
| Логи и метрики | `GET /api/logs`, `GET /api/logs/metrics` |
| MCP проекта | `/mcp/{project-token}` |

## Admin API

| Раздел | Основные endpoints |
| --- | --- |
| Аутентификация | `POST /api/pro/login`, `GET /api/pro/auth/me`, OIDC-маршруты `/api/pro/sso/…` |
| Пользователи, сессии, токены | `/api/pro/users`, `/api/pro/sessions`, `/api/pro/tokens` |
| Конфигурация RagBox | `GET, PUT /api/pro/ragbox/config` |
| Администрирование RagBox | `/api/pro/ragbox/admin/vector-connections`, `/api/pro/ragbox/admin/workspaces` |
| Базы знаний RagBox | `GET /api/pro/ragbox/knowledge-bases`, `POST /api/pro/ragbox/knowledge-bases/sync` |
| Upload в базу знаний | `POST /api/pro/ragbox/collections/{collectionID}/files:upload`, multipart: `file`, `storage_mode=keep|transient` |
| Конфигурация PolicyBox | `GET, PUT /api/pro/policybox/config` |
| Тестовый flow PolicyBox | `POST /api/pro/policybox/runtime/ticket`, `/exchange`, `/validate` |
| Storage | `GET, PUT /api/pro/storage/config`; `GET /api/pro/storage/runtime-status`, `/shadow-status` |

API развивается вместе с релизами MCPBox. Эта страница — операционный обзор; точные поля payload следует сверять с UI работающего сервиса и документацией конкретного релиза. Не автоматизируйте destructive operations без проверки в непроизводственной среде.

## Пример административного запроса

Создайте API-токен в админке с нужным scope и задайте его в окружении как `MCPBOX_ADMIN_TOKEN`. Токен проекта не заменяет административный токен.

```bash
curl --fail-with-body http://127.0.0.1:38181/healthz
curl --fail-with-body \
  -H "Authorization: Bearer ${MCPBOX_ADMIN_TOKEN}" \
  http://127.0.0.1:38181/api/projects
```

`MCPBOX_ADMIN_TOKEN` здесь — переменная для примера curl, а не настройка сервера. `POST /api/pro/login` принимает JSON с `email` и `password` и возвращает `token`, `user`, `session`; используйте полученный token в заголовке Bearer. Не публикуйте ответ login в журналах.

Общие read/write scopes не описывают все привилегированные маршруты: изменение Storage и CLI, например, требует `pro:admin`. Точные проверки находятся рядом с регистрацией маршрутов.

При `401` проверьте токен, при `403` — права. На MCP listener отправляйте MCP JSON-RPC, а не административные запросы. Обычный HTTP health-check не подтверждает успешность MCP-вызова.

## Исходники контракта

- [Core routes](../../../internal/httpapi/server.go)
- [Login and SSO](../../../internal/prohttp/routes_auth.go)
- [Storage routes](../../../internal/prohttp/routes.go)
- [CLI schema and API](../../CLI.md)
- [Authentication middleware](../../../internal/proauth/middleware.go)

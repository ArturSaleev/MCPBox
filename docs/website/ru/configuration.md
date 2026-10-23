# Справочник настроек

[Оглавление](README.md) | [English](../en/configuration.md)

## Сеть и локальные модели

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `MCPBOX_HOST` | `127.0.0.1` | Адрес админки и Admin API |
| `MCPBOX_PORT` | `38181` | Порт админки |
| `MCPBOX_MCP_HOST` | `127.0.0.1` | Адрес MCP; локальная машина |
| `MCPBOX_MCP_PORT` | `38182` | Порт MCP |
| `MCPBOX_PUBLIC_MCP_URL` | не задана | Публичный HTTPS origin MCP/OAuth без пути |
| `MCPBOX_PRO_BOOTSTRAP_TOKEN` | не задана | Аварийный административный bearer-токен |
| `MCPBOX_OLLAMA_MODEL` | не задана | Предпочитаемая модель Ollama |
| `MCPBOX_LLAMACPP_MODEL` | не задана | Путь к GGUF-модели |
| `MCPBOX_LLAMACPP_MODEL_NAME` | не задана | Переопределение имени модели |
| `MCPBOX_LLAMACPP_PORT` | `39200` | Порт llama.cpp |
| `MCPBOX_LLAMACPP_CHAT_TEMPLATE_FILE` | не задана | Файл шаблона чата |
| `MCPBOX_LLAMACPP_CTX_SIZE` | не задана | Размер контекста llama-server |
| `MCPBOX_LLAMACPP_N_GPU_LAYERS` | не задана | Число GPU-слоёв llama-server |

Приоритет настроек сети: флаг командной строки, окружение, значение редакции. Флаги: `--host`, `--port`, `--mcp-host`, `--mcp-port`. Недопустимые порты вызывают ошибку. Для работы только на локальной машине:

```bash
MCPBOX_MCP_HOST=127.0.0.1 ./MCPBox
```

Задавайте переменные в оболочке или окружении службы. В точке входа не настроена загрузка `.env`. Для дочерних процессов нужны установленные runtime и доступные в PATH служебной учётной записи бинарники. Ollama и llama.cpp устанавливаются отдельно по необходимости.

## Публичный MCP URL и OAuth

Задайте `MCPBOX_PUBLIC_MCP_URL=https://mcp.example.com` в окружении процесса MCPBox и перезапустите службу. Значение должно быть HTTPS origin: без учётных данных, пути, query и fragment. Завершающий слеш допустим. Некорректное значение игнорируется; без корректного значения адрес определяется по запросу и заголовкам прокси.

Этот origin используется для OAuth discovery, метаданных защищённого ресурса и основного URL подключения проекта (`https://mcp.example.com/mcp/<project-token>`). Интерфейс показывает и копирует адрес от бэкенда, даже если админка использует другой домен или порт. Переменная не меняет адреса listener и не настраивает прокси.

При раздельных listener маршруты `/.well-known/` и `/oauth/` обслуживаются на MCP listener (по умолчанию `38182`); административный listener возвращает для них 404. Направляйте эти пути и `/mcp/` на MCP listener. OIDC-вход в административную панель по-прежнему использует `/api/pro/sso/callback` на admin listener. См. [подключение ChatGPT](chatgpt.md) и [сетевой доступ](network-access.md).

## OIDC SSO

Задайте `MCPBOX_PRO_OIDC_ISSUER_URL`, `MCPBOX_PRO_OIDC_CLIENT_ID`, `MCPBOX_PRO_OIDC_CLIENT_SECRET` и `MCPBOX_PRO_OIDC_REDIRECT_URL`. Путь callback: `/api/pro/sso/callback`. Зарегистрируйте точный публичный URL у провайдера идентификации.

| Дополнительная переменная | По умолчанию |
| --- | --- |
| `MCPBOX_PRO_OIDC_PROVIDER_NAME` | `OIDC` |
| `MCPBOX_PRO_OIDC_SCOPES` | `openid profile email`, через пробел |
| `MCPBOX_PRO_OIDC_DEFAULT_ROLE` | `reader` |
| `MCPBOX_PRO_OIDC_SESSION_DAYS` | `30` |
| `MCPBOX_PRO_OIDC_AUTO_CREATE_USERS` | `true` |
| `MCPBOX_PRO_OIDC_ALLOWED_HD` | пусто |

Переменная `MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET` больше не используется. См. [миграцию](policybox-migration.md).

## Данные и интеграции

По умолчанию `mcpbox.db` находится рядом с фактическим исполняемым файлом, независимо от текущего каталога терминала. В корне данных также находятся `package_store/`, `knowledge_base/managed/`, `knowledge_base/indexes/` и совместимый каталог `.mcpboxpro/rag-shadow/`. Служебной учётной записи нужны права записи.

Встраиваемый Go API принимает `app.Options.StoreDSN`; это не флаг `--db` и не переменная окружения исполняемого приложения. Относительные DSN разрешаются рядом с бинарником; абсолютные и специальные DSN с `:` / `file:` сохраняются. `go run .` использует временный бинарник: для постоянной установки соберите отдельный исполняемый файл.

RagBox, PostgreSQL и backend журналов настраиваются в Settings. SQLite остаётся bootstrap-базой. Выбор PostgreSQL не переносит автоматически все core-таблицы. После изменения проверьте runtime/shadow status; см. [интеграции](knowledge-integrations.md).

Имена `MCPBOX_PRO_*`, `/api/pro/*`, `pro:*` и `.mcpboxpro` сохранены для совместимости и не требуют платной редакции.

Исходники: [runtime](../../../app/runtime.go), [точка входа](../../../main.go), [SSO](../../../internal/prosso/service.go), [launcher](../../../internal/httpapi/llamacpp.go).

Настройки по умолчанию выше относятся к текущим исходникам. Опубликованный v1.3.0 использует прежний MCP bind `0.0.0.0`; до обновления явно задавайте `--mcp-host 127.0.0.1`. Удалённый доступ включается через `--mcp-host 0.0.0.0` или `MCPBOX_MCP_HOST=0.0.0.0`. [Сеть и изоляция проектов](network-access.md).

## Оформление

Кнопка с солнцем, луной или монитором внизу навигации переключает **Светлую**, **Тёмную** и **Системную** тему. Такой же переключатель доступен на экране входа. Системная тема следует настройке ОС, в том числе при её изменении во время работы. Явный выбор светлой или тёмной темы имеет приоритет над ОС. Выбор сохраняется в этом браузере под ключом `mcpbox-theme` и действует для страниц, форм, меню, диалогов и уведомлений; настройки сервера и проектов не меняются.

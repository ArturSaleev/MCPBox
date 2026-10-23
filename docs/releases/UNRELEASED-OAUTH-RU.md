# MCPBox — изменения OAuth/MCP для следующего релиза (черновик)

> **Статус:** реализовано в исходниках, интеграционно проверено на Ubuntu через Tailscale Funnel и ChatGPT. Не назначать номер версии до подготовки релиза. Этот файл — заметки к следующему релизу, не утверждение о публикации.

## Исправлено

- OAuth для MCP-проектов обслуживается на **MCP listener** (обычно порт **38182**) вместе с MCP endpoint, а не на административном listener (обычно **38181**). Маршруты `/.well-known/` и `/oauth/` на административном listener не обслуживаются.
- Исправлен URL метаданных защищённого MCP-ресурса в заголовке `WWW-Authenticate`: он формируется с публичным origin и конкретным путём проекта, а не с локальным адресом и портом.
- Добавлена переменная окружения `MCPBOX_PUBLIC_MCP_URL`. Если она содержит корректный HTTPS origin без пути, OAuth Discovery и публичный URL метаданных используют этот origin. Пример: `MCPBOX_PUBLIC_MCP_URL=https://mcp.example.com`. Без неё сохраняется определение адреса по HTTP-запросу/заголовкам reverse proxy.
- Добавлены Go-тесты на разделение OAuth между listener'ами и формирование публичного URL метаданных.
- В UI добавлено предпочтение публичного `/mcp/<project-token>` на HTTPS-origin открытой панели; **известное ограничение**: если панель находится на другом порту/домене, UI может показать адрес панели, а не внешний URL MCP. Перед релизом нужно исправить UI, чтобы он использовал явный Public MCP URL из backend.

## Проверка

- `go test ./internal/httpapi ./internal/prohttp` — успешно (23.10.2026, выполнено вручную).
- Ubuntu: после сборки и настройки `MCPBOX_PUBLIC_MCP_URL` OAuth Discovery на порту 38182 вернул правильный HTTPS issuer/authorize/token endpoints.
- Tailscale Funnel: после сохранения путей `/mcp/`, `/oauth/`, `/.well-known/` внешний запрос к `/.well-known/oauth-authorization-server` вернул HTTP 200.
- Подключение ChatGPT по OAuth к HomeLinux через MCPBox прошло успешно; создание и чтение тестового файла проверено.

## Развёртывание (Linux)

- Не менять внутренние порты: admin 38181, MCP 38182.
- В systemd через override добавить `Environment="MCPBOX_PUBLIC_MCP_URL=https://<публичный-домен>"` (без завершающего слеша и порта, если используется Funnel :443).
- Снаружи публиковать только пути MCP/OAuth, не административную панель. HTTPS/Tailscale Funnel может проксировать на localhost:38182; отдельный Nginx для MCPBox не нужен.
- У Tailscale `--set-path` необходимо учитывать преобразование пути; проверять фактические `/.well-known/` и `/oauth/` запросы. Не публиковать административный `/` через Funnel.

## Затронутые файлы исходников

- `internal/httpapi/server.go`
- `internal/prohttp/oauth.go`
- `internal/httpapi/runtime_connect_test.go`
- `html/src/app/App.tsx` (изменение отображения URL требует доработки перед релизом)

## Перед публикацией

- [ ] Исправить интерфейс: показывать явно настроенный Public MCP URL, а не HTTPS-origin панели на :10000.
- [ ] Повторно выполнить тесты и сборку встроенного React UI + Go для всех поддерживаемых платформ.
- [ ] Проверить прямой доступ, reverse proxy, Tailscale Serve/Funnel и отсутствие публичного доступа к admin API.
- [ ] Проверить, что `/.well-known/` и `/oauth/` недоступны на отдельном административном listener, а также поведение при совпадающих адресах listener'ов.
- [ ] Просмотреть безопасность доверия к `X-Forwarded-Host`/`X-Forwarded-Proto` от недоверенных клиентов.

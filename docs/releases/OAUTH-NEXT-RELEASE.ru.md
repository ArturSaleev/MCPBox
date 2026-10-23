# OAuth и MCP — заметки к следующему релизу

Статус: изменения внесены в исходный код, выпуск релиза ещё не выполнялся.

## Изменения

- OAuth discovery и endpoints `/oauth/` доступны через MCP listener (порт 38182), а не через отдельный административный listener (38181).
- Защищённый MCP endpoint возвращает корректный публичный путь метаданных в `WWW-Authenticate`.
- Добавлена переменная `MCPBOX_PUBLIC_MCP_URL`, задающая публичный HTTPS origin для OAuth и метаданных.
- Добавлены тесты маршрутизации и публичного URL. `go test ./internal/httpapi ./internal/prohttp` успешно выполнен 23 октября 2026.
- Соединение ChatGPT с MCPBox через OAuth и Tailscale Funnel проверено на Ubuntu.

## Затронутые файлы

- `internal/httpapi/server.go`
- `internal/prohttp/oauth.go`
- `internal/httpapi/runtime_connect_test.go`
- `html/src/app/App.tsx`

## Перед публикацией

- [x] Исправлено отображение MCP URL в UI: backend ставит `MCPBOX_PUBLIC_MCP_URL` первым в `connect_urls`/`connect_url`, а frontend использует предоставленный адрес без подстановки HTTPS-origin админ-панели (:10000). Добавлены тесты явного публичного URL и fallback без него.
- [ ] Запустить полную сборку UI и Go, проверить релизные платформы.
- [ ] Проверить маршруты OAuth/MCP и закрытость админ-панели извне.
- [ ] Проверить использование forwarded headers только от доверенного proxy.

Это внутренний черновик changelog, версия и дата релиза пока не назначены.

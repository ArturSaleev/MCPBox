# AGENTS.md

Инструкция для ИИ-агентов, работающих с кодом **MCPBox** — приложения, а не сайта.
Для человека: `README.md` (обзор), `DEVELOPER.md` (архитектура), `CONTRIBUTING.md`
(правила участия), `SECURITY.md` (уязвимости).

Публичный сайт и веб-документация — отдельный проект:
<https://github.com/ArturSaleev/mcpboxsite> (приватный), прод — <https://mcpbox.sh>.
Здесь только исходники продукта.

> Обновляйте этот файл, когда меняется сборка, структура пакетов, релизный процесс
> или правила совместимости.

---

## 1. Что это

MCPBox — self-hosted control plane для MCP-серверов: один Go-бинарник со встроенным
React-интерфейсом и локальным хранилищем.

Что делает:

- группирует MCP-серверы по **проектам** и выдаёт каждому проекту **один MCP URL**;
- запускает локальные `stdio`-серверы и проксирует удалённые `http_stream`;
- агрегирует `tools` / `resources` / `prompts` проекта и маршрутизирует вызовы
  обратно в нужный сервер;
- даёт доступ к разрешённым **CLI**-операциям, **Skills** (каталог, версии, скрипты
  в Docker) и **базе знаний** (локальный Bleve, опционально внешний RagBox);
- ведёт пользователей, роли, сессии, токены, project OAuth и опциональный OIDC SSO;
- показывает health-check, аудит действий и лёгкие метрики запросов;
- синхронизирует каталог интеграций (Market) из JSON-манифеста.

Чем **не** является: распределённой оркестрацией по нескольким хостам и полноценной
observability-платформой (метрики намеренно лёгкие и внутренние).

Лицензия — Apache-2.0. Возможности бывшего MCPBoxPro включены в этот же репозиторий
без активации; исторические имена `pro*` сохранены для совместимости.

Релизы: <https://github.com/ArturSaleev/MCPBox/releases>.

---

## 2. Стек и требования

| Слой | Технологии |
|---|---|
| Бэкенд | Go 1.26.9 (`go.mod`), стандартный `net/http`, `mux` без внешнего роутера |
| Хранение | SQLite через GORM (`glebarez/sqlite`, чистый Go, без CGO), опционально PostgreSQL |
| Поиск | Bleve v2 для локальной базы знаний |
| MCP | `mark3labs/mcphost`, `mark3labs/mcp-go`, собственный aggregation-слой |
| UI | React 18 + Vite 6 + MUI 7 + Radix + Tailwind 4 + react-router 7 |
| Раннеры | Docker — только для запуска скриптов Skills; Ollama / llama.cpp / LM Studio — локальные лаунчеры |
| CI | GitHub Actions, Node 24, Go из `go.mod` |

Сборка релизов идёт с `CGO_ENABLED=0`, поэтому бинарник переносимый.

---

## 3. Структура

```text
main.go                  точка входа: edition, startup hooks, admin-команды
app/                     рантайм: listeners, пути данных, запуск/остановка, edition API
connectruntime/          общий слой доступа к проекту (MCP-соединение)
internal/httpapi/        HTTP API, MCP-эндпоинт и агрегация, лаунчеры, встроенный UI
internal/orchestrator/   жизненный цикл локальных MCP-процессов, inspection, health
internal/storage/        GORM-хранилище и запросы
internal/models/         GORM-модели
internal/installer/      установка пакетов Market
internal/cli/            реестр CLI, каталог (embed cli.json), ограниченное исполнение
internal/skills/         каталог Skills (embed catalog_seed.json), пакеты, рантайм Docker
internal/rag/            извлечение текста, индексы Bleve, поиск
internal/proauth/        пользователи, роли, сессии, токены
internal/prosso/         OIDC SSO
internal/prohttp/        защищённые админ-маршруты и интеграции
internal/proprojects/    project shadows
internal/proragbox/      внешний RagBox: конфиг, привязки, ingest, поиск
internal/prostorage/     настройки хранилища и опциональный PostgreSQL
internal/httpapi/ui/dist встроенная сборка фронтенда (go:embed)
html/                    исходники админ-UI (React + Vite)
docs/website/{en,ru}/    публичная документация (по 18 страниц)
docs/*.md                внутренние документы: CLI, SKILLS, MERGE, STORAGE_ARCHITECTURE_RU, CI_VALIDATION
catalog/                 манифесты и источники каталога Skills
scripts/                 проверки документации, релизных артефактов, контрольных сумм
cmd/skills-catalog/      утилита сборки каталога Skills
build-release.sh         сборка шести релизных бинарников и SHA256SUMS
```

`internal/httpapi/server.go` — самый большой файл (~120 КБ): перед правкой ищите
нужный обработчик, а не читайте целиком. Аналогично `server_test.go`.

---

## 4. Сборка и запуск

Фронтенд **встроен** в бинарник через `//go:embed all:ui/dist`
(`internal/httpapi/ui.go`). Без собранного `internal/httpapi/ui/dist` бинарник
соберётся, но отдаст пустой интерфейс.

Порядок всегда такой:

```bash
npm --prefix html ci
npm --prefix html run build          # пишет в internal/httpapi/ui/dist
GOWORK=off go build -o MCPBox .
./MCPBox
```

Windows (PowerShell):

```powershell
npm --prefix html ci
npm --prefix html run build
$env:GOWORK = "off"
go build -o MCPBox.exe .
.\MCPBox.exe
```

`GOWORK=off` обязателен: так работает CI и так проект собирается независимо от
любых `go.work` рядом.

Запуск из исходников — `GOWORK=off go run .`. Важно: `go run` кладёт бинарник во
временный каталог, а данные хранятся **рядом с исполняемым файлом**, поэтому для
реальной работы и любых ручных проверок используйте собранный бинарник в отдельной
записываемой папке.

Служебные подкоманды (`main.go`): `admin create`, `project-http-stdio`,
`ollama-chat`, `llamacpp-chat`.

Первичный администратор:

```bash
./MCPBox admin create --email admin@example.com --password '<пароль>' --name 'Admin'
```

или интерактивно при первом запуске в терминале.

### Порты и адреса

| Что | По умолчанию |
|---|---|
| Админ-UI и API | `http://127.0.0.1:38181/` |
| MCP проекта | `http://127.0.0.1:38182/mcp/<project-token>` |

Приоритет настроек: CLI-флаг → переменная окружения → значение edition → дефолт.
Флаги `-host`, `-port`, `-mcp-host`, `-mcp-port`; переменные `MCPBOX_HOST`,
`MCPBOX_PORT`, `MCPBOX_MCP_HOST`, `MCPBOX_MCP_PORT`.

В текущих исходниках оба слушателя по умолчанию loopback. Опубликованные бинарники
v1.3.0 слушали MCP на `0.0.0.0` — это историческое отличие, не баг.

### Данные

Рядом с исполняемым файлом: `mcpbox.db`, `package_store/`, `knowledge_base/`,
`skill_store/`, `.mcpboxpro/`. База и `skill_store/` связаны: бэкапить нужно вместе.
Пути не зависят от рабочего каталога терминала.

---

## 5. Модель и инварианты

Ключевые сущности: `Project`, `MCPServer` (`stdio` или `http_stream`),
`CatalogItem`, `InstalledIntegration`, `InstalledPackage`, `ProjectPackageInstance`,
`PerformanceMetric`.

Инварианты, которые нельзя ломать:

- у проекта ровно один MCP URL: `/mcp/{project_token}` (алиас `/connect/{project_token}`);
- эндпоинт агрегирует **включённые** серверы проекта; выключенные не участвуют;
- paused-проект блокирует новые MCP-соединения;
- `initialize` MCPBox отвечает сам, затем fan-out на list/call;
- при конфликте имён MCPBox добавляет стабильные алиасы;
- в ответы подмешиваются проектные хелперы: `get_project_env_and_rules`,
  `search_project_knowledge` (при подключённой базе знаний), промпт `project_prompt`;
- вызовы `tools/call`, `resources/read`, `prompts/get` маршрутизируются в исходный сервер;
- health-check выполняется при создании, изменении, старте и по кнопке; результат
  (статус, ошибка, время) сохраняется в БД;
- секреты в трейсах health-check маскируются — не добавляйте логирование сырых
  аргументов, URL, заголовков и JSON;
- project prompt не заменяет авторизацию в целевой системе.

Транспорт: `POST /mcp/{token}` без `sessionId` — синхронный JSON-RPC; `GET` — legacy
SSE-поток с `endpoint`-событием, дальше `POST ...?sessionId=...`. Оба режима
обслуживает один и тот же URL — это требование совместимости клиентов.

---

## 6. Авторизация и API

- Админ-API требует логин; чтение — scope `pro:read`, запись — `pro:write`,
  привилегированные операции — `pro:admin`.
- Публично доступны только метаданные, логин, SSO-маршруты; MCP/OAuth проверяют
  доступ по своим правилам.
- `GET /api/meta` — идентичность сборки и capabilities для UI.
- Исторические имена сохраняются: маршруты `/api/pro/*`, scopes `pro:*`,
  переменные `MCPBOX_PRO_*`, имена таблиц БД. Менять их можно только вместе с
  планом миграции.

---

## 7. Хранилище

SQLite — базовая БД по умолчанию. PostgreSQL опционален и **не** означает, что все
таблицы ядра переехали: смотрите `docs/STORAGE_ARCHITECTURE_RU.md` и проверяйте,
какая таблица где живёт, прежде чем менять слой хранения.

Миграции — аддитивные. Данные пользователя (БД, настройки, пакеты, база знаний,
`.mcpboxpro/`) нельзя терять; изменения схемы должны иметь путь отката.

---

## 8. Тесты и проверки

```bash
GOWORK=off go test ./...
GOWORK=off go test -race ./internal/httpapi   # для конкурентных изменений
gofmt -l .                                    # должно быть пусто
GOWORK=off go vet ./...
GOWORK=off go mod verify
python3 scripts/check-doc-links.py
python3 -m unittest discover -s scripts -p 'test_*.py'
```

Проверки CI (`.github/workflows/ci.yml`) выполняют ровно это плюс:

- `bash -n build-release.sh scripts/release-checksums.sh`;
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`;
- `go build -trimpath`;
- `git diff --exit-code -- go.mod go.sum html/package-lock.json` — зависимости
  должны быть зафиксированы (после `npm ci`/`go mod tidy` коммитьте lock-файлы).

Тестов ~36 файлов, основной массив — `internal/httpapi/server_test.go`. У фронтенда
нет отдельного lint/test-скрипта: только `build` и `dev`.

Юнит-тесты не подтверждают интеграцию с реальными MCP-клиентами, OAuth, RagBox и
Docker — это проверяется отдельно, и в отчёте нужно честно разделять одно и другое.

---

## 9. Релизы

```bash
./build-release.sh host    # только текущая платформа
./build-release.sh all     # darwin/linux/windows × amd64/arm64
```

Скрипт сначала собирает UI, затем `go build -trimpath -ldflags="-s -w"` с
`CGO_ENABLED=0`, кладёт `release/MCPBox-<os>-<arch>[.exe]` и `SHA256SUMS`.
`release/` в git не коммитится.

Пуш тега `v*` запускает `.github/workflows/release-build.yml`: прогон CI →
`build-release.sh all` → проверка набора артефактов и хешей
(`scripts/check-release-artifacts.py`) → upload артефактов.

---

## 10. Правила и грабли

1. **`internal/httpapi/ui/dist` коммитится.** В `.gitignore` для него стоит
   исключение: «Embedded frontend build output must stay tracked». После правок в
   `html/` пересоберите UI и закоммитьте результат — иначе бинарник повезёт старый
   интерфейс, а Go-код останется прежним.
2. **`GOWORK=off`** для всех go-команд.
3. **Совместимость важнее чистоты.** `pro*`-имена, `/api/pro/*`, `MCPBOX_PRO_*`,
   имена таблиц — публичный контракт; переименование только с миграцией.
4. **Данные пользователя неприкосновенны.** Аддитивные миграции, никаких
   «пересоздадим БД».
5. **Обе локализации документации.** Меняется поведение — обновите
   `docs/website/en/*` и `docs/website/ru/*`, плюс `README.md`/`README-ru.md` при
   необходимости. `CONTRIBUTING.md` требует именно этого.
6. **Секреты.** Ни в коде, ни в фикстурах, ни в скриншотах, ни в логах; в примерах —
   только синтетические данные.
7. **Windows-специфика** вынесена в файлы с суффиксами `_windows.go` / `_unix.go` —
   не ломайте build-теги и проверяйте, что собирается на целевой ОС.
8. **Vite dev-сервер не равен приложению**: у него нет прокси к API, поэтому UI
   проверяется только встроенной сборкой (`npm --prefix html run build` + бинарник).
9. **`app.DefaultPort = 38180`** — историческая константа-фолбэк. Рабочие порты
   задаются edition (`38181` / `38182`); не «исправляйте» это, не разобравшись.
10. **RagBox — внешний сервис.** Не превращайте его в часть ядра и не переносите
    его данные.

---

## 11. Куда идти за типовой задачей

| Задача | Где смотреть |
|---|---|
| HTTP-маршрут, ответ API | `internal/httpapi/server.go`, тесты рядом |
| Семантика MCP, агрегация, транспорт | `internal/httpapi/mcp_aggregate.go`, `connectruntime/` |
| Жизненный цикл локальных серверов, health | `internal/orchestrator/` |
| Модели и запросы к БД | `internal/models/`, `internal/storage/` |
| Логин, роли, токены, OIDC | `internal/proauth/`, `internal/prosso/`, `internal/prohttp/` |
| Market и установка пакетов | `internal/installer/`, `internal/httpapi/integrations.go` |
| CLI-реестр и исполнение | `internal/cli/`, `internal/httpapi/cli*.go` |
| Skills | `internal/skills/`, `internal/httpapi/skills*.go`, `cmd/skills-catalog/` |
| База знаний, Bleve | `internal/rag/` |
| UI | `html/src/`, затем пересборка в `internal/httpapi/ui/dist` |
| Сборка/релизы | `build-release.sh`, `scripts/`, `.github/workflows/` |

---

## 12. Состояние на 2026-10-21

- Последний тег — `v1.4.1`; модуль `github.com/ArturSaleev/MCPBox`; Go 1.26.9.
- ~117 Go-файлов, ~36 файлов тестов; CI зелёный на `main`.
- UI собирается в `internal/httpapi/ui/dist` (~1.4 МБ) и лежит в репозитории.
- Каталог Skills встроен (`internal/skills/catalog_seed.json`, ~1.2 МБ) — это тот же
  набор, что `skills.json` на сайте; обновляется через `cmd/skills-catalog`.

Известные расхождения, которые стоит учитывать (не выдавайте их за новое поведение):

- `DEVELOPER.md` содержит раздел про 1.3.0 и местами описывает предыдущее состояние;
- `docs/releases/` содержит заметки только для 1.3.1 — заметок 1.4.x там нет;
- публичная документация на сайте — самостоятельные адаптации, автоматической
  синхронизации с `docs/website/` нет.

Возможные улучшения (обсуждаемы, не обещаны):

- единый источник версии (сейчас версия фигурирует в тегах и `html/package.json`);
- заметки к релизам 1.4.x в `docs/releases/`;
- прокси к API для `npm --prefix html run dev`, чтобы UI можно было отлаживать без
  пересборки бинарника;
- автоматическая сверка `docs/website/*` с репозиторием сайта.

---

## 13. Ссылки

- Репозиторий: <https://github.com/ArturSaleev/MCPBox>
- Релизы: <https://github.com/ArturSaleev/MCPBox/releases>
- Обсуждения: <https://github.com/ArturSaleev/MCPBox/discussions>
- Сайт и веб-документация: <https://mcpbox.sh> · репозиторий сайта
  <https://github.com/ArturSaleev/mcpboxsite> (приватный)
- Документация: `docs/website/en/README.md`, `docs/website/ru/README.md`
- Разработка: `DEVELOPER.md`, `docs/website/en/development.md`
- Участие: `CONTRIBUTING.md`

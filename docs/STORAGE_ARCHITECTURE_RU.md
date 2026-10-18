# Storage Architecture

Этот документ сохраняет историю поэтапной реализации storage. После объединения MCPBox/MCPBoxPro используется одно приложение. Текущий обзор: [RU](website/ru/knowledge-integrations.md) / [EN](website/en/knowledge-integrations.md); эксплуатация: [RU](website/ru/operations.md) / [EN](website/en/operations.md). Упоминания Free/Pro ниже относятся к прежней архитектуре.

## Базовый принцип

- `MCPBox` free не меняется и продолжает жить только на `SQLite`.
- `MCPBox` всегда поднимает bootstrap/system layer на `SQLite`.
- bootstrap `SQLite` не убирается даже после настройки production backend'ов.

## Роли storage-слоев

### 1. Bootstrap SQLite

Этот слой существует всегда и отвечает за:

- первый запуск;
- создание первого администратора;
- вход в Pro admin UI;
- сохранение конфигурации production backend'ов;
- хранение storage profile и правил fallback.

### 2. Operational backend

Целевой production backend для core state `MCPBox`:

- по умолчанию `sqlite`;
- после настройки администратором — `postgres`.

На первой фазе это пока только конфигурационный контур. Полный перенос operational state в `postgres` будет идти поэтапно.

### 3. Logs / events backend

Для high-volume logging и analytics поддерживается отдельный backend:

- `operational`
- `clickhouse`
- `elasticsearch`

Правило fallback:

- если `ClickHouse` или `Elasticsearch` не настроены, логирование должно падать обратно в `operational` backend;
- если operational backend уже переведен на `PostgreSQL`, fallback логов идет в `PostgreSQL`;
- если production backend еще не включен, fallback остается на bootstrap/runtime SQLite.

## Почему не config file как основной путь

Основной путь должен идти через admin UI:

1. сервис стартует;
2. bootstrap admin входит в Pro admin UI;
3. вводит `PostgreSQL` и при необходимости `ClickHouse` или `Elasticsearch`;
4. система сохраняет это в bootstrap `SQLite`;
5. после активации runtime начинает использовать внешние backend'ы.

На текущей фазе этот сценарий реализован через отдельный `Storage` tab в Pro settings:

- настройка `operational backend`;
- настройка `logs backend`;
- сохранение PostgreSQL полей (`host`, `port`, `database`, `username`, `password`) и previews для секретов;
- просмотр runtime fallback status;
- просмотр `project shadow` sync status.

Файл конфигурации можно допустить только как аварийный override, но не как главный продуктовый сценарий.

## Первая реализованная фаза

В первой фазе вводится bootstrap-конфиг storage backend'ов в `MCPBox`:

- конфигурация operational backend;
- конфигурация logs backend;
- явное правило fallback `logs -> operational`.

Это делается без вмешательства в `MCPBox` и без смешивания edition logic со storage backend logic.

## Текущие Pro API endpoints

- `GET /api/pro/storage/config`
  Возвращает сохраненный bootstrap-конфиг storage backend'ов.
- `PUT /api/pro/storage/config`
  Сохраняет bootstrap-конфиг storage backend'ов в `SQLite`.
- `GET /api/pro/storage/runtime-status`
  Возвращает фактический runtime status:
  - какой operational backend выбран;
  - какой backend реально активирован;
  - произошел ли fallback на bootstrap `SQLite`;
  - произошел ли fallback логов в operational backend;
  - видит ли система `PostgreSQL`, `ClickHouse` и `Elasticsearch`.
- `GET /api/pro/storage/shadow-status`
  Возвращает состояние `project shadow` синхронизации для operational backend:
  - сколько mirrored `projects`;
  - сколько mirrored `rag_collections`;
  - сколько mirrored project-to-collection links;
  - когда был последний sync по каждому типу.

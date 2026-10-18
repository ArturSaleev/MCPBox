# PolicyBox Runtime Integration

Этот документ описывает текущий runtime flow между внешней системой, `PolicyBox` и `MCPBox`.

## Источник истины

- внешняя система знает конечного пользователя
- внешняя система знает, какой AI-проект нужно открыть
- `PolicyBox` принимает runtime context, проверяет policy и выпускает runtime session
- `MCPBox` не аутентифицирует runtime-пользователя заново
- `MCPBox` доверяет только bearer token, проверенному через `PolicyBox`

## Текущий flow

1. Пользователь уже вошёл во внешнюю систему.
2. Backend внешней системы собирает runtime JSON.
3. Backend вызывает `POST /api/runtime/ticket` в `PolicyBox`.
4. `PolicyBox` проверяет policy проекта и сохраняет runtime context.
5. `PolicyBox` возвращает одноразовый `ticket`.
6. Клиент, gateway или BFF вызывает `POST /api/runtime/token-exchange`.
7. В ответ получает runtime bearer token.
8. Клиент или gateway вызывает `MCPBox /connect/runtime` с `Authorization: Bearer ...`.
9. `MCPBox` сам вызывает `PolicyBox /api/runtime/validate`.
10. `MCPBox` получает validated context, находит локальный проект и продолжает обычный MCP connect flow.

Для `AgentBox` добавлен соседний runtime bootstrap flow:

8. Клиент или `AgentBox` вызывает `MCPBox /api/agentbox/bootstrap` с тем же `Authorization: Bearer ...`.
9. `MCPBox` сам вызывает `PolicyBox /api/runtime/validate`.
10. `MCPBox` резолвит локальный проект через binding и возвращает bootstrap payload для embedded runtime.

Дополнительно для project endpoint:

- если в локальном проекте `MCPBox` включена галочка `Проверка идентификации`, обычный `/mcp/{project_token}` тоже начинает требовать `Authorization: Bearer ...`
- такой запрос валидируется через тот же `PolicyBox /api/runtime/validate`
- доступ даётся только если validated runtime project совпадает с binding этого локального проекта
- если bearer отсутствует, token невалиден или binding указывает на другой проект, доступ запрещается

Важно:

- базовый runtime flow больше не требует `X-Client-ID` и `X-Client-Secret` для `ticket` и `validate`
- trusted clients остаются optional-механизмом только для более жёсткого enterprise-режима, например защищённого `token-exchange`

## Текущий runtime JSON

Актуальный payload для `POST /api/runtime/ticket`:

```json
{
  "project_slug": "workspace-assistant",
  "requested_scopes": ["chat:use", "rag:query"],
  "context": {
    "tenant_id": "tenant-main",
    "workspace_id": "workspace-1",
    "resource_id": "task-42",
    "department_id": "operations"
  },
  "metadata": {
    "source_system": "external-platform",
    "page": "assistant-console"
  },
  "user": {
    "id": "operator-1",
    "name": "Ivan Ivanov",
    "role": "operator",
    "attributes": {
      "department": "operations",
      "region": "kz-almaty"
    }
  }
}
```

Смысл полей:

- `project_slug`
  Идентификатор AI-проекта в `PolicyBox`
- `requested_scopes`
  Какие runtime-права просит внешний клиент
- `context`
  Основной business/runtime context
- `metadata`
  Дополнительные интеграционные и диагностические данные
- `user`
  Конечный пользователь или subject из внешней системы

## Legacy compatibility

Для совместимости `PolicyBox` всё ещё понимает старые поля:

- `clinic_id`
- `department_id`
- `patient_id`
- `encounter_id`

Но новые интеграции должны использовать `context`, а не строить контракт вокруг этих полей.

## Что делает MCPBox

После `validate` `MCPBox` использует:

- `project_id`
- `project_slug`
- `granted_scopes`
- `context`
- `metadata`
- `user`

Дальше `MCPBox`:

1. сопоставляет `PolicyBox project` с локальным проектом через binding
2. создаёт runtime access context
3. маршрутизирует запрос в обычный `/mcp/{project_token}` flow

## Project binding

`PolicyBox` и `MCPBox` живут отдельно, поэтому проект нужно связать явно:

- в `PolicyBox` у проекта есть свой `project_slug` и `project_id`
- в `MCPBox` у локального проекта есть свой `project_token`
- runtime route в `MCPBox` использует binding `PolicyBox project -> MCPBox project`

Практически это значит:

- runtime token валиден сам по себе, но без binding `MCPBox` не поймёт, в какой локальный проект маршрутизировать запрос
- для проектов с включённой `Проверкой идентификации` этот же binding обязателен и на обычном `/mcp/{project_token}` endpoint
- тот же binding теперь используется и для `AgentBox` bootstrap, чтобы `AgentBox` не решал сам, какой local project открыть

## AgentBox bootstrap contract

Для embedded runtime `MCPBox` теперь отдаёт `POST /api/agentbox/bootstrap`.

Этот endpoint:

- принимает bearer runtime token
- валидирует его через `PolicyBox`
- резолвит локальный `MCPBox` project
- возвращает минимальный runtime bootstrap payload

Текущий payload включает:

- `agent_name`
- `project_name`
- `project_token`
- `prompt`
- `connect_url`
- `summary_blocks`
- `recommended_hints`
- `action_definitions`
- optional `provider_policy`
- `runtime_access`

Этого достаточно, чтобы `AgentBox`:

- поднял widget
- показал project summary
- показал project actions
- создал chat session
- получил runtime prompt
- позже смог подключаться к `MCPBox /connect/runtime` тем же bearer token

Сейчас `action_definitions` уже отдаются в минимальном рабочем формате:

- actions рендерятся отдельно от hints
- первый поддерживаемый тип: `kind = "prompt"`
- такой action выполняется через отдельный `AgentBox /api/actions/execute`

Пока `provider_policy` считается подготовленным расширением контракта:

- `AgentBox` уже умеет его учитывать
- для `llama.cpp` `MCPBox` теперь уже может собирать project-level `provider_policy` из project fields `llama_cpp_model_name` / `llama_cpp_model_path`
- для остальных провайдеров отдельный project-level source of truth пока еще не выделен в отдельную конфигурацию

## Когда нужны trusted clients

Trusted clients не нужны для базового flow:

- `ticket`
- `token-exchange` в простом режиме
- `validate`

Они нужны только если ты хочешь включить более жёсткий режим:

- отдельный trusted gateway/BFF
- защищённый `token-exchange`
- IP allowlist
- ops-инвентаризацию интеграций через admin API

## Практические правила интеграции

- не передавай `POLICYBOX_JWT_SECRET` наружу
- не смешивай admin auth и runtime access
- не считай `MCPBox` источником истины по runtime-пользователю
- передавай business context через `context`
- используй `metadata` только для вспомогательных данных
- держи `project_slug` стабильным для внешней интеграции

## Минимальный чеклист

1. Поднять `PolicyBox`.
2. Создать project policy в `PolicyBox`.
3. Сформировать runtime JSON во внешней системе.
4. Проверить flow `JSON -> ticket -> token -> validate`.
5. Настроить binding `PolicyBox project -> MCPBox project` в `MCPBox`.
6. Проверить `MCPBox /connect/runtime` с bearer token.
7. Если проект должен быть закрыт только для верифицированных runtime-пользователей, включить в проекте `Проверка идентификации` и проверить уже обычный `/mcp/{project_token}` с тем же bearer token.

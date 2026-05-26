# MCPBox Pro Roadmap

Этот документ описывает запланированное направление `Pro` для MCPBox. Это продуктовая дорожная карта, а не описание уже реализованного функционала.

Статус:
- `Free / 1.2.0` уже реализован
- `Pro` пока не реализован
- этот файл описывает только будущие возможности

## Что Уже Есть В Free

Текущая бесплатная версия сфокусирована на встроенном локальном MCP control plane и встроенном поисковом слое Knowledge Base / RAG.

Сейчас в free-версии есть:
- локальные Knowledge Base коллекции
- локальная индексация через полнотекстовый поиск Bleve
- project-level `search_project_knowledge`
- поддержка кода, текста, `CSV`, `XLSX`, `DOCX`, `PPTX` и `PDF` с текстовым слоем
- без внешней vector database
- без генерации embedding-индексов
- без OCR pipeline

## Планируемые Возможности Pro

Ни одно из направлений ниже не входит в `1.2.0`. Это будущая `Pro`-линейка.

### 1. Продвинутый RAG И Внешние Источники Данных

Цель:
- расширить текущий локальный поисковый слой до полноценной enterprise knowledge system

Планируемое направление:
- REST API для внешнего обогащения контекста, например `POST /api/v1/knowledge-base/enrich`
- one-click коннекторы для Notion, Confluence, Jira, Google Drive и GitHub
- гибридный поиск, который сочетает lexical retrieval на Bleve и semantic или vector retrieval

Отличие от Free:
- `Free` использует только локальный полнотекстовый поиск Bleve
- `Pro` должен добавить внешнюю синхронизацию и semantic retrieval слой

### 2. Auth, SSO, RBAC И Agent Tokens

Цель:
- поддержать multi-user и enterprise-сценарии управления доступом

Планируемое направление:
- SSO и OAuth2 вход через Google Workspace, GitHub, GitLab, LDAP или Active Directory
- RBAC для админов, разработчиков, аналитиков и других внутренних ролей
- scoped API tokens для агентов и автоматизаций с TTL и ограничением прав

Отличие от Free:
- в `Free` пока нет edition-level auth или role system
- `Pro` должен добавить identity, access policies и delegated tokens

### 3. Продвинутый OCR RAG

Цель:
- сделать сканированные и image-based документы пригодными для поиска

Планируемое направление:
- OCR ingestion для сканированных PDF, счетов, договоров, актов и изображений
- локальная обработка через Tesseract или локальные мультимодальные модели через Ollama
- извлечение текста без обязательной отправки данных в сторонние облака

Отличие от Free:
- `Free` поддерживает только PDF с текстовым слоем
- `Pro` должен добавить OCR для сканов и изображений

### 4. Team Control Plane

Цель:
- сделать MCPBox пригодным для централизованной командной эксплуатации

Планируемое направление:
- shared server-side deployment MCPBox для команды
- централизованное управление конфигами и секретами для внутренних сервисов
- группировка MCP-серверов по окружениям: Development, Staging и Production

Отличие от Free:
- `Free` сфокусирован на локальном embedded workflow для одного пользователя или небольшой команды
- `Pro` должен добавить централизованный team operations layer

### 5. Security И Guardrails

Цель:
- усилить безопасность до того, как действия модели дойдут до чувствительных систем

Планируемое направление:
- блокировка или подтверждение для деструктивных действий, например опасных SQL-запросов
- маскирование PII перед тем, как логи или tool context попадут в payload, видимый модели
- более сильный аудит и policy enforcement вокруг действий модели

Отличие от Free:
- `Free` уже даёт audit visibility, но не включает advanced policy enforcement
- `Pro` должен добавить активные safety controls и data redaction

### 6. Code Awareness И AST Parsing

Цель:
- улучшить качество поиска по коду по сравнению с обычным text chunking

Планируемое направление:
- AST-aware parsing через инструменты вроде tree-sitter
- code chunks, выровненные по классам, методам, функциям и синтаксическим границам
- более точный retrieval для больших и сложных codebase

Отличие от Free:
- `Free` использует практичный line/text chunking и Bleve
- `Pro` должен добавить syntax-aware понимание кода

### 7. Agent Memory Layer

Цель:
- дать подключенным моделям переиспользуемую систему памяти для предпочтений пользователя, фактов проекта и ранее принятых решений между сессиями

Планируемое направление:
- project-scoped и user-scoped записи памяти для фактов, предпочтений, troubleshooting notes и pinned decisions
- MCP tools и API routes для поиска, добавления, просмотра, обновления и удаления memory-записей
- retrieval, который подмешивает в контекст только релевантную память вместо повторной передачи всей истории диалога
- optional future semantic ranking для recall памяти после более простого structured или full-text MVP

Отличие от Free:
- `Free` дает только контекст текущей сессии и project knowledge search
- `Pro` должен добавить явное долгосрочное хранилище памяти и retrieval для агентов

## Граница Между Free И Pro

`Free / 1.2.0`:
- встроенный локальный control plane
- локальный полнотекстовый Knowledge Base search
- без semantic vector pipeline
- без OCR
- без SSO и RBAC

`Pro / future`:
- enterprise ingestion и коннекторы
- гибридный lexical plus semantic retrieval
- OCR для сканов и изображений
- multi-user auth и role controls
- централизованный team deployment и security guardrails
- AST-aware code intelligence
- долгосрочная agent memory для пользователей и проектов

# CI и проверки релизов

[Оглавление](README.md) | [English](../en/ci.md)

## Workflow CI

`.github/workflows/ci.yml` настроен на push веток, pull requests и ручной запуск. Его также вызывает релизный workflow. Проверяются форматирование Go, синтаксис shell, локальные ссылки README, регрессионные тесты валидатора релизов, `npm ci`, сборка встроенного frontend, `go mod verify`, `go vet ./...`, `go test ./...` и сборка Go-бинарника. `GOWORK=off` исключает зависимость от родительского workspace.

Используются Ubuntu, Node.js 24 и версия Go из `go.mod`. Actions закреплены SHA конкретных проверенных релизов. Права read-only, checkout не сохраняет credentials; PR-проверкам не нужны секреты репозитория или `pull_request_target`.

### Отчёты по зависимостям

Отдельная advisory-задача запускает `npm audit --omit=dev --audit-level=high` и закреплённый `govulncheck v1.9.0`. Находки и ошибки сканирования видны в предупреждениях, summary и артефакте `dependency-audit-reports` со сроком хранения 14 дней. Эти два шага не блокируют CI; ошибки установки зависимостей и подготовки среды не скрываются. Находки разбираются отдельно; более строгий release gate требует отдельного решения. Успех основных проверок не означает отсутствие уязвимостей в отчётах.

## Workflow сборки релиза

`.github/workflows/release-build.yml` запускается по тегу `v*` или вручную. Сначала вызывает CI, затем запускает поддерживаемый `./build-release.sh all`: шесть плоских бинарников и `SHA256SUMS` создаются в отдельном каталоге runner. `scripts/check-release-artifacts.py` требует точный набор файлов, сверяет все SHA-256 и проверяет соответствие заголовков ELF/Mach-O/PE именам файлов до загрузки.

Actions artifact называется `MCPBox-release-<commit SHA>` и хранится 14 дней. Workflow готовит файлы, но не создаёт GitHub Release и не заменяет assets существующих релизов. Публикуйте бинарники и манифест одной сборки. После скачивания Actions artifact на Unix может потребоваться право запуска. Проверка заголовков и кросс-сборка не заменяют запуск на родной чистой ОС.

## Включение и первый запуск владельцем

Опубликуйте workflows и вызываемые scripts на GitHub. Если Actions выключены, разрешите необходимые GitHub Actions в настройках репозитория. Запустите CI вручную на нужной ветке, проверьте **Tests and build** и advisory-отчёты. В branch protection/rulesets назначьте фактически появившийся успешный **Tests and build** обязательной проверкой до merge: наличие YAML само по себе не включает required checks.

Вручную запустите Release build, скачайте артефакт и проверьте манифест. Badge CI добавляется в оба README только после успешного запуска на GitHub. Пока hosted-запуск не подтверждён, badge не добавлен. Доступ на запись и запуск GitHub проверяются отдельно от локальной валидации.

## Локальные команды

```bash
npm --prefix html ci
npm --prefix html run build
GOWORK=off go mod verify
GOWORK=off go vet ./...
GOWORK=off go test ./...
GOWORK=off go build -trimpath -o MCPBox .
python3 scripts/check-doc-links.py
python3 -m unittest discover -s scripts -p 'test_*.py'
```

Для релиза направьте `RELEASE_DIR` в отдельный тестовый каталог, выполните `./build-release.sh all`, затем `python3 scripts/check-release-artifacts.py <каталог>`. Не используйте каталог рабочей установки для CI. Статус native-проверок описан в [проверке релизов](release-verification.md).

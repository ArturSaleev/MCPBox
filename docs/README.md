# MCPBox documentation / Документация MCPBox

- [Русская документация](website/ru/README.md)
- [English documentation](website/en/README.md)

The guides cover installation, configuration, everyday use, integrations, API, operations, and development. / Руководства охватывают установку, настройки, работу с проектами, интеграции, API, эксплуатацию и разработку.

## Technical references / Технические материалы

- [Developer guide (EN)](../DEVELOPER.md) / [Руководство разработчика (RU)](../DEVELOPER-ru.md)
- [CLI reference](CLI.md)
- [Migration to unified MCPBox (RU)](MERGE.md)
- [Storage implementation notes (RU)](STORAGE_ARCHITECTURE_RU.md)
- [PolicyBox runtime integration (RU)](POLICYBOX_RUNTIME_INTEGRATION_RU.md)

Historical implementation notes can describe earlier phases. Use the bilingual guides for the current entry points and verify detailed contracts against source.

## Link validation / Проверка ссылок

From the repository root / Из корня репозитория:

```bash
python3 scripts/check-doc-links.py
python3 scripts/check-doc-links.py --external --published-ref main
```

The first command checks local targets and Git tracking. The second also checks HTTP availability of external links and relative targets in published `main`, including screenshots. It requires `curl`; release binaries are checked with HEAD without downloading them. Localhost examples are skipped. HTTP success does not validate authentication flows or document contents. / Первая команда проверяет наличие файлов и их учёт Git; вторая также проверяет HTTP-доступность внешних ссылок и целей в опубликованной `main`, включая снимки. Нужен `curl`; бинарники проверяются HEAD-запросом без скачивания. Примеры localhost пропускаются. Успешный HTTP-ответ не подтверждает работу авторизации или корректность содержимого.

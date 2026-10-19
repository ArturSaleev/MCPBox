# Skills catalog parser / Парсер каталога Skills

Builds the static `schema_version: 1` JSON consumed by MCPBox. No Skill scripts or install commands are executed. MCPBox and the parser remain Apache-2.0; third-party packages keep their own licenses.

Создаёт JSON `schema_version: 1` для маркета MCPBox. Не запускает скрипты навыков и команды установки. Лицензия приложения и парсера — Apache-2.0; сторонние навыки сохраняют свои лицензии.

## Repeatable update / Повторное обновление

Run from the MCPBox repository / Из корня MCPBox:

```sh
GOWORK=off go run ./cmd/skills-catalog \
  --repositories-file catalog/skills-sources.txt \
  --cache-dir release/skills-parser/cache \
  --keep-going \
  --out ../MCPSite/public/skills.json
```

The repository list can be edited. Commits are resolved by `git ls-remote`; GitHub API authentication is not required. Each archive is downloaded at a fixed commit and cached. Full descriptions come from the repository's `SKILL.md`, rather than truncated directory cards. Names are deduplicated by repository/path, not display name. Requirements are taken from author metadata and are not invented.

Список репозиториев можно редактировать. Commit определяется через `git ls-remote`, без токена GitHub API. Архив скачивается по конкретному commit и кэшируется. Полные описания берутся из `SKILL.md`, а не из обрезанных карточек сайта. Дубли определяются по репозиторию/пути, а не по названию. Требования берутся из метаданных автора; отсутствующие значения не выдумываются.

## Directory discovery / Обнаружение через сайт

```sh
GOWORK=off go run ./cmd/skills-catalog --out skills.json
```

Follows directory, category, author, detail and pagination links on `https://mcpservers.org/ru/agent-skills`. If Cloudflare requires browser verification, parsing stops; the existing catalog is not replaced. The parser never solves challenges or bypasses access controls.

Обходит ссылки каталога, категорий, авторов, навыков и пагинации. Если источник требует проверку Cloudflare, работа останавливается, старый каталог сохраняется. Парсер не решает проверки и не обходит ограничения доступа.

For previously saved public HTML pages / Для сохранённых публичных HTML-страниц:

```sh
GOWORK=off go run ./cmd/skills-catalog \
  --html-dir catalog/skills-pages \
  --cache-dir release/skills-parser/cache \
  --keep-going --out skills.json
```

`--html file.html` accepts one page; `--html-dir directory` reads all `.html` files. Browser verification pages are rejected. HTML files are read as data and never executed. The initial export contains 95 publicly accessible pages linked from the directory landing page; this is not a claim of complete website coverage. The repository index includes all valid Skills found in each discovered repository, so it may contain more Skills than the saved directory cards.

`--html file.html` читает одну страницу, `--html-dir directory` — все `.html`. Страницы проверки отклоняются. HTML обрабатывается как данные, без выполнения. Начальный экспорт содержит 95 доступных страниц, связанных с главной страницей каталога: это не полный обход всего сайта. В обнаруженных репозиториях индексируются все корректные Skills, поэтому количество навыков может превышать количество сохранённых карточек.

## Failure handling / Ошибки

By default, a repository failure aborts the build and keeps the previous JSON. `--keep-going` explicitly allows a partial update and retains old entries for failing repositories when an existing valid output is present. A report is written to `<output>.report.json` with the source list, success/failure counts and reasons. Never publish that report as the catalog itself. All-empty results and invalid catalogs cannot overwrite an existing catalog. Output replacement is atomic; public JSON permissions are `0644`.

По умолчанию ошибка репозитория прерывает сборку без замены JSON. `--keep-going` явно разрешает частичное обновление и сохраняет прежние записи недоступных репозиториев, если уже есть корректный выходной файл. Отчёт `<output>.report.json` содержит список источников, количества успешных/неуспешных операций и причины ошибок. Отчёт не является каталогом. Пустой или некорректный результат не заменяет рабочий JSON. Замена атомарная, права публичного JSON — `0644`.

Other flags / Другие параметры:

- `--repositories URL,URL`: explicit sources / источники через запятую.
- `--resolve-with api`: resolve commits through GitHub API instead of git; optional `GITHUB_TOKEN` / GitHub API вместо git, необязательный `GITHUB_TOKEN`.
- `--source URL`: alternate HTTPS directory / другой HTTPS-каталог.
- `--max-pages 10000`: fail rather than silently truncate discovery / остановка вместо скрытого усечения обхода.
- `--report file.json`: custom report path / путь отчёта.

## Files for publication / Что публиковать

Upload **only the generated `skills.json`** to the site's public root, where it becomes `https://mcpbox.sh/skills.json`. Local site source: `../MCPSite/public/skills.json`. Parser executables, source snapshots, caches and reports do not belong in the public website root. The parser has no upload/deploy operation.

На сайт загружается **только готовый `skills.json`**, по адресу `https://mcpbox.sh/skills.json`. Исходник сайта: `../MCPSite/public/skills.json`. Бинарники парсера, снимки страниц, кэш и отчёты не нужно размещать в публичном корне. Парсер сам ничего на сервер не отправляет.

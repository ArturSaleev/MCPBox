# GitHub community setup / Настройка GitHub

Repository-owner follow-up for the community files in this change. No settings are enabled by committing these files alone.

## Private vulnerability reporting — required / Обязательно

The owner chose GitHub-only reporting; do not add a public email address. The read-only repository API returned `enabled: false` during preparation.

As a repository owner/admin, open **Settings → Security and quality → Advanced Security** (GitHub may vary the navigation wording), find **Private vulnerability reporting**, and enable it. Then verify that **Security → Advisories → Report a vulnerability** is available to an ordinary signed-in reporter. Configure security notifications so new reports reach the maintainer. Review [GitHub's setup instructions](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository).

После включения проверьте кнопку приватного отчёта и уведомления. Затем обновите раздел «Состояние настройки / Setup status» в обеих SECURITY-версиях на подтверждённое состояние. До этого не заявляйте, что приватный канал уже работает.

## Publish and verify / Публикация

1. Publish `CONTRIBUTING.md`, `CONTRIBUTING-ru.md`, `SECURITY.md`, `SECURITY-ru.md`, `.github/ISSUE_TEMPLATE/` and `.github/PULL_REQUEST_TEMPLATE.md` on the default branch.
2. Open **Issues → New issue**. Check the Bug report and Feature request forms, required fields, and documentation/security links. Do not submit a real issue just to preview a form.
3. Open a test PR creation page and verify the PR body template without publishing a test PR.
4. Check **Security → Policy** and the Russian policy link. Verify private reporting separately as above.

## Optional community features / Необязательные функции

Discussions is currently disabled. Keep setup questions in documentation and reproducible failures in Bug reports; enable Discussions only when the owner wants a separate Q&A surface and can moderate it.

A standalone Code of Conduct is deferred until the owner chooses a private conduct-reporting contact and moderation process. CONTRIBUTING already states basic respectful-participation rules. Do not invent a reporting address, enforcement team or response SLA.

## Validation limits / Границы проверки

Local YAML/schema checks validate the issue-form files; they do not prove GitHub has loaded them. Live publication/settings changes require repository write/admin access, which the available integration previously rejected with HTTP 403. Keep this task under review until publication and private reporting are verified.

## Actions / CI

After publishing `.github/workflows/`, run CI and Release build manually. Review [English](website/en/ci.md) / [Русский](website/ru/ci.md), then require the observed **Tests and build** status in repository rules. Add a README badge only after a successful hosted run.

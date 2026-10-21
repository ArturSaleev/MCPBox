# Contributing to MCPBox

[Русский](CONTRIBUTING-ru.md) · [Documentation](docs/website/en/README.md) · [Security reports](SECURITY.md)

Bug reports, focused improvements, documentation and translations are welcome. Issues and pull requests may be written in English or Russian. For substantial changes, describe the problem in a feature request before implementing a large patch.

## Choose the right channel

- **Bug:** use the [Bug report form](https://github.com/ArturSaleev/MCPBox/issues/new?template=bug_report.yml), with a minimal reproduction.
- **Feature:** use the [Feature request form](https://github.com/ArturSaleev/MCPBox/issues/new?template=feature_request.yml), explaining the problem and desired outcome.
- **Security vulnerability:** follow [SECURITY.md](SECURITY.md). Do not disclose it in a public issue or PR.
- **Setup questions:** start with [troubleshooting](docs/website/en/troubleshooting.md). GitHub Discussions is not required for contributing; use a bug report if the documented setup fails.

## Local development

Use Go 1.26+ and Node.js/npm compatible with the Vite version in `html/package.json`. From your fork or local checkout:

```bash
npm --prefix html ci
npm --prefix html run build
GOWORK=off go build -o MCPBox .
GOWORK=off go test ./...
```

Rebuild the frontend and then the Go executable after UI changes: the executable embeds `internal/httpapi/ui/dist`. Preserve the existing generated-assets convention when submitting UI changes. The Vite dev server has no API proxy configured, so it is not equivalent to running the full embedded application.

Run the built executable in a dedicated writable directory with test data. Data is stored beside the executable by default, including `mcpbox.db`, `package_store/`, `knowledge_base/` and `.mcpboxpro/`. `go run` uses a temporary executable. Never point tests at a real installation's data or reuse production credentials.

Current source defaults both listeners to loopback; administration uses `38181`, MCP uses `38182`. See [network access](docs/website/en/network-access.md) for explicit remote configuration and the difference from published v1.3.0 binaries.

## Scope and implementation

- Keep changes focused on one problem; separate unrelated formatting or refactoring.
- Use `gofmt` for changed Go files. Follow the surrounding frontend style; `html/package.json` currently defines build/dev scripts, not a separate lint or test command.
- Add meaningful regression tests for changed behavior, particularly authentication, project isolation, execution limits and persistent state. Documentation-only changes do not need backend tests.
- Preserve user databases, settings and external integration data. Use additive migrations where possible and document any upgrade or rollback impact.
- Keep secrets out of code, fixtures, screenshots, logs and generated assets. Use synthetic data in examples.
- RagBox is a separate integration. Preserve its data and boundaries; do not add unrelated integrations.
- Update both English and Russian documentation when changing documented behavior. Keep compatibility identifiers such as `/api/pro/*` and `MCPBOX_PRO_*` unless the change includes a migration plan.

## Validation

Run checks relevant to the change and report their actual results. Backend changes normally require `GOWORK=off go test ./...`; concurrency-sensitive changes may also need a targeted `go test -race`. UI changes require a frontend rebuild and a check of the resulting application. External OAuth, AI clients, RagBox require separate integration checks; do not describe unit tests as live integration validation.

Documentation link checks:

```bash
python3 scripts/check-doc-links.py
python3 scripts/check-doc-links.py --external --published-ref main
```

The published check only passes for targets already on `main`; new pages can fail until merged. The local check intentionally flags referenced files that Git does not track. Online checks require `curl` and skip localhost examples.

For packaging changes, use `RELEASE_DIR` pointing to a separate test directory. The maintained script is `./build-release.sh`; `all` builds the six flat binaries and `SHA256SUMS`. Do not overwrite an installed release or runtime database just to test packaging.

## Pull requests

Explain the problem, resulting behavior, validation and any migration/security impact. State what you could not test. Reference the related issue when one exists. Use the PR template; screenshots are useful for UI changes after removing secrets. A small fix does not require a feature proposal or a promised delivery date.

Respect other contributors: discuss the code, avoid personal attacks, and do not publish another person's private information. Contributions are submitted under this repository's [Apache-2.0 license](LICENSE); retain relevant third-party notices.

## Contributor sign-off (DCO)

Every commit in a pull request must be signed off. Create it with `git commit -s` (long form `--signoff`); each sign-off appends a trailer line to the commit message:

```text
Signed-off-by: Full Name <name@example.com>
```

The name and email must identify the person who made the contribution. The trailer may appear anywhere in the trailer block, but it must be present in every commit of the pull request.

A sign-off certifies that you have the right to submit the contribution under this repository's [Apache-2.0 license](LICENSE), as defined by the [Developer Certificate of Origin 1.1](https://developercertificate.org/). It is a confirmation, not a transfer of copyright: you keep the copyright in your contribution.

Pull requests that contain unsigned commits, or whose sign-off identity does not match the commit author, are not accepted. To fix a missing sign-off, amend or rebase the affected commits and push again.

[CI workflows, advisory reports and release preparation](docs/website/en/ci.md).

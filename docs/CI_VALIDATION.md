# CI local validation / Локальная проверка CI

Validation against the current working tree, 2026-10-18. Hosted Actions execution is not confirmed.

- Local runtime: macOS arm64, Go 1.26.0, Node.js 26.8.1. The workflow selects Ubuntu and Node.js 24; those hosted versions were not executed locally.
- `npm ci`, Vite build, `go mod verify`, `go vet ./...`, `go test ./...` and `go build` passed.
- actionlint v1.7.12 accepted both workflow definitions. Release-validator regression tests passed, including tampering and wrong-architecture failures.
- The current source built all six targets via the canonical release script in a separate temporary directory; the exact file set, SHA-256 and target architecture checks passed for every binary. This was cross-compilation, not native execution of foreign targets.
- Workflow permissions, trigger paths and YAML are locally checked; this does not confirm GitHub has loaded the workflows or enforced branch rules.

## Advisory findings / Находки аудита

`npm audit --omit=dev --audit-level=high` reported one high-severity finding in `react-router` 7.13.0 (affected range `6.0.0 - 7.18.1`; audit proposes 7.18.4). `govulncheck v1.9.0` reported 20 symbol-level findings from seven modules. These are scanner results requiring review, not proof of exploitation. Scanning used the toolchain selected by the Go tool (govulncheck's tool module selected Go 1.26.9 locally).

No automatic `npm audit fix`, dependency upgrades or vulnerability suppressions were applied. The scans are advisory and their failures remain visible. Resolve the findings as a separate tested dependency update before treating the release as security-reviewed. Database updates may change subsequent scan results.

| Go advisory | Module | Found in | Fixed in |
| --- | --- | --- | --- |
| GO-2026-6629 | golang.org/x/text | golang.org/x/text@v0.35.0 | golang.org/x/text@v0.41.0 |
| GO-2026-6617 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.60.0 |
| GO-2026-6612 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.60.0 |
| GO-2026-6611 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.60.0 |
| GO-2026-6610 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.60.0 |
| GO-2026-6603 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.60.0 |
| GO-2026-6348 | google.golang.org/grpc | google.golang.org/grpc@v1.79.1 | google.golang.org/grpc@v1.83.1 |
| GO-2026-6061 | google.golang.org/grpc | google.golang.org/grpc@v1.79.1 | google.golang.org/grpc@v1.82.1 |
| GO-2026-5970 | golang.org/x/text | golang.org/x/text@v0.35.0 | golang.org/x/text@v0.39.0 |
| GO-2026-5764 | github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream | github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream@v1.7.5 | github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream@v1.7.8 |
| GO-2026-5506 | go.opentelemetry.io/otel | go.opentelemetry.io/otel@v1.40.0 | go.opentelemetry.io/otel@v1.41.0 |
| GO-2026-5320 | github.com/yuin/goldmark | github.com/yuin/goldmark@v1.7.16 | github.com/yuin/goldmark@v1.7.17 |
| GO-2026-5030 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5029 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5028 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5027 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5026 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5025 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.55.0 |
| GO-2026-5004 | github.com/jackc/pgx/v5 | github.com/jackc/pgx/v5@v5.6.0 | github.com/jackc/pgx/v5@v5.9.2 |
| GO-2026-4918 | golang.org/x/net | golang.org/x/net@v0.51.0 | golang.org/x/net@v0.53.0 |

Проверки сборки прошли локально; запуск GitHub, Node.js 24/Ubuntu и required checks ожидают публикации и проверки владельцем. Находки аудита не устранены и не скрыты.

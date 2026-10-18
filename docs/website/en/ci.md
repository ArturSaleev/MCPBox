# CI and release checks

[Documentation](README.md) | [Русский](../ru/ci.md)

## CI workflow

`.github/workflows/ci.yml` is configured for branch pushes, pull requests and manual runs. It is also reusable by the release workflow. It checks Go formatting, shell syntax, local README links, the release-validator regression tests, `npm ci`, the embedded frontend build, `go mod verify`, `go vet ./...`, `go test ./...`, and a Go executable build. `GOWORK=off` keeps the checkout independent of parent workspaces.

The workflow uses Ubuntu, Node.js 24 and the Go version from `go.mod`. Actions are pinned to verified release commit SHAs. Permissions are read-only, checkout credentials are not persisted, and PR jobs do not need repository secrets or `pull_request_target`.

### Dependency reports

A separate advisory job runs `npm audit --omit=dev --audit-level=high` and pinned `govulncheck v1.9.0`. Findings and scan errors are visible in warnings, the run summary and the `dependency-audit-reports` artifact (14-day retention). These two scan steps are non-blocking; dependency installation/setup failures are not silently ignored. Review findings separately and agree on any stricter policy before making the audit a release gate. A successful main check does not mean the dependency reports are clean.

## Release build workflow

`.github/workflows/release-build.yml` runs for `v*` tags or manually. It calls CI first, then uses the maintained `./build-release.sh all` to produce six flat platform/architecture binaries and `SHA256SUMS` in an isolated runner directory. `scripts/check-release-artifacts.py` requires the exact file set, verifies every SHA-256 and checks ELF/Mach-O/PE machine headers against filenames before upload.

The downloadable Actions artifact is named `MCPBox-release-<commit SHA>` and retained for 14 days. This workflow prepares artifacts; it does not create a GitHub Release or replace an existing release's assets. Publish binaries and their checksum manifest from the same run. Artifact downloads may require restoring execute permission on Unix. Header checks and cross-compilation do not replace native clean-OS runtime testing.

## Owner activation and first run

Publish the workflows and referenced scripts on GitHub. If Actions is disabled, enable the required GitHub Actions in repository settings. Run CI manually on the intended branch and inspect **Tests and build** and the advisory reports. Use branch protection/rulesets to require the actual successful **Tests and build** check before merge; merely committing a workflow does not enable required checks.

Run Release build manually to verify artifact delivery, then download it and recheck the manifest. Add a CI badge to both READMEs only after a successful GitHub-hosted CI run. No badge has been added while live execution is unverified. Repository write access and a hosted run remain separate from local validation.

## Local equivalents

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

For a release, set `RELEASE_DIR` to a separate test directory, run `./build-release.sh all`, then `python3 scripts/check-release-artifacts.py <directory>`. Never use an active installation's data directory for CI tests. See [release verification](release-verification.md) for native test status.

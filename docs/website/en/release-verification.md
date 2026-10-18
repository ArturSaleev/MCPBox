# Verify a downloaded release

[Documentation](README.md) | [Русский](../ru/release-verification.md)

Download binaries from the [official GitHub release](https://github.com/ArturSaleev/MCPBox/releases/tag/v1.3.0). Before renaming or running a file, calculate its SHA-256 and compare it with the line for that exact filename in [v1.3.0 SHA256SUMS](../../checksums/v1.3.0-SHA256SUMS.txt).

## Check one file

macOS (example: Apple Silicon):

```bash
shasum -a 256 MCPBox-macos-arm64
```

Linux (example: Intel/AMD):

```bash
sha256sum MCPBox-linux-amd64
```

Windows PowerShell (example: Intel/AMD):

```powershell
Get-FileHash .\MCPBox-windows-amd64.exe -Algorithm SHA256
```

Compare all 64 hexadecimal characters; letter case does not matter. If the value differs, do not run the file: confirm the version/architecture, download again from the official release and investigate persistent mismatches. A matching checksum establishes file identity, not code signing, notarization or the absence of vulnerabilities.

If you have all six binaries and the manifest in one directory, Linux can check them with `sha256sum -c v1.3.0-SHA256SUMS.txt`; on macOS use `shasum -a 256 -c v1.3.0-SHA256SUMS.txt`. These commands report missing files when only one binary was downloaded; use the individual-file instructions above instead.

## Provenance and current verification

The v1.3.0 manifest records the `sha256:` digests returned by GitHub's release-assets API. All six local release files were independently hashed and matched those digests. This checks the already-published v1.3.0 bytes; rebuilding the current source may produce different hashes. The manifest is included in this repository; uploading it as a separate release asset remains pending repository write access.

| Published artifact | SHA-256 matched GitHub | Runtime validation |
| --- | --- | --- |
| macOS arm64 | Yes | Isolated data directory on the current Apple Silicon Mac; not a clean OS |
| macOS amd64 | Yes | Pending on Intel macOS |
| Linux amd64 | Yes | Pending on clean Linux amd64 |
| Linux arm64 | Yes | Pending on clean Linux arm64 |
| Windows amd64 | Yes | Pending on clean Windows amd64 |
| Windows arm64 | Yes | Pending on clean Windows arm64 |

The macOS arm64 smoke check covers admin creation/login, health, embedded UI, protected API, project creation/listing and persistent data beside the executable when launched from another directory. It does not establish compatibility with every OS version or external MCP/SSO service.

## Security prompts and startup errors

- **macOS Gatekeeper:** an unidentified-developer or notarization warning can occur. Verify the source and checksum, review [Apple's guidance](https://support.apple.com/102445), and stop if macOS reports malware or tampering. Keep Gatekeeper enabled; do not strip quarantine attributes as an installation step.
- **Windows SmartScreen:** a reputation warning can occur for an unfamiliar executable. Verify the source and checksum and follow your organization's policy. Keep SmartScreen/Defender enabled. See [Microsoft's guidance](https://support.microsoft.com/en-us/office/protect-my-pc-from-viruses).
- **Wrong architecture / executable format:** choose amd64 for Intel/AMD or arm64 for ARM. Do not assume translation/emulation is installed.
- **Permission denied:** on macOS/Linux, grant execute permission to the verified file with `chmod +x MCPBox`; install in a directory writable by your account.
- **Port already in use:** choose free admin/MCP ports; keep the same ports in client URLs and any tunnel configuration.

## Release maintainer checklist

`./build-release.sh all` is the canonical tracked script. It generates six flat binaries named like the public assets and `release/SHA256SUMS`. `host` generates only the current host binary and a manifest covering that run. Existing requested binaries are replaced only after all requested builds succeed; other files and runtime data are preserved. The old ignored `release/build-release.*` scripts are not the maintained entry point.

The build requires Go, installed frontend npm dependencies and `sha256sum` or `shasum`. Publish the manifest alongside the binaries from the same run. Before release, test each file on its native clean OS: first admin, login, UI, project, one MCP tool call, data persistence, restart and upgrade from a backed-up 1.2.x database. Record the OS/version and result. A cross-build or matching hash does not complete that runtime check.

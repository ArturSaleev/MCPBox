#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
MODE="${1:-all}"
RELEASE_DIR="${RELEASE_DIR:-$ROOT_DIR/release}"
BUILD_CACHE_DIR="${GOCACHE:-${TMPDIR:-/tmp}/mcpbox-gocache}"
BUILD_TMP_DIR="${GOTMPDIR:-${TMPDIR:-/tmp}/mcpbox-gotmp}"

usage() {
  cat <<EOF
Usage: ./build-release.sh [host|all]
Build flat MCPBox-<os>-<arch> binaries and SHA256SUMS in RELEASE_DIR.
macOS assets use "macos"; Windows assets use ".exe".
Requires Go, npm, installed frontend dependencies, and sha256sum or shasum.
Environment: RELEASE_DIR, GOCACHE, GOTMPDIR.
No existing binary is replaced unless every requested target builds successfully.
EOF
}
case "$MODE" in
  -h|--help|help) usage; exit 0 ;;
  host|all) ;;
  *) usage >&2; exit 2 ;;
esac
for executable in go npm; do
  command -v "$executable" >/dev/null 2>&1 || { echo "Missing $executable" >&2; exit 1; }
done
if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
  echo "Install sha256sum or shasum." >&2
  exit 1
fi
if [[ "$MODE" == host ]]; then
  TARGETS=("$(GOWORK=off go env GOOS)/$(GOWORK=off go env GOARCH)")
else
  TARGETS=(darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/arm64 windows/amd64)
fi
mkdir -p "$RELEASE_DIR" "$BUILD_CACHE_DIR" "$BUILD_TMP_DIR"
RELEASE_DIR="$(cd "$RELEASE_DIR" && pwd)"
BUILD_CACHE_DIR="$(cd "$BUILD_CACHE_DIR" && pwd)"
BUILD_TMP_DIR="$(cd "$BUILD_TMP_DIR" && pwd)"
staging_dir="$(mktemp -d "$RELEASE_DIR/.build.XXXXXX")"
trap 'rm -rf "$staging_dir"' EXIT

npm --prefix html run build
artifacts=()
for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  asset_os="$goos"
  [[ "$goos" != darwin ]] || asset_os=macos
  artifact="MCPBox-$asset_os-$goarch"
  [[ "$goos" != windows ]] || artifact="$artifact.exe"
  echo "Building $target -> $artifact"
  GOWORK=off GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 GOCACHE="$BUILD_CACHE_DIR" GOTMPDIR="$BUILD_TMP_DIR" \
    go build -trimpath -ldflags="-s -w" -o "$staging_dir/$artifact" .
  artifacts+=("$artifact")
done
bash "$ROOT_DIR/scripts/release-checksums.sh" "$staging_dir" "${artifacts[@]}"
for artifact in "${artifacts[@]}"; do
  mv "$staging_dir/$artifact" "$RELEASE_DIR/$artifact"
done
mv "$staging_dir/SHA256SUMS" "$RELEASE_DIR/SHA256SUMS"
echo "Built ${#artifacts[@]} target(s) in $RELEASE_DIR; SHA256SUMS covers this run only."

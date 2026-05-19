#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

export GOCACHE="$ROOT/.gocache"
RELEASE_DIR="$ROOT/release"

mkdir -p "$RELEASE_DIR"

echo "[1/3] Building embedded UI..."
npm --prefix html run build

echo "[2/3] Cleaning previous release binaries..."
rm -f \
  "$RELEASE_DIR/MCPBox-windows-amd64.exe" \
  "$RELEASE_DIR/MCPBox-windows-arm64.exe" \
  "$RELEASE_DIR/MCPBox-linux-amd64" \
  "$RELEASE_DIR/MCPBox-linux-arm64" \
  "$RELEASE_DIR/MCPBox-macos-amd64" \
  "$RELEASE_DIR/MCPBox-macos-arm64"

build() {
  local goos="$1"
  local goarch="$2"
  local output="$3"

  echo "  - ${goos}/${goarch}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$output" .
}

echo "[3/3] Building Go binaries..."
build windows amd64 "$RELEASE_DIR/MCPBox-windows-amd64.exe"
build windows arm64 "$RELEASE_DIR/MCPBox-windows-arm64.exe"
build linux amd64 "$RELEASE_DIR/MCPBox-linux-amd64"
build linux arm64 "$RELEASE_DIR/MCPBox-linux-arm64"
build darwin amd64 "$RELEASE_DIR/MCPBox-macos-amd64"
build darwin arm64 "$RELEASE_DIR/MCPBox-macos-arm64"

echo "Release build completed successfully."

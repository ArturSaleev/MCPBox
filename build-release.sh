#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

MODE="${1:-all}"
RELEASE_DIR="${RELEASE_DIR:-$ROOT_DIR/release}"
BIN_NAME="MCPBox"
RELEASE_NAME="MCPBox"
MAIN_PACKAGE="."
BUILD_CACHE_DIR="${GOCACHE:-/private/tmp/mcpbox-gocache}"
BUILD_TMP_DIR="${GOTMPDIR:-/private/tmp/mcpbox-gotmp}"

HOST_GOOS="$(GOWORK=off go env GOOS)"
HOST_GOARCH="$(GOWORK=off go env GOARCH)"
HOST_TARGET="${HOST_GOOS}/${HOST_GOARCH}"

ALL_TARGETS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/arm64"
  "linux/amd64"
  "windows/arm64"
  "windows/amd64"
)

usage() {
  cat <<EOF
Usage:
  ./build-release.sh [host|all]

Modes:
  host  Build only for the current Go host target
  all   Build all supported platforms and architectures (default)

Environment:
  RELEASE_DIR  Override release output path
  GOCACHE      Override Go build cache path
  GOTMPDIR     Override Go temporary build path

Artifacts:
  $RELEASE_DIR/<os>/<arch>/$RELEASE_NAME/
EOF
}

ensure_prerequisites() {
  command -v go >/dev/null 2>&1 || {
    echo "error: go is not installed" >&2
    exit 1
  }
  command -v npm >/dev/null 2>&1 || {
    echo "error: npm is not installed" >&2
    exit 1
  }
}

build_embedded_ui() {
  echo
  echo "==> Building embedded UI"
  (
    cd "$ROOT_DIR/html"
    npm run build
  )
}

binary_name_for_target() {
  local goos="$1"
  if [[ "$goos" == "windows" ]]; then
    printf '%s.exe' "$BIN_NAME"
  else
    printf '%s' "$BIN_NAME"
  fi
}

artifact_path_for_target() {
  local goos="$1"
  local goarch="$2"
  local binary_name
  binary_name="$(binary_name_for_target "$goos")"
  printf '%s/%s/%s/%s/%s' "$RELEASE_DIR" "$goos" "$goarch" "$RELEASE_NAME" "$binary_name"
}

build_target() {
  local target="$1"
  local goos="${target%/*}"
  local goarch="${target#*/}"
  local binary_name
  binary_name="$(binary_name_for_target "$goos")"
  local platform_dir="$RELEASE_DIR/$goos/$goarch"
  local target_dir="$platform_dir/$RELEASE_NAME"
  local output_path
  output_path="$(artifact_path_for_target "$goos" "$goarch")"

  echo
  echo "==> Building $target"

  mkdir -p "$target_dir"

  if GOWORK=off GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 GOCACHE="$BUILD_CACHE_DIR" GOTMPDIR="$BUILD_TMP_DIR" \
    go build -trimpath -ldflags="-s -w" -o "$output_path" "$MAIN_PACKAGE"; then
    cp "$ROOT_DIR/LICENSE" "$ROOT_DIR/README.md" "$ROOT_DIR/README-ru.md" "$target_dir/"
    SUCCESS_TARGETS+=("$target")
    echo "Built artifact: $output_path"
  else
    FAILED_TARGETS+=("$target")
    echo "Build failed for target: $target"
  fi
}

select_targets() {
  case "$MODE" in
    host)
      TARGETS=("$HOST_TARGET")
      ;;
    all)
      TARGETS=("${ALL_TARGETS[@]}")
      ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown mode: $MODE" >&2
      usage
      exit 1
      ;;
  esac
}

print_summary() {
  set +u
  local success_count="${#SUCCESS_TARGETS[@]}"
  local failed_count="${#FAILED_TARGETS[@]}"

  echo
  echo "Build summary"
  echo "-------------"
  echo "Successful: $success_count"
  for target in "${SUCCESS_TARGETS[@]}"; do
    echo "  - $target"
  done

  echo "Failed: $failed_count"
  for target in "${FAILED_TARGETS[@]}"; do
    echo "  - $target"
  done

  if (( failed_count > 0 )); then
    exit 1
  fi

  set -u
}

ensure_prerequisites
build_embedded_ui
select_targets

mkdir -p "$RELEASE_DIR"
mkdir -p "$BUILD_CACHE_DIR"
mkdir -p "$BUILD_TMP_DIR"

echo "Building releases into $RELEASE_DIR"
echo "Using GOCACHE=$BUILD_CACHE_DIR"
echo "Using GOTMPDIR=$BUILD_TMP_DIR"

declare -a SUCCESS_TARGETS=()
declare -a FAILED_TARGETS=()

for target in "${TARGETS[@]}"; do
  build_target "$target"
done

print_summary

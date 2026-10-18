#!/usr/bin/env bash
set -euo pipefail

if (( $# < 2 )); then
  echo "Usage: $0 DIRECTORY FILE [FILE ...]" >&2
  exit 2
fi
cd "$1"
shift
for artifact in "$@"; do
  if [[ ! -f "$artifact" || "$artifact" == */* || "$artifact" == -* || "$artifact" == *$'\n'* ]]; then
    echo "Expected a regular file name in DIRECTORY: $artifact" >&2
    exit 1
  fi
done
manifest_tmp="$(mktemp .SHA256SUMS.XXXXXX)"
trap 'rm -f "$manifest_tmp"' EXIT
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum -- "$@" > "$manifest_tmp"
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 -- "$@" > "$manifest_tmp"
else
  echo "Install sha256sum or shasum to generate SHA256SUMS." >&2
  exit 1
fi
mv "$manifest_tmp" SHA256SUMS

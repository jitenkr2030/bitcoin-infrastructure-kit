#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
arch="$(uname -m)"
case "$arch" in
  aarch64|arm64) binary="bin/btc-kit-linux-arm64" ;;
  x86_64|amd64) binary="bin/btc-kit-linux-amd64" ;;
  *)
    if command -v go >/dev/null 2>&1; then
      echo "No bundled binary for $arch; building from source..."
      exec ./scripts/run.sh
    fi
    echo "Unsupported CPU architecture: $arch. Install Go 1.23+ and run ./scripts/run.sh" >&2
    exit 1
    ;;
esac
if [[ ! -x "$binary" ]]; then
  if command -v go >/dev/null 2>&1; then exec ./scripts/run.sh; fi
  echo "Binary missing: $binary" >&2; exit 1
fi
exec "./$binary" "$@"

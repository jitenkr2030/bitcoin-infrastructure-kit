#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Install Go 1.23+ inside your Ubuntu/proot environment." >&2
  exit 1
fi

version="$(go env GOVERSION | sed 's/^go//')"
major="$(printf '%s' "$version" | cut -d. -f1)"
minor="$(printf '%s' "$version" | cut -d. -f2 | sed 's/[^0-9].*//')"
if [[ "$major" -lt 1 || ( "$major" -eq 1 && "$minor" -lt 23 ) ]]; then
  echo "Go 1.23+ required; found $(go version)" >&2
  exit 1
fi

echo "==> Downloading Go modules"
go mod download
echo "==> Running tests"
go test ./...
echo "==> Running go vet"
go vet ./...
echo "==> Building btc-kit"
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/btc-kit .
echo
echo "Build complete: $(pwd)/bin/btc-kit"
./bin/btc-kit help

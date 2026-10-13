#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p bin

go test ./...
go vet ./...

GOOS=linux GOARCH=arm64 \
  go build -trimpath -ldflags="-s -w" -o bin/btc-kit-linux-arm64 .

GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o bin/btc-kit-linux-amd64 .

GOOS="$(go env GOOS)" GOARCH="$(go env GOARCH)" \
  go build -trimpath -ldflags="-s -w" -o bin/btc-kit .

echo "Build successful:"
ls -lh bin/btc-kit bin/btc-kit-linux-arm64 bin/btc-kit-linux-amd64

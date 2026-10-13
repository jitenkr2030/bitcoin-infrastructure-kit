# Bitcoin Infrastructure Kit (`btc-kit`)

A Go-based, self-hostable Bitcoin developer toolkit for Linux, including Ubuntu under Termux/proot.

## Included in this release

- `btc-kit fees`: fetch recommended Bitcoin transaction fee rates from mempool.space.
- `btc-kit tx decode <hex>`: decode a raw Bitcoin transaction locally (no RPC credentials needed).
- `btc-kit node health`: query a Bitcoin Core node's blockchain status through authenticated RPC.
- `btc-kit serve`: run a small HTTP API with `/health`, `/ready`, and `/v1/fees`.
- Optional API key protection for `/v1/fees`, request timeouts, bounded response bodies, graceful shutdown.
- Configuration through environment variables; no secrets stored in source.
- Unit tests and Linux build scripts.

**Important:** This is a carefully structured MVP starter, not an audited financial product. The HTTP service does not provide wallet custody, signing, trading, or payment processing. Do not expose it publicly until you have reviewed the security checklist and placed it behind TLS/reverse proxy, monitoring, and appropriate access controls.

## Requirements

- Ubuntu/Debian or another Linux environment
- Go 1.23+ (Go 1.24+ recommended)
- Internet access for downloading Go modules and fetching public fee data
- Bitcoin Core is optional; only required for `node health`

## Quick start

```bash
unzip bitcoin-infrastructure-kit.zip
cd bitcoin-infrastructure-kit
./start.sh help
```

Prebuilt Linux ARM64 and AMD64 binaries are included, so common Ubuntu/proot setups can run the program without installing Go. The first API/fee request still needs internet access. To rebuild from source and run tests, install Go 1.23+ and use `./scripts/run.sh`.

```bash
./start.sh fees
./start.sh tx decode <RAW_TRANSACTION_HEX>
```

Start the API on loopback:

```bash
./bin/btc-kit serve
```

In another terminal:

```bash
curl -sS http://127.0.0.1:8080/health
curl -sS http://127.0.0.1:8080/ready
curl -sS http://127.0.0.1:8080/v1/fees
```

## Commands

```text
btc-kit help
btc-kit fees
btc-kit tx decode <raw-transaction-hex>
btc-kit node health
btc-kit serve
```

### Bitcoin Core RPC

Configure the URL and credentials via environment variables. Use a local node and do not expose RPC to the public internet.

```bash
export BTC_RPC_URL="http://127.0.0.1:8332/"
export BTC_RPC_USER="your_rpc_username"
export BTC_RPC_PASS="your_rpc_password"
./bin/btc-kit node health
```

Use a dedicated Bitcoin Core `rpcauth` user where possible. Do not commit credentials or put them into shell history on a shared system.

### HTTP API

Defaults to `127.0.0.1:8080`. To change the bind address:

```bash
export BTC_KIT_ADDR="127.0.0.1:8080"
./bin/btc-kit serve
```

For a non-loopback bind, an API key is mandatory:

```bash
export BTC_KIT_ADDR="0.0.0.0:8080"
export BTC_KIT_API_KEY="$(openssl rand -hex 32)"
./bin/btc-kit serve
```

Call the protected endpoint:

```bash
curl -H "Authorization: Bearer $BTC_KIT_API_KEY" \
  http://127.0.0.1:8080/v1/fees
```

Do not use the built-in HTTP server directly on the public internet. For public deployment, use a TLS reverse proxy, firewall, request limits, logs, monitoring, and a deployment-specific security review.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `BTC_KIT_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `BTC_KIT_API_KEY` | empty | If set, protects `/v1/fees`; non-loopback binds require it |
| `BTC_KIT_READ_TIMEOUT` | `5s` | HTTP read timeout |
| `BTC_KIT_WRITE_TIMEOUT` | `15s` | HTTP write timeout |
| `BTC_KIT_IDLE_TIMEOUT` | `60s` | HTTP idle timeout |
| `BTC_KIT_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `BTC_RPC_URL` | empty | Bitcoin Core RPC URL |
| `BTC_RPC_USER` | empty | Bitcoin Core RPC username |
| `BTC_RPC_PASS` | empty | Bitcoin Core RPC password |
| `BTC_KIT_FEE_URL` | `https://mempool.space/api/v1/fees/recommended` | Fee data provider endpoint |

## Build and test

```bash
go test ./...
go vet ./...
./scripts/build.sh
```

The project uses a Go module. First run needs internet access to download dependencies.

## Ubuntu under Termux/proot

Run inside the Ubuntu proot, not the Termux host:

```bash
sudo apt update
sudo apt install -y ca-certificates curl git unzip build-essential
```

Install Go 1.23+ using your preferred trusted distribution source, then unzip and run `./scripts/run.sh`.

If your proot environment is ARM64, Go must match the `linux/arm64` architecture. Check with:

```bash
uname -m
go version
```

## Security checklist before public production

- Put the service behind HTTPS/TLS and a reverse proxy.
- Keep the app bound to localhost behind that proxy when possible.
- Use a long random API key and rotate it.
- Restrict firewall ingress.
- Do not expose Bitcoin Core RPC publicly.
- Add persistent per-key quotas, abuse detection, structured logs, metrics, and billing before offering a paid API.
- Monitor the upstream provider and set service-level limits.
- Pin and review dependency updates; run `govulncheck` and a security review.
- Never request seed phrases or private keys.

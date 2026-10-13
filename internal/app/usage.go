package app

import (
	"fmt"
	"io"
)

func PrintUsage(out io.Writer) {
	fmt.Fprintln(out, `btc-kit — Bitcoin Infrastructure Kit

Usage:
  btc-kit fees
  btc-kit tx decode <raw-transaction-hex>
  btc-kit node health
  btc-kit serve

Environment:
  BTC_KIT_ADDR       HTTP listen address (default 127.0.0.1:8080)
  BTC_KIT_API_KEY    Protect /v1/fees; required for non-loopback binds
  BTC_RPC_URL        Bitcoin Core RPC URL (optional)
  BTC_RPC_USER       Bitcoin Core RPC username (optional)
  BTC_RPC_PASS       Bitcoin Core RPC password (optional)`)
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jitenkr2030/bitcoin-infrastructure-kit/internal/app"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, "btc-kit:", err)
		os.Exit(1)
	}
}

func execute() error {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		app.PrintUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "fees":
		return app.RunFees(context.Background(), os.Stdout)
	case "tx":
		if len(args) != 3 || args[1] != "decode" {
			return errors.New("usage: btc-kit tx decode <raw-transaction-hex>")
		}
		return app.DecodeTransaction(args[2], os.Stdout)
	case "node":
		if len(args) != 2 || args[1] != "health" {
			return errors.New("usage: btc-kit node health")
		}
		return app.RunNodeHealth(context.Background(), os.Stdout)
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return app.RunServer(ctx)
	default:
		app.PrintUsage(os.Stdout)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

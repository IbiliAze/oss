// Command vaultlet is the client for a vaultlet server.
//
// It is a thin shell around internal/adapters/cli: signal handling, and turning
// an error into an exit code. Everything else lives in the adapter, where it
// can be tested without a process.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/IbiliAze/vaultlet/internal/adapters/driving/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "vaultlet:", err)
		os.Exit(1)
	}
}

func run() error {
	// The first interrupt cancels ctx, unwinding open streams and deferred
	// cleanup; stop() restores the default handler so a second one kills us
	// outright, even if a command is ignoring cancellation.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := cli.Execute(ctx)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

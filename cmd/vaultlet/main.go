package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/IbiliAze/vaultlet/internal/adapters/driven/bitwarden"
	"github.com/IbiliAze/vaultlet/internal/adapters/driving/grpcserver"
	"github.com/IbiliAze/vaultlet/internal/config"
	"github.com/IbiliAze/vaultlet/internal/ports"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	store, err := newStore(cfg)
	if err != nil {
		return fmt.Errorf("open backend %q: %w", cfg.Backend, err)
	}
	if closer, ok := store.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	server := grpcserver.New(store)
	server.Listen(cfg.Listen)
	return nil
}

func newStore(cfg config.Config) (ports.SecretStore, error) {
	switch cfg.Backend {
	case "bitwarden":
		return bitwarden.New(cfg.Bitwarden)
	default:
		return nil, fmt.Errorf("unknown backend %q", cfg.Backend)
	}
}

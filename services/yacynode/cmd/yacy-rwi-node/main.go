package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "node terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if err := applog.Configure(os.Getenv); err != nil {
		return fmt.Errorf("configure logging: %w", err)
	}

	config, err := yacynodeassembly.LoadServiceConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("load node config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(ctx, config)
}

func serve(ctx context.Context, config yacynodeassembly.ServiceConfig) error {
	ops, err := servergroup.Listen(ctx, config.Serving.OpsAddr)
	if err != nil {
		return err
	}
	listeners, err := yacynodeassembly.ListenersFor(ctx, config)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	registry := opsmetrics.ProcessRegistry()
	opsMux := opsmetrics.MuxFor(registry)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return yacynodeassembly.RunService(ctx, config, listeners, registry)
	})
}

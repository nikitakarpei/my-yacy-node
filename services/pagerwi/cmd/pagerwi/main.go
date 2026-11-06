package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func main() {
	os.Exit(start())
}

func start() int {
	if err := applog.Configure(os.Getenv); err != nil {
		slog.ErrorContext(
			context.Background(),
			"pagerwi logging invalid",
			slog.Any("error", err),
		)
		return 2
	}

	cfg, err := pagerwiassembly.LoadServiceConfig(os.Getenv)
	if err != nil {
		slog.ErrorContext(
			context.Background(),
			"pagerwi config invalid",
			slog.Any("error", err),
		)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve(ctx, cfg); err != nil {
		slog.ErrorContext(ctx, "pagerwi failed", slog.Any("error", err))
		return 1
	}
	return 0
}

func serve(ctx context.Context, cfg pagerwiassembly.ServiceConfig) error {
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return err
	}
	listeners, err := pagerwiassembly.ListenersFor(ctx, cfg)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	registry := opsmetrics.ProcessRegistry()
	opsMux := opsmetrics.MuxFor(registry)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return pagerwiassembly.RunService(ctx, cfg, listeners, registry)
	})
}

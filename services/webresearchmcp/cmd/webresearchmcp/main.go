package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	webresearchmcpassembly "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/assembly"
)

func main() {
	os.Exit(start())
}

func start() int {
	if err := applog.Configure(os.Getenv); err != nil {
		slog.ErrorContext(
			context.Background(),
			"webresearchmcp logging invalid",
			slog.Any("error", err),
		)
		return 2
	}

	cfg, err := webresearchmcpassembly.LoadServiceConfig(os.Getenv)
	if err != nil {
		slog.ErrorContext(
			context.Background(),
			"webresearchmcp config invalid",
			slog.Any("error", err),
		)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve(ctx, cfg); err != nil {
		slog.ErrorContext(ctx, "webresearchmcp failed", slog.Any("error", err))
		return 1
	}
	return 0
}

func serve(ctx context.Context, cfg webresearchmcpassembly.ServiceConfig) error {
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return err
	}
	listeners, err := webresearchmcpassembly.ListenersFor(ctx, cfg)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	registry := opsmetrics.ProcessRegistry()
	opsMux := opsmetrics.MuxFor(registry)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return webresearchmcpassembly.RunService(ctx, cfg, listeners, registry)
	})
}

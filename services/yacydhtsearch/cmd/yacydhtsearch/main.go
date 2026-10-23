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
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "yacydhtsearch terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if err := applog.Configure(os.Getenv); err != nil {
		return err
	}
	cfg, err := yacydhtsearchassembly.LoadServiceConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(ctx, cfg)
}

func serve(ctx context.Context, cfg yacydhtsearchassembly.ServiceConfig) error {
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return err
	}
	listeners, err := yacydhtsearchassembly.ListenersFor(ctx, cfg)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	registry := opsmetrics.ProcessRegistry()
	opsMux := yacydhtsearchassembly.OpsMuxFor(cfg, registry)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return yacydhtsearchassembly.RunService(ctx, cfg, listeners, registry)
	})
}

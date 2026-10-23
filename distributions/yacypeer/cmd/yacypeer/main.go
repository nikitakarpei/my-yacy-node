package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "yacypeer terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if err := yacypeerassembly.ConfigureLogging(os.Getenv); err != nil {
		return err
	}
	cfg, err := yacypeerassembly.LoadServiceConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return err
	}
	return yacypeerassembly.RunService(ctx, cfg, ops, os.Getenv)
}

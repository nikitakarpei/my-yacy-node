package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func main() {
	os.Exit(start())
}

func start() int {
	if err := applog.Configure(os.Getenv); err != nil {
		slog.ErrorContext(
			context.Background(),
			"corpusmarkdown logging invalid",
			slog.Any("error", err),
		)
		return 2
	}

	cfg, err := LoadServiceConfig(os.Getenv)
	if err != nil {
		slog.ErrorContext(
			context.Background(),
			"corpusmarkdown config invalid",
			slog.Any("error", err),
		)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listeners, err := listenersFor(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "corpusmarkdown failed", slog.Any("error", err))
		return 1
	}

	if err := RunService(ctx, cfg, listeners); err != nil {
		slog.ErrorContext(ctx, "corpusmarkdown failed", slog.Any("error", err))
		return 1
	}
	return 0
}

func listenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	recall, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return Listeners{}, errors.Join(err, recall.Close())
	}
	return Listeners{Recall: recall, Ops: ops}, nil
}

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
	cfg, err := yacypeerassembly.LoadPeerConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	web, err := servergroup.Listen(ctx, cfg.WebAddr)
	if err != nil {
		return err
	}
	return yacypeerassembly.RunPeer(ctx, cfg, web, os.Getenv)
}

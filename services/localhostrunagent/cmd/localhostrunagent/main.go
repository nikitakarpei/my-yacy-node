package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "localhostrunagent terminated",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
}

func run() error {
	if err := applog.Configure(os.Getenv); err != nil {
		return err
	}

	configuration, err := LoadAgentConfiguration(os.Getenv)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return RunAgent(ctx, configuration)
}

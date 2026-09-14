package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/processrestartinterval"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
)

const shortestRestartInterval = 10 * time.Second

func main() {
	os.Exit(agentProcessExitStatus())
}

func agentProcessExitStatus() int {
	processStart := time.Now()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := run(ctx)
	if err == nil {
		return 0
	}

	slog.ErrorContext(ctx, "localhostrunagent terminated", slog.Any("error", err))
	processrestartinterval.HoldTheExit(ctx, processStart, shortestRestartInterval)

	return 1
}

func run(ctx context.Context) error {
	if err := applog.Configure(os.Getenv); err != nil {
		return err
	}

	configuration, err := LoadAgentConfiguration(os.Getenv)
	if err != nil {
		return err
	}

	return RunAgent(ctx, configuration)
}

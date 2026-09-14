package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processentrypoint"
)

func main() {
	ctx := context.Background()

	if err := applog.Configure(os.Getenv); err != nil {
		slog.ErrorContext(ctx, "entrypoint terminated", slog.Any("error", err))
		os.Exit(1)
	}

	exitStatus, err := processentrypoint.Run(ctx, os.Args[1:], os.Getenv, os.Environ())
	if err != nil {
		slog.ErrorContext(ctx, "entrypoint terminated", slog.Any("error", err))
		os.Exit(1)
	}

	os.Exit(exitStatus)
}

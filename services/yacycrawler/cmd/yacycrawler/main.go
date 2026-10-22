package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "crawler terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if err := applog.Configure(os.Getenv); err != nil {
		return err
	}
	cfg, err := LoadServiceConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listeners, err := listenersFor(ctx, cfg)
	if err != nil {
		return err
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "yacycrawler_info",
			Help: "YaCy crawler application identity.",
		}, func() float64 { return 1 }),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return RunService(ctx, cfg, listeners, registry)
}

func listenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return Listeners{}, err
	}
	if !cfg.ServesAPI() {
		return Listeners{Ops: ops}, nil
	}
	api, err := servergroup.Listen(ctx, cfg.APIAddr)
	if err != nil {
		_ = ops.Close()
		return Listeners{}, err
	}

	return Listeners{Ops: ops, API: api}, nil
}

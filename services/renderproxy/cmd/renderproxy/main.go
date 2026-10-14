package main

import (
	"context"
	"errors"
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
		slog.ErrorContext(context.Background(), "renderproxy terminated", slog.Any("error", err))
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
			Name: "renderproxy_info",
			Help: "Render proxy application identity.",
		}, func() float64 { return 1 }),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return RunService(ctx, cfg, listeners, registry)
}

func listenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	proxy, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return Listeners{}, errors.Join(err, proxy.Close())
	}

	return Listeners{Proxy: proxy, Ops: ops}, nil
}

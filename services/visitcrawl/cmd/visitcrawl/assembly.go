package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitmetrics"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	msgServiceStarted  = "visitcrawl started"
	msgServiceStopped  = "visitcrawl stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	metrics := visitmetrics.New(registry)
	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:       cfg.NATSURL,
		OrdersSubject: cfg.OrdersSubject,
	})
	if err != nil {
		return fmt.Errorf("open crawl order broker: %w", err)
	}
	defer broker.Close()

	placement := visitintake.NewBoundedPlacement(
		broker.Orders.Place, metrics, cfg.OrderTimeout, cfg.MaxInFlight,
	)

	mux := http.NewServeMux()
	visitintake.MountVisitIntake(mux, placement, cfg.CrawlProfile, metrics, cfg.LinkSecret)

	publicServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           http.MaxBytesHandler(mux, cfg.MaxBodyBytes),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.OrdersSubject),
		slog.String("listenAddr", cfg.ListenAddr),
		slog.String("opsAddr", cfg.OpsAddr),
	)

	err = servergroup.Run(ctx, opsShutdownLimit, []servergroup.NamedServer{
		{Name: "intake", Server: publicServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	crawlorderplacementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderplacementobservers/applog"
	crawlorderplacementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderplacementobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
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
	crawlOrderPlacementObservers := visitintake.CrawlOrderPlacementObservers{
		crawlorderplacementobserversapplog.CrawlOrderPlacementLog{},
		crawlorderplacementobserversprometheus.New(registry),
	}
	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:            cfg.CrawlNATSURL,
		CrawlOrdersSubject: cfg.CrawlOrdersSubject,
	})
	if err != nil {
		return fmt.Errorf("open crawl order broker: %w", err)
	}
	defer broker.Close()

	placementAttempts := visitintake.NewCrawlOrderPlacementAttempts(
		broker.Orders, crawlOrderPlacementObservers, cfg.OrderTimeout, cfg.MaxInFlight,
	)

	mux := http.NewServeMux()
	visitintake.MountVisitIntake(
		mux, placementAttempts, cfg.CrawlProfile, cfg.LinkSecret,
	)

	publicServer := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: httpobservation.NewHandler(
			http.MaxBytesHandler(mux, cfg.MaxBodyBytes),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "visitcrawl"),
		),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
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

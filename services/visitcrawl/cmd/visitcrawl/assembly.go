package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	backgroundcrawlorderplacementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/backgroundcrawlorderplacementobservers/applog"
	backgroundcrawlorderplacementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/backgroundcrawlorderplacementobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderplacers/background"
	crawlorderpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderpublicationobservers/applog"
	crawlorderpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderpublicationobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	msgServiceStarted  = "visitcrawl started"
	msgServiceStopped  = "visitcrawl stopped"
)

type Listeners struct {
	Intake net.Listener
	Ops    net.Listener
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:            cfg.CrawlNATSURL,
		CrawlOrdersSubject: cfg.CrawlOrdersSubject,
	}, crawlorderbroker.CrawlOrderPublicationObservers{
		crawlorderpublicationobserversapplog.CrawlOrderPublicationLog{},
		crawlorderpublicationobserversprometheus.New(registry),
	})
	if err != nil {
		return fmt.Errorf("open crawl order broker: %w", err)
	}
	defer broker.Close()

	placer := background.New(
		broker.OrderPlacer,
		background.BackgroundCrawlOrderPlacementObservers{
			backgroundcrawlorderplacementobserversapplog.BackgroundCrawlOrderPlacementLog{},
			backgroundcrawlorderplacementobserversprometheus.New(registry),
		},
		cfg.OrderTimeout,
		cfg.MaxInFlight,
	)

	mux := http.NewServeMux()
	visitintake.MountVisitIntake(
		mux,
		placer,
		cfg.CrawlProfile,
		cfg.LinkSecret,
	)

	publicServer := &http.Server{
		Handler: http.MaxBytesHandler(
			httpobservation.NewHandler(
				mux,
				httpaccesslog.New(),
				httpmetrics.NewEndpointMetrics(registry, "visitcrawl"),
			),
			cfg.MaxBodyBytes,
		),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
	opsServer := &http.Server{
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.String("listenAddr", listeners.Intake.Addr().String()),
		slog.String("opsAddr", listeners.Ops.Addr().String()),
	)

	err = servergroup.Run(ctx, opsShutdownLimit, []servergroup.NamedServer{
		{Name: "intake", Listener: listeners.Intake, Server: publicServer},
		{Name: "ops", Listener: listeners.Ops, Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

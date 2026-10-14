package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	pageofferpublishersjetstream "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/pageofferpublishers/jetstream"
	scrapeintakepkg "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeintake"
	scrapeintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeintakeobservers/applog"
	scrapeintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeintakeobservers/prometheus"
	scrapeoutcomefeedobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeoutcomefeedobservers/applog"
	scrapeoutcomefeedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeoutcomefeedobservers/prometheus"
	scrapeoutcomefeedsnats "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeoutcomefeeds/nats"
	scrapestreamsjetstream "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapestreams/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	opsReadHeaderLimit    = 10 * time.Second
	opsShutdownLimit      = 15 * time.Second
	metricNamespace       = "pagescrape"
	pageOfferBacklogLimit = 1000
)

type Listeners struct {
	Ops net.Listener
}

func RunService(ctx context.Context, cfg ServiceConfig, listeners Listeners) error {
	broker, connection, err := jetstreamconnect.Open(cfg.ScrapeNATSURL)
	if err != nil {
		return err
	}
	defer connection.Close()

	scrapeRequests, err := scrapestreamsjetstream.CreateScrapeRequestsStream(
		ctx, broker, scrapestreamsjetstream.ScrapeRequestsStreamLimits{
			MaxMsgs: cfg.ScrapeRequestsKept,
		},
	)
	if err != nil {
		return err
	}
	pageOffers, err := scrapestreamsjetstream.CreateScrapePageOffersStream(
		ctx, broker, scrapestreamsjetstream.ScrapePageOffersStreamLimits{
			MaxBytes: cfg.PageOfferMaxBytes,
		},
	)
	if err != nil {
		return err
	}
	consumer, err := scrapeRequestConsumerFor(ctx, scrapeRequests, cfg)
	if err != nil {
		return err
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: metricNamespace + "_info",
			Help: "Page scrape application identity.",
		}, func() float64 { return 1 }),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	outcomeFeed := scrapeoutcomefeedsnats.NewScrapeOutcomeFeed(
		connection,
		scrapeoutcomefeedsnats.ScrapeOutcomeFeedObservers{
			scrapeoutcomefeedobserversapplog.ScrapeOutcomeFeedLog{},
			scrapeoutcomefeedobserversprometheus.New(registry),
		},
	)
	intake := scrapeintakepkg.NewScrapeRequestConsumer(
		consumer,
		redirectfollowingfetch.New(
			pagefetchershttp.New(
				cfg.ProxyURL,
				cfg.ProxyDialMode,
				cfg.UserAgent,
				cfg.MaxBodyBytes,
				cfg.FetchDeadline,
			),
			cfg.MaxRedirectHops,
		),
		pageofferpublishersjetstream.NewPageOfferPublisher(broker),
		outcomeFeed,
		scrapeintakepkg.ScrapeIntakeObservers{
			scrapeintakeobserversapplog.ScrapeIntakeLog{},
			scrapeintakeobserversprometheus.New(registry),
		},
		consumerbacklog.New(
			pageOffers, pageOfferBacklogLimit, wallclock.Clock{}, registry, metricNamespace,
		),
		cfg.ScrapeDeferralWindow,
		cfg.ScrapeIntakeConcurrency,
		wallclock.Clock{},
	)

	opsServer := &http.Server{
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "pagescrape started",
		slog.String("ops", listeners.Ops.Addr().String()),
		slog.String("durable", cfg.ScrapeRequestDurable),
		slog.Int("scrapeIntakeConcurrency", cfg.ScrapeIntakeConcurrency),
		slog.Duration("scrapeDeferralWindow", cfg.ScrapeDeferralWindow),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Listener: listeners.Ops, Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run scrape request consumer: %w", err)
			}
			return nil
		},
		outcomeFeed.CarryIntakeReceipts,
	)
	slog.InfoContext(ctx, "pagescrape stopped")
	return err
}

func scrapeRequestConsumerFor(
	ctx context.Context,
	scrapeRequests jetstream.Stream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	consumer, err := scrapeRequests.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.ScrapeRequestDurable,
		FilterSubject: pagescrapecontract.ScrapeRequestSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: int(cfg.ScrapeRequestsKept),
	})
	if err != nil {
		return nil, fmt.Errorf("create scrape request consumer: %w", err)
	}
	return consumer, nil
}

package pagescrapeassembly

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

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
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	shutdownLimit         = 15 * time.Second
	metricNamespace       = "pagescrape"
	pageOfferBacklogLimit = 1000
)

type Listeners struct{}

func ListenersFor(context.Context, ServiceConfig) (Listeners, error) {
	return Listeners{}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	_ Listeners,
	registry *prometheus.Registry,
) error {
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

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: metricNamespace + "_info",
		Help: "Page scrape application identity.",
	}, func() float64 { return 1 }))
	outcomeFeed := scrapeoutcomefeedsnats.NewScrapeOutcomeFeed(
		connection,
		scrapeoutcomefeedsnats.ScrapeOutcomeFeedObservers{
			scrapeoutcomefeedobserversapplog.ScrapeOutcomeFeedLog{},
			scrapeoutcomefeedobserversprometheus.New(registry),
		},
	)
	intake := scrapeintakepkg.NewScrapeRequestConsumer(
		consumer,
		pageFetchFor(cfg),
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

	slog.InfoContext(ctx, "pagescrape started",
		slog.String("durable", cfg.ScrapeRequestDurable),
		slog.Int("scrapeIntakeConcurrency", cfg.ScrapeIntakeConcurrency),
		slog.Duration("scrapeDeferralWindow", cfg.ScrapeDeferralWindow),
	)
	err = servergroup.Run(ctx, shutdownLimit, nil,
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

func pageFetchFor(cfg ServiceConfig) *redirectfollowingfetch.RedirectFollowingPageFetch {
	return redirectfollowingfetch.New(
		pagefetchershttp.New(
			cfg.ProxyURL,
			cfg.ProxyDialMode,
			cfg.UserAgent,
			cfg.MaxBodyBytes,
			cfg.FetchDeadline,
		),
		cfg.MaxRedirectHops,
	)
}

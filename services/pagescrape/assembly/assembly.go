package pagescrapeassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	offerattestersunattested "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/offerattesters/unattested"
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
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	shutdownLimit         = 15 * time.Second
	metricNamespace       = "pagescrape"
	pageOfferBacklogLimit = 1000
)

type Dial = func(ctx context.Context, network, address string) (net.Conn, error)

type Listeners struct{}

func ListenersFor(context.Context, ServiceConfig) (Listeners, error) {
	return Listeners{}, nil
}

func (Listeners) Close() error {
	return nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	_ Listeners,
	dial Dial,
	brokerAccess jetstreamconnect.BrokerAccess,
	intakeAdmission pullintake.Admission,
	registry *prometheus.Registry,
	health *servicehealth.Health,
) error {
	broker, connection, err := jetstreamconnect.Open(
		cfg.ScrapeNATSURL, dial, health, brokerAccess,
	)
	if err != nil {
		return err
	}
	defer connection.Close()

	streams, err := attachScrapeStreams(ctx, broker, cfg)
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
		streams.scrapeRequestConsumer,
		pageFetchFor(cfg, dial),
		offerattestersunattested.New(wallclock.Clock{}),
		pageofferpublishersjetstream.NewPageOfferPublisher(broker),
		outcomeFeed,
		scrapeintakepkg.ScrapeIntakeObservers{
			scrapeintakeobserversapplog.ScrapeIntakeLog{},
			scrapeintakeobserversprometheus.New(registry),
		},
		pullintake.AdmissionOfEach(
			consumerbacklog.New(
				streams.pageOffers,
				pageOfferBacklogLimit,
				wallclock.Clock{},
				registry,
				metricNamespace,
			),
			intakeAdmission,
		),
		cfg.ScrapeDeferralWindow,
		cfg.ScrapeIntakeConcurrency,
		wallclock.Clock{},
	)

	receiptCarrier, err := outcomeFeed.StartCarryingIntakeReceipts(ctx)
	if err != nil {
		return err
	}
	health.MarkStarted()
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
		receiptCarrier.Run,
	)
	slog.InfoContext(ctx, "pagescrape stopped")
	return err
}

type scrapeStreams struct {
	scrapeRequestConsumer jetstream.Consumer
	pageOffers            jetstream.Stream
}

func attachScrapeStreams(
	ctx context.Context,
	broker jetstream.JetStream,
	cfg ServiceConfig,
) (scrapeStreams, error) {
	if cfg.ProvisionsStreams {
		return provisionScrapeStreams(ctx, broker, cfg)
	}
	return openScrapeStreams(ctx, broker, cfg.ScrapeRequestDurable)
}

func provisionScrapeStreams(
	ctx context.Context,
	broker jetstream.JetStream,
	cfg ServiceConfig,
) (scrapeStreams, error) {
	scrapeRequests, err := scrapestreamsjetstream.CreateScrapeRequestsStream(
		ctx, broker, scrapestreamsjetstream.ScrapeRequestsStreamLimits{
			MaxMsgs: int64(cfg.ScrapeRequestsKept),
		},
	)
	if err != nil {
		return scrapeStreams{}, err
	}
	pageOffers, err := scrapestreamsjetstream.CreateScrapePageOffersStream(
		ctx, broker, scrapestreamsjetstream.ScrapePageOffersStreamLimits{
			MaxBytes: cfg.PageOfferMaxBytes,
		},
	)
	if err != nil {
		return scrapeStreams{}, err
	}
	consumer, err := createScrapeRequestConsumer(ctx, scrapeRequests, cfg)
	if err != nil {
		return scrapeStreams{}, err
	}
	return scrapeStreams{scrapeRequestConsumer: consumer, pageOffers: pageOffers}, nil
}

func createScrapeRequestConsumer(
	ctx context.Context,
	scrapeRequests jetstream.Stream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	consumer, err := scrapeRequests.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.ScrapeRequestDurable,
		FilterSubject: pagescrapecontract.ScrapeRequestSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.ScrapeRequestsKept,
	})
	if err != nil {
		return nil, fmt.Errorf("create scrape request consumer: %w", err)
	}
	return consumer, nil
}

func openScrapeStreams(
	ctx context.Context,
	broker jetstream.JetStream,
	scrapeRequestDurable string,
) (scrapeStreams, error) {
	scrapeRequests, err := scrapestreamsjetstream.OpenScrapeRequestsStream(ctx, broker)
	if err != nil {
		return scrapeStreams{}, err
	}
	pageOffers, err := scrapestreamsjetstream.OpenScrapePageOffersStream(ctx, broker)
	if err != nil {
		return scrapeStreams{}, err
	}
	consumer, err := scrapeRequests.Consumer(ctx, scrapeRequestDurable)
	if err != nil {
		return scrapeStreams{}, fmt.Errorf("open scrape request consumer: %w", err)
	}
	return scrapeStreams{scrapeRequestConsumer: consumer, pageOffers: pageOffers}, nil
}

func pageFetchFor(cfg ServiceConfig, dial Dial) *redirectfollowingfetch.RedirectFollowingPageFetch {
	return redirectfollowingfetch.New(
		pagefetchershttp.New(
			pagefetchershttp.Proxy{URL: cfg.ProxyURL, DialMode: cfg.ProxyDialMode, Dial: dial},
			cfg.UserAgent,
			cfg.MaxBodyBytes,
			cfg.FetchDeadline,
		),
		cfg.MaxRedirectHops,
	)
}

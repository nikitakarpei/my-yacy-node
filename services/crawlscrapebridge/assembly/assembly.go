package crawlscrapebridgeassembly

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	crawledpageopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/crawledpageopeningobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/crawledpagerelay"
	scraperequestobserversapplog "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/scraperequestobservers/applog"
	scraperequestobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/scraperequestobservers/prometheus"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/scraperequests/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
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
	scrapeRequestBroker, scrapeRequestConnection, err := jetstreamconnect.Open(
		cfg.ScrapeRequestNATSURL,
	)
	if err != nil {
		return err
	}
	defer scrapeRequestConnection.Close()
	crawledPageOpening := streamopening.Start(
		ctx,
		crawledPageOpenerFor(cfg),
		wallclock.Clock{},
		crawledpageopeningobserversapplog.CrawledPageOpeningLog{},
	)
	defer crawledPageOpening.Close()

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "crawlscrapebridge_info",
		Help: "Crawl scrape bridge application identity.",
	}, func() float64 { return 1 }))
	relay := crawledpagerelay.NewCrawledPageRelay(
		scraperequestsjetstream.NewScrapeRequests(
			scrapeRequestBroker,
			scraperequestsjetstream.ScrapeRequestPublicationObservers{
				scraperequestobserversapplog.ScrapeRequestLog{},
				scraperequestobserversprometheus.New(registry),
			},
		),
		cfg.CrawledPageRelayConcurrency,
	)

	slog.InfoContext(ctx, "crawlscrapebridge started",
		slog.String("durable", cfg.CrawledPageDurable),
		slog.Int("crawledPageRelayConcurrency", cfg.CrawledPageRelayConcurrency),
	)
	err = relayCrawledPages(ctx, crawledPageOpening, relay)
	slog.InfoContext(ctx, "crawlscrapebridge stopped")
	return err
}

func relayCrawledPages(
	ctx context.Context,
	crawledPageOpening *streamopening.Opening[crawledPages],
	relay *crawledpagerelay.CrawledPageRelay,
) error {
	pages, opened := crawledPageOpening.Opened(ctx)
	if !opened {
		return nil
	}
	defer pages.Close()
	if err := relay.Run(ctx, pullintake.PendingMessagesOf(pages.consumer)); err != nil {
		return fmt.Errorf("relay crawled pages: %w", err)
	}
	return nil
}

type crawledPages struct {
	consumer   jetstream.Consumer
	connection *nats.Conn
}

func (pages crawledPages) Close() {
	pages.connection.Close()
}

func crawledPageOpenerFor(cfg ServiceConfig) func(ctx context.Context) (crawledPages, error) {
	return func(ctx context.Context) (crawledPages, error) {
		js, connection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
		if err != nil {
			return crawledPages{}, err
		}
		consumer, err := crawledPageConsumerFor(ctx, js, cfg)
		if err != nil {
			connection.Close()
			return crawledPages{}, err
		}
		return crawledPages{consumer: consumer, connection: connection}, nil
	}
}

func crawledPageConsumerFor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := js.Stream(ctx, yacycrawlcontract.CrawledPagesStreamName)
	if err != nil {
		return nil, fmt.Errorf("lookup crawled pages stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.CrawledPageDurable,
		FilterSubject: yacycrawlcontract.IndexablePageSubject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.CrawledPageRelayConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create crawled page consumer: %w", err)
	}
	return consumer, nil
}

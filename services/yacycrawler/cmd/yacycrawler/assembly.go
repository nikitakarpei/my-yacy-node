package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	acceptedordersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
	hostpageallowancesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/hostpageallowances/jetstream"
	pendingvisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisits/jetstream"
	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/alwaysdue"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/scraperequests/jetstream"
	visitclaimsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	msgServiceStarted  = "crawler started"
	msgServiceStopped  = "crawler stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	metrics := progressobserversprometheus.New(registry)
	brokers, err := openCrawlBrokers(cfg)
	if err != nil {
		return err
	}
	defer brokers.Close()
	if err := ensureNATSState(ctx, brokers.crawl, cfg); err != nil {
		return err
	}
	orders, err := buildOrderConsumer(ctx, brokers.crawl, cfg, metrics)
	if err != nil {
		return err
	}
	visits, err := buildVisitConsumer(ctx, brokers, cfg, metrics)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer(cfg, registry)}},
		orders.Run,
		visits.Run,
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

type crawlBrokers struct {
	crawl          jetstream.JetStream
	scrapeRequests jetstream.JetStream
	connections    []io.Closer
}

func openCrawlBrokers(cfg ServiceConfig) (crawlBrokers, error) {
	crawl, crawlConnection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return crawlBrokers{}, err
	}
	scrapeRequests, scrapeRequestConnection, err := jetstreamconnect.Open(
		cfg.ScrapeRequestNATSURL,
	)
	if err != nil {
		_ = crawlConnection.Close()
		return crawlBrokers{}, err
	}
	return crawlBrokers{
		crawl:          crawl,
		scrapeRequests: scrapeRequests,
		connections:    []io.Closer{crawlConnection, scrapeRequestConnection},
	}, nil
}

func (b crawlBrokers) Close() {
	for _, connection := range b.connections {
		_ = connection.Close()
	}
}

func opsServer(cfg ServiceConfig, registry *prometheus.Registry) *http.Server {
	return &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
}

func ensureNATSState(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if err := ensureOrdersStream(ctx, js, cfg); err != nil {
		return err
	}
	if err := ensureFrontierStream(ctx, js, cfg); err != nil {
		return err
	}
	if err := visitclaimsjetstream.Ensure(ctx, js, cfg.VisitClaimBucketSpec()); err != nil {
		return err
	}
	if err := hostpageallowancesjetstream.Ensure(
		ctx, js, cfg.HostPageAllowanceBucketSpec(),
	); err != nil {
		return err
	}
	if err := acceptedordersjetstream.Ensure(ctx, js, cfg.AcceptedOrderBucketSpec()); err != nil {
		return err
	}
	return ensurePageVisitBucket(ctx, js, cfg)
}

func ensureOrdersStream(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{cfg.CrawlOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		return fmt.Errorf("ensure orders stream: %w", err)
	}
	return nil
}

func ensureFrontierStream(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       pendingvisit.StreamName,
		Subjects:   []string{pendingvisit.Subject},
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: cfg.PendingVisitDuplicateWindow(),
	}); err != nil {
		return fmt.Errorf("ensure frontier stream: %w", err)
	}
	return nil
}

func ensurePageVisitBucket(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) error {
	if !cfg.SuppressesRecrawl() {
		return nil
	}
	return dueaftergrace.Ensure(ctx, js, cfg.PageVisitBucketSpec())
}

func buildOrderConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	metrics *progressobserversprometheus.CrawlMetrics,
) (*orderintake.OrderConsumer, error) {
	consumer, err := ordersConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	orders, err := acceptedOrders(ctx, js)
	if err != nil {
		return nil, err
	}
	return orderintake.NewOrderConsumer(
		consumer,
		orders,
		pendingvisitsjetstream.New(js),
		metrics,
		cfg.OrderIntakeConcurrency(),
	), nil
}

func ordersConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	consumer, err := js.CreateOrUpdateConsumer(ctx, yacycrawlcontract.OrdersStreamName,
		jetstream.ConsumerConfig{
			Durable:       cfg.CrawlOrdersDurable,
			FilterSubject: cfg.CrawlOrdersSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       cfg.CrawlOrdersAckWait(),
		})
	if err != nil {
		return nil, fmt.Errorf("create orders consumer: %w", err)
	}
	return consumer, nil
}

func acceptedOrders(
	ctx context.Context,
	js jetstream.JetStream,
) (*acceptedordersjetstream.Orders, error) {
	bucket, err := js.KeyValue(ctx, acceptedordersjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open accepted order bucket: %w", err)
	}
	return acceptedordersjetstream.New(bucket), nil
}

func buildVisitConsumer(
	ctx context.Context,
	brokers crawlBrokers,
	cfg ServiceConfig,
	metrics *progressobserversprometheus.CrawlMetrics,
) (*visitintake.VisitConsumer, error) {
	consumer, err := visitsConsumer(ctx, brokers.crawl, cfg)
	if err != nil {
		return nil, err
	}
	claims, err := visitClaims(ctx, brokers.crawl, cfg)
	if err != nil {
		return nil, err
	}
	ledger, err := visitLedger(ctx, brokers.crawl, cfg, claims)
	if err != nil {
		return nil, err
	}
	orders, err := acceptedOrders(ctx, brokers.crawl)
	if err != nil {
		return nil, err
	}
	visitorFor, err := buildVisitorFor(ctx, brokers, cfg, metrics)
	if err != nil {
		return nil, err
	}
	return visitintake.NewVisitConsumer(
		consumer,
		claims,
		ledger,
		orders,
		pendingvisitsjetstream.New(brokers.crawl),
		visitorFor,
		metrics,
		cfg.FetchConcurrency,
	), nil
}

func visitsConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	consumer, err := js.CreateOrUpdateConsumer(ctx, pendingvisit.StreamName,
		jetstream.ConsumerConfig{
			Durable:       cfg.PendingVisitDurable,
			FilterSubject: pendingvisit.Subject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       cfg.PendingVisitAckWait(),
		})
	if err != nil {
		return nil, fmt.Errorf("create pending visit consumer: %w", err)
	}
	return consumer, nil
}

func visitClaims(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (*visitclaimsjetstream.Claims, error) {
	bucket, err := js.KeyValue(ctx, visitclaimsjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open visit claim bucket: %w", err)
	}
	return visitclaimsjetstream.New(bucket, cfg.VisitClaimLimits()), nil
}

func visitLedger(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	claims *visitclaimsjetstream.Claims,
) (*visitallowance.Ledger, error) {
	bucket, err := js.KeyValue(ctx, hostpageallowancesjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open host page allowance bucket: %w", err)
	}
	return visitallowance.New(
		claims,
		hostpageallowancesjetstream.New(bucket),
		cfg.FetchRetryBounds(),
	), nil
}

func buildVisitorFor(
	ctx context.Context,
	brokers crawlBrokers,
	cfg ServiceConfig,
	metrics *progressobserversprometheus.CrawlMetrics,
) (pagevisit.VisitorFor, error) {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	recrawl, err := recrawlRule(ctx, brokers.crawl, cfg)
	if err != nil {
		return nil, err
	}
	return pagevisit.New(
		fetch,
		wallclock.Clock{},
		recrawl,
		metrics,
		scraperequestsjetstream.New(brokers.scrapeRequests),
	), nil
}

func recrawlRule(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (pagevisit.RecrawlRule, error) {
	if !cfg.SuppressesRecrawl() {
		return alwaysdue.AlwaysDue{}, nil
	}
	bucket, err := js.KeyValue(ctx, dueaftergrace.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page visit bucket: %w", err)
	}
	return dueaftergrace.New(bucket, wallclock.Clock{}, cfg.RecrawlGrace), nil
}

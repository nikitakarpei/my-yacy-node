package main

import (
	"context"
	"fmt"
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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchtiming"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
	pendingvisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisits/jetstream"
	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/alwaysdue"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/scraperequests/jetstream"
	visitclaimsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const (
	fetchRetryLimit        = 3
	fetchRetryFloor        = 500 * time.Millisecond
	fetchRetryCeiling      = 30 * time.Second
	maxDeferPerURL         = 3
	opsReadHeaderLimit     = 10 * time.Second
	opsShutdownLimit       = 15 * time.Second
	ordersAckWait          = 30 * time.Second
	orderIntakeConcurrency = 4
	visitAckWaitOfDeadline = 3
	pendingVisitDuplicates = 2 * time.Minute
	msgServiceStarted      = "crawler started"
	msgServiceStopped      = "crawler stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	metrics := progressobserversprometheus.New(registry)
	js, conn, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	scrapeRequestJetStream, scrapeRequestConnection, err := jetstreamconnect.Open(
		cfg.ScrapeRequestNATSURL,
	)
	if err != nil {
		return err
	}
	defer func() { _ = scrapeRequestConnection.Close() }()
	if err := ensureNATSState(ctx, js, cfg); err != nil {
		return err
	}

	orders, err := buildOrderConsumer(ctx, js, cfg, metrics)
	if err != nil {
		return err
	}
	visits, err := buildVisitConsumer(ctx, js, scrapeRequestJetStream, cfg, metrics)
	if err != nil {
		return err
	}

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
	)

	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		orders.Run,
		visits.Run,
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func ensureNATSState(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if err := ensureOrdersStream(ctx, js, cfg.CrawlOrdersSubject); err != nil {
		return err
	}
	if err := ensureFrontierStream(ctx, js); err != nil {
		return err
	}
	if err := visitclaimsjetstream.Ensure(ctx, js, cfg.VisitClaimBucketSpec()); err != nil {
		return err
	}
	if err := acceptedordersjetstream.Ensure(ctx, js, cfg.AcceptedOrderBucketSpec()); err != nil {
		return err
	}
	return ensurePageVisitBucket(ctx, js, cfg)
}

func ensureOrdersStream(ctx context.Context, js jetstream.JetStream, subject string) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{subject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		return fmt.Errorf("ensure orders stream: %w", err)
	}
	return nil
}

func ensureFrontierStream(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       pendingvisit.StreamName,
		Subjects:   []string{pendingvisit.Subject},
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: pendingVisitDuplicates,
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
	if cfg.RecrawlGrace <= 0 {
		return nil
	}
	if err := dueaftergrace.Ensure(ctx, js, cfg.PageVisitBucketSpec()); err != nil {
		return fmt.Errorf("ensure page visit bucket: %w", err)
	}
	return nil
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
		orderIntakeConcurrency,
	), nil
}

func buildVisitConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	scrapeRequestJetStream jetstream.JetStream,
	cfg ServiceConfig,
	metrics *progressobserversprometheus.CrawlMetrics,
) (*visitintake.VisitConsumer, error) {
	consumer, err := visitsConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	claims, err := visitClaims(ctx, js)
	if err != nil {
		return nil, err
	}
	orders, err := acceptedOrders(ctx, js)
	if err != nil {
		return nil, err
	}
	visitorFor, err := buildVisitorFor(ctx, js, scrapeRequestJetStream, cfg, metrics)
	if err != nil {
		return nil, err
	}
	return visitintake.NewVisitConsumer(
		consumer,
		claims,
		orders,
		pendingvisitsjetstream.New(js),
		visitorFor,
		metrics,
		retrydelay.Bounds{Floor: fetchRetryFloor, Ceiling: fetchRetryCeiling},
		cfg.FetchConcurrency,
	), nil
}

func visitClaims(
	ctx context.Context,
	js jetstream.JetStream,
) (*visitclaimsjetstream.Claims, error) {
	bucket, err := js.KeyValue(ctx, visitclaimsjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open visit claim bucket: %w", err)
	}
	return visitclaimsjetstream.New(bucket, visitclaimsjetstream.Config{
		MaxDeferralsPerURL: maxDeferPerURL,
		MaxAttemptsPerURL:  fetchRetryLimit,
	}), nil
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

func buildVisitorFor(
	ctx context.Context,
	js jetstream.JetStream,
	scrapeRequestJetStream jetstream.JetStream,
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
	recrawl, err := recrawlRule(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	return pagevisit.New(
		fetchtiming.New(metrics, wallclock.Clock{}, fetch),
		recrawl,
		metrics,
		scraperequestsjetstream.New(scrapeRequestJetStream),
	), nil
}

func recrawlRule(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (pagevisit.RecrawlRule, error) {
	if cfg.RecrawlGrace <= 0 {
		return alwaysdue.AlwaysDue{}, nil
	}
	bucket, err := js.KeyValue(ctx, dueaftergrace.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page visit bucket: %w", err)
	}
	return dueaftergrace.New(bucket, wallclock.Clock{}, cfg.RecrawlGrace), nil
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
			AckWait:       ordersAckWait,
		})
	if err != nil {
		return nil, fmt.Errorf("create orders consumer: %w", err)
	}
	return consumer, nil
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
			AckWait:       cfg.FetchDeadline * visitAckWaitOfDeadline,
		})
	if err != nil {
		return nil, fmt.Errorf("create pending visit consumer: %w", err)
	}
	return consumer, nil
}

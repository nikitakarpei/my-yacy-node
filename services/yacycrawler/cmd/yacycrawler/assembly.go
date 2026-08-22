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

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchtiming"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordertraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/reachedpagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/redirectrecording"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	crawloutcomereceiversgrpc "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawloutcomereceivers/grpc"
	disposedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/disposedpages/jetstream"
	orderreceiversjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/orderreceivers/jetstream"
	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
	reachedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/reachedpages/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/alwaysdue"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	redirectresolversjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolvers/jetstream"
)

const (
	fetchRetryLimit    = 3
	fetchRetryFloor    = 500 * time.Millisecond
	fetchRetryCeiling  = 30 * time.Second
	maxDeferPerURL     = 3
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	ordersAckWait      = 30 * time.Second
	msgServiceStarted  = "crawler started"
	msgServiceStopped  = "crawler stopped"
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
	if err := ensureNATSState(ctx, js, cfg); err != nil {
		return err
	}

	consumer, err := ordersConsumer(ctx, js, cfg)
	if err != nil {
		return err
	}
	orderReceiver, err := orderreceiversjetstream.NewOrderReceiver(ctx, consumer)
	if err != nil {
		return fmt.Errorf("start order receiver: %w", err)
	}

	disposedPages, err := openDisposedPages(ctx, js)
	if err != nil {
		return err
	}
	redirectResolutions, err := openRedirectResolutions(ctx, js)
	if err != nil {
		return err
	}
	visitorSource, err := buildVisitorSource(ctx, js, cfg, metrics, redirectResolutions)
	if err != nil {
		return err
	}
	traverser := ordertraversal.New(
		traversalConfig(cfg),
		visitorSource,
		metrics,
		disposal.NewDisposer(metrics, disposedPages),
		wallclock.Clock{},
	)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	crawlOutcomeReceiver := crawloutcomereceiversgrpc.NewCrawlOutcomeReceiver(
		redirectResolutions, disposedPages, cfg.ListenAddr,
	)

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.OrdersSubject),
		slog.String("listen", cfg.ListenAddr),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
	)

	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			return ordersettlement.New(
				traverser,
				metrics,
				wallclock.Clock{},
				ordersAckWait/2,
			).Settle(runCtx, orderReceiver.Deliveries())
		},
		crawlOutcomeReceiver.Serve,
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func ensureNATSState(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if err := ensureOrdersStream(ctx, js, cfg.OrdersSubject); err != nil {
		return err
	}
	if err := ensureReachedPagesStream(ctx, js); err != nil {
		return err
	}
	if err := ensureRedirectResolutionBucket(ctx, js); err != nil {
		return err
	}
	if err := ensureDisposedPagesBucket(ctx, js); err != nil {
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

func ensureReachedPagesStream(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.ReachedPagesStreamName,
		Subjects:  []string{yacycrawlcontract.ReachedPageSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   DefaultMaxMsgs,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		return fmt.Errorf("ensure reached pages stream: %w", err)
	}
	return nil
}

func ensureRedirectResolutionBucket(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   yacycrawlcontract.RedirectResolutionBucketName,
		MaxBytes: DefaultRedirectResolutionMaxBytes,
	}); err != nil {
		return fmt.Errorf("ensure redirect resolution bucket: %w", err)
	}
	return nil
}

func ensureDisposedPagesBucket(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   yacycrawlcontract.DisposedPagesBucketName,
		MaxBytes: DefaultDisposedPagesMaxBytes,
		TTL:      DefaultDisposedPagesRetention,
	}); err != nil {
		return fmt.Errorf("ensure disposed pages bucket: %w", err)
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

func openRedirectResolutions(
	ctx context.Context,
	js jetstream.JetStream,
) (*redirectresolversjetstream.RedirectResolutions, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return nil, fmt.Errorf("open redirect resolution bucket: %w", err)
	}
	return redirectresolversjetstream.NewRedirectResolutions(bucket), nil
}

func openDisposedPages(
	ctx context.Context,
	js jetstream.JetStream,
) (*disposedpagesjetstream.DisposedPages, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.DisposedPagesBucketName)
	if err != nil {
		return nil, fmt.Errorf("open disposed pages bucket: %w", err)
	}
	return disposedpagesjetstream.NewDisposedPages(bucket), nil
}

func buildVisitorSource(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	metrics *progressobserversprometheus.CrawlMetrics,
	redirectResolutions redirectrecording.RedirectResolutions,
) (pagevisit.VisitorSource, error) {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	extractor, err := buildExtractor(cfg)
	if err != nil {
		return nil, err
	}
	recrawl, err := recrawlRule(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	return pagevisit.New(
		redirectrecording.New(
			redirectResolutions,
			fetchtiming.New(metrics, wallclock.Clock{}, fetch),
		),
		recrawl,
		extractor,
		metrics,
		reachedpagepublication.NewPublisher(metrics, reachedpagesjetstream.New(js)),
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
			Durable:       cfg.OrdersDurable,
			FilterSubject: cfg.OrdersSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       ordersAckWait,
			MaxAckPending: 1,
		})
	if err != nil {
		return nil, fmt.Errorf("create orders consumer: %w", err)
	}
	return consumer, nil
}

func buildExtractor(cfg ServiceConfig) (pagevisit.PageExtractor, error) {
	admitted, err := admittedMediaTypesFor(cfg)
	if err != nil {
		return nil, err
	}
	extraction, err := contentextraction.New(admitted.extractors)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvContentTypes, err)
	}
	return extraction, nil
}

func traversalConfig(cfg ServiceConfig) ordertraversal.Config {
	return ordertraversal.Config{
		RunPageBudget:    cfg.RunPageBudget,
		VisitConcurrency: cfg.FetchConcurrency,
		MaxAdmittedURLs:  cfg.FrontierCap,
		Frontier: frontier.Config{
			MaxDeferralsPerURL: maxDeferPerURL,
			MaxAttemptsPerURL:  fetchRetryLimit,
			RetryDelay: retrydelay.Bounds{
				Floor:   fetchRetryFloor,
				Ceiling: fetchRetryCeiling,
			},
		},
	}
}

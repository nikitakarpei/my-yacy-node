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
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	acceptedordersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/observedfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	crawledpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/applog"
	crawledpageobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
	crawlorderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/applog"
	crawlorderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/prometheus"
	linkresolutionobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/applog"
	linkresolutionobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/prometheus"
	mediatypeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/applog"
	mediatypeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/prometheus"
	pagefetchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/applog"
	pagefetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/prometheus"
	pagevisitlimitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/jetstream"
	pagevisitrecordobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/applog"
	pagevisitrecordobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/prometheus"
	pagevisitrecordsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/norecords"
	pendingpagevisitobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/applog"
	pendingpagevisitobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/prometheus"
	pendingpagevisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisits/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	refusalenforcementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/applog"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
	takenpagevisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/jetstream"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	metricNamespace    = "yacycrawler"
	msgServiceStarted  = "crawler started"
	msgServiceStopped  = "crawler stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	crawlOrderObservers := orderintake.CrawlOrderObservers{
		crawlorderobserversapplog.CrawlOrderLog{}, crawlorderobserversprometheus.New(registry),
	}
	pendingPageVisitObservers := pagevisitintake.PendingPageVisitObservers{
		pendingpagevisitobserversapplog.PendingPageVisitLog{},
		pendingpagevisitobserversprometheus.New(registry),
	}
	pageFetchObservers := observedfetch.PageFetchObservers{
		pagefetchobserversapplog.PageFetchLog{},
		pagefetchobserversprometheus.New(registry, cfg.FetchDeadline),
	}
	htmlPageReading := pagehtmlreading.NewHTMLPageReading(
		pagehtml.NewHTMLParser(pagehtml.MediaTypeObservers{
			mediatypeobserversapplog.MediaTypeLog{},
			mediatypeobserversprometheus.New(registry),
		}),
		linkdiscovery.NewLinkDiscovery(linkdiscovery.LinkResolutionObservers{
			linkresolutionobserversapplog.LinkResolutionLog{},
			linkresolutionobserversprometheus.New(registry),
		}),
		pagehtmlreading.RefusalEnforcementObservers{
			refusalenforcementobserversapplog.RefusalEnforcementLog{},
			refusalenforcementobserversprometheus.New(registry),
		},
	)
	crawledPageObservers := crawledpagesjetstream.CrawledPagePublicationObservers{
		crawledpageobserversapplog.CrawledPagePublicationLog{},
		crawledpageobserversprometheus.New(registry),
	}
	pageVisitRecordObservers := pagevisitrecordsjetstream.PageVisitRecordObservers{
		pagevisitrecordobserversapplog.PageVisitRecordLog{},
		pagevisitrecordobserversprometheus.New(registry),
	}
	js, connection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := ensureNATSState(ctx, js, cfg); err != nil {
		return err
	}
	orders, err := buildOrderConsumer(ctx, js, cfg, crawlOrderObservers)
	if err != nil {
		return err
	}
	crawledPageAdmission, err := openCrawledPageAdmission(ctx, js, cfg, registry)
	if err != nil {
		return err
	}
	visits, err := buildPageVisitConsumer(
		ctx, js, cfg, crawledPageAdmission, pendingPageVisitObservers, pageFetchObservers,
		htmlPageReading, crawledPageObservers, pageVisitRecordObservers,
	)
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
	if err := ensureCrawledPagesStream(ctx, js, cfg); err != nil {
		return err
	}
	if err := takenpagevisitsjetstream.Ensure(ctx, js, cfg.TakenPageVisitBucketSpec()); err != nil {
		return err
	}
	if err := pagevisitlimitsjetstream.Ensure(
		ctx, js, cfg.PageVisitLimitBucketSpec(),
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
		Name:       pendingpagevisit.StreamName,
		Subjects:   []string{pendingpagevisit.Subject},
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: cfg.PendingPageVisitDuplicateWindow(),
	}); err != nil {
		return fmt.Errorf("ensure frontier stream: %w", err)
	}
	return nil
}

func ensureCrawledPagesStream(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.CrawledPagesStreamName,
		Subjects:  []string{yacycrawlcontract.EveryCrawledPageSubject},
		Retention: jetstream.LimitsPolicy,
		Discard:   jetstream.DiscardOld,
		MaxAge:    cfg.CrawledPageRetention(),
	}); err != nil {
		return fmt.Errorf("ensure crawled pages stream: %w", err)
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
	return pagevisitrecordsjetstream.Ensure(ctx, js, cfg.PageVisitBucketSpec())
}

func buildOrderConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	observer orderintake.CrawlOrderObserver,
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
		pendingpagevisitsjetstream.New(js),
		observer,
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

func openCrawledPageAdmission(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) (*consumerbacklog.Admission, error) {
	crawledPages, err := js.Stream(ctx, yacycrawlcontract.CrawledPagesStreamName)
	if err != nil {
		return nil, fmt.Errorf("open crawled pages stream: %w", err)
	}
	return consumerbacklog.New(
		crawledPages, cfg.CrawledPageBacklogLimit(), wallclock.Clock{}, registry, metricNamespace,
	), nil
}

func buildPageVisitConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	crawledPageAdmission *consumerbacklog.Admission,
	pendingPageVisitObserver pagevisitintake.PendingPageVisitObserver,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver pagevisitrecordsjetstream.PageVisitRecordObserver,
) (*pagevisitintake.PageVisitConsumer, error) {
	consumer, err := visitsConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	takenPageVisits, err := takenPageVisitsOf(ctx, js)
	if err != nil {
		return nil, err
	}
	allowances, err := pageVisitAllowances(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	orders, err := acceptedOrders(ctx, js)
	if err != nil {
		return nil, err
	}
	pageVisitor, err := buildPageVisitor(
		ctx,
		js,
		cfg,
		pageFetchObserver,
		htmlPageReading,
		crawledPageObserver,
		pageVisitRecordObserver,
	)
	if err != nil {
		return nil, err
	}
	return pagevisitintake.NewPageVisitConsumer(
		consumer,
		takenPageVisits,
		allowances,
		orders,
		pageVisitor,
		pendingPageVisitObserver,
		crawledPageAdmission,
		cfg.FetchConcurrency,
	), nil
}

func visitsConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	consumer, err := js.CreateOrUpdateConsumer(ctx, pendingpagevisit.StreamName,
		jetstream.ConsumerConfig{
			Durable:       cfg.PendingPageVisitDurable,
			FilterSubject: pendingpagevisit.Subject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       cfg.PendingPageVisitAckWait(),
		})
	if err != nil {
		return nil, fmt.Errorf("create pending page visit consumer: %w", err)
	}
	return consumer, nil
}

func takenPageVisitsOf(
	ctx context.Context,
	js jetstream.JetStream,
) (*takenpagevisitsjetstream.TakenPageVisits, error) {
	bucket, err := js.KeyValue(ctx, takenpagevisitsjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open taken page visit bucket: %w", err)
	}
	return takenpagevisitsjetstream.New(bucket), nil
}

func pageVisitAllowances(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) (*pagevisitallowance.Allowances, error) {
	bucket, err := js.KeyValue(ctx, pagevisitlimitsjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page visit limit bucket: %w", err)
	}
	return pagevisitallowance.New(
		pagevisitlimitsjetstream.New(bucket, cfg.MaxPerURL()),
		cfg.FetchRetryBounds(),
	), nil
}

func buildPageVisitor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver pagevisitrecordsjetstream.PageVisitRecordObserver,
) (pagevisit.PageVisitor, error) {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	pageVisitRecords, err := pageVisitRecordsOf(ctx, js, cfg, pageVisitRecordObserver)
	if err != nil {
		return nil, err
	}
	return pagevisit.New(
		observedfetch.New(fetch, wallclock.Clock{}, pageFetchObserver),
		dueaftergrace.New(wallclock.Clock{}, cfg.RecrawlGrace),
		pageVisitRecords,
		htmlPageReading,
		crawledpagesjetstream.New(js, crawledPageObserver),
		pendingpagevisitsjetstream.New(js),
	), nil
}

func pageVisitRecordsOf(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	observer pagevisitrecordsjetstream.PageVisitRecordObserver,
) (pagevisit.PageVisitRecords, error) {
	if !cfg.SuppressesRecrawl() {
		return norecords.PageVisitRecords{}, nil
	}
	bucket, err := js.KeyValue(ctx, pagevisitrecordsjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page visit bucket: %w", err)
	}
	return pagevisitrecordsjetstream.New(bucket, wallclock.Clock{}, observer), nil
}

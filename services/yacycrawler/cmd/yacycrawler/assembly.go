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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
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
	pagevisitfailureobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitfailureobservers/applog"
	pagevisitfailureobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitfailureobservers/prometheus"
	pagevisitlimitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/jetstream"
	pagevisitrecordobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/applog"
	pagevisitrecordobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/prometheus"
	pendingpagevisitobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/applog"
	pendingpagevisitobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/prometheus"
	pendingpagevisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisits/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	refusalenforcementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/applog"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
	takenpagevisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/jetstream"
	visitedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitedpages/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitedpages/norecords"
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
	crawlOrderObservers := orderintake.CrawlOrderObservers{
		crawlorderobserversapplog.CrawlOrderLog{}, crawlorderobserversprometheus.New(registry),
	}
	pendingPageVisitObservers := pagevisitintake.PendingPageVisitObservers{
		pendingpagevisitobserversapplog.PendingPageVisitLog{},
		pendingpagevisitobserversprometheus.New(registry),
	}
	pageFetchObservers := pagevisit.PageFetchObservers{
		pagefetchobserversapplog.PageFetchLog{}, pagefetchobserversprometheus.New(registry),
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
	)
	refusalEnforcementObservers := pagevisit.RefusalEnforcementObservers{
		refusalenforcementobserversapplog.RefusalEnforcementLog{},
		refusalenforcementobserversprometheus.New(registry),
	}
	pageVisitFailureObservers := pagevisit.PageVisitFailureObservers{
		pagevisitfailureobserversapplog.PageVisitFailureLog{},
		pagevisitfailureobserversprometheus.New(registry),
	}
	crawledPageObservers := crawledpagesjetstream.CrawledPagePublicationObservers{
		crawledpageobserversapplog.CrawledPagePublicationLog{},
		crawledpageobserversprometheus.New(registry),
	}
	pageVisitRecordObservers := visitedpagesjetstream.PageVisitRecordObservers{
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
	visits, err := buildPageVisitConsumer(
		ctx, js, cfg, pendingPageVisitObservers, pageFetchObservers, htmlPageReading,
		refusalEnforcementObservers, crawledPageObservers, pageVisitRecordObservers,
		pageVisitFailureObservers,
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
	return visitedpagesjetstream.Ensure(ctx, js, cfg.PageVisitBucketSpec())
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

func buildPageVisitConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	pendingPageVisitObserver pagevisitintake.PendingPageVisitObserver,
	pageFetchObserver pagevisit.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	refusalEnforcementObserver pagevisit.RefusalEnforcementObserver,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver visitedpagesjetstream.PageVisitRecordObserver,
	pageVisitFailureObserver pagevisit.PageVisitFailureObserver,
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
		refusalEnforcementObserver,
		crawledPageObserver,
		pageVisitRecordObserver,
		pageVisitFailureObserver,
	)
	if err != nil {
		return nil, err
	}
	return pagevisitintake.NewPageVisitConsumer(
		consumer,
		takenPageVisits,
		allowances,
		orders,
		pendingpagevisitsjetstream.New(js),
		pageVisitor,
		pendingPageVisitObserver,
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
	pageFetchObserver pagevisit.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	refusalEnforcementObserver pagevisit.RefusalEnforcementObserver,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver visitedpagesjetstream.PageVisitRecordObserver,
	pageVisitFailureObserver pagevisit.PageVisitFailureObserver,
) (pagevisit.PageVisitor, error) {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	visitedPages, err := visitedPagesOf(ctx, js, cfg, pageVisitRecordObserver)
	if err != nil {
		return nil, err
	}
	pageFetcher := pagevisit.NewObservedPageFetcher(fetch, wallclock.Clock{}, pageFetchObserver)
	crawledPages := crawledpagesjetstream.New(js, crawledPageObserver)
	return pagevisit.New(
		pageFetcher,
		dueaftergrace.New(wallclock.Clock{}, cfg.RecrawlGrace),
		visitedPages,
		htmlPageReading,
		refusalEnforcementObserver,
		crawledPages,
		pageVisitFailureObserver,
	), nil
}

func visitedPagesOf(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	observer visitedpagesjetstream.PageVisitRecordObserver,
) (pagevisit.VisitedPages, error) {
	if !cfg.SuppressesRecrawl() {
		return norecords.VisitedPages{}, nil
	}
	bucket, err := js.KeyValue(ctx, visitedpagesjetstream.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page visit bucket: %w", err)
	}
	return visitedpagesjetstream.New(bucket, wallclock.Clock{}, observer), nil
}

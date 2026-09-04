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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
	crawledpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/applog"
	crawledpageobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
	crawlorderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/applog"
	crawlorderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/prometheus"
	hostpageallowancesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/hostpageallowances/jetstream"
	linkresolutionobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/applog"
	linkresolutionobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/prometheus"
	mediatypeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/applog"
	mediatypeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/prometheus"
	pagefetchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/applog"
	pagefetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/prometheus"
	pendingvisitobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisitobservers/applog"
	pendingvisitobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisitobservers/prometheus"
	pendingvisitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisits/jetstream"
	recrawlrecordobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrecordobservers/applog"
	recrawlrecordobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrecordobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/alwaysdue"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	refusalenforcementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/applog"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
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
	crawlOrderObservers := orderintake.CrawlOrderObservers{
		crawlorderobserversapplog.CrawlOrderLog{}, crawlorderobserversprometheus.New(registry),
	}
	pendingVisitObservers := visitintake.PendingVisitObservers{
		pendingvisitobserversapplog.PendingVisitLog{},
		pendingvisitobserversprometheus.New(registry),
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
	crawledPageObservers := crawledpagesjetstream.CrawledPagePublicationObservers{
		crawledpageobserversapplog.CrawledPagePublicationLog{},
		crawledpageobserversprometheus.New(registry),
	}
	recrawlRecordObservers := pagevisit.RecrawlRecordObservers{
		recrawlrecordobserversapplog.RecrawlRecordLog{},
		recrawlrecordobserversprometheus.New(registry),
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
	visits, err := buildVisitConsumer(
		ctx, js, cfg, pendingVisitObservers, pageFetchObservers, htmlPageReading,
		refusalEnforcementObservers, crawledPageObservers, recrawlRecordObservers,
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
	return dueaftergrace.Ensure(ctx, js, cfg.PageVisitBucketSpec())
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
		pendingvisitsjetstream.New(js),
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

func buildVisitConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	pendingVisitObserver visitintake.PendingVisitObserver,
	pageFetchObserver pagevisit.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	refusalEnforcementObserver pagevisit.RefusalEnforcementObserver,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	recrawlRecordObserver pagevisit.RecrawlRecordObserver,
) (*visitintake.VisitConsumer, error) {
	consumer, err := visitsConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	claims, err := visitClaims(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	ledger, err := visitLedger(ctx, js, cfg, claims)
	if err != nil {
		return nil, err
	}
	orders, err := acceptedOrders(ctx, js)
	if err != nil {
		return nil, err
	}
	visitor, err := buildVisitor(
		ctx,
		js,
		cfg,
		pageFetchObserver,
		htmlPageReading,
		refusalEnforcementObserver,
		crawledPageObserver,
		recrawlRecordObserver,
	)
	if err != nil {
		return nil, err
	}
	return visitintake.NewVisitConsumer(
		consumer,
		claims,
		ledger,
		orders,
		pendingvisitsjetstream.New(js),
		visitor,
		pendingVisitObserver,
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

func buildVisitor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	pageFetchObserver pagevisit.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	refusalEnforcementObserver pagevisit.RefusalEnforcementObserver,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	recrawlRecordObserver pagevisit.RecrawlRecordObserver,
) (pagevisit.Visitor, error) {
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
	pageFetcher := pagevisit.NewObservedPageFetcher(fetch, wallclock.Clock{}, pageFetchObserver)
	bestEffortRecrawl := pagevisit.NewBestEffortRecrawlRule(recrawl, recrawlRecordObserver)
	crawledPages := crawledpagesjetstream.New(js, crawledPageObserver)
	return pagevisit.New(
		pageFetcher,
		bestEffortRecrawl,
		htmlPageReading,
		refusalEnforcementObserver,
		crawledPages,
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

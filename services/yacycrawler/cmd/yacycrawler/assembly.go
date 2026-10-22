package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/pebblemetrics"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/observedfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	crawledpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/applog"
	crawledpageobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
	crawlorderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/applog"
	crawlorderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/prometheus"
	crawlsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawls/vault"
	frontiermetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontiermetrics/prometheus"
	frontierobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontierobservers/applog"
	frontiersvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontiers/vault"
	linkresolutionobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/applog"
	linkresolutionobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/prometheus"
	mediatypeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/applog"
	mediatypeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/prometheus"
	pagefetchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/applog"
	pagefetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/prometheus"
	pagevisitlimitsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/vault"
	pagevisitrecordobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/applog"
	pagevisitrecordobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/norecords"
	pagevisitrecordsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/vault"
	pendingpagevisitobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/applog"
	pendingpagevisitobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	refusalenforcementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/applog"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
	takenpagevisitsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/vault"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	metricNamespace    = "yacycrawler"
	msgServiceStarted  = "crawler started"
	msgServiceStopped  = "crawler stopped"
)

type Listeners struct {
	Ops net.Listener
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
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
	pageVisitRecordObservers := pagevisitrecordsvault.PageVisitRecordObservers{
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
	storage, err := openCrawlState(cfg, registry)
	if err != nil {
		return err
	}
	defer func() { _ = storage.Close() }()
	crawlState, err := openCrawlStateStores(
		ctx, storage, cfg, pageVisitRecordObservers,
	)
	if err != nil {
		return err
	}
	frontiermetricsprometheus.New(registry, crawlState.frontier)
	orders, err := buildOrderConsumer(ctx, js, crawlState, cfg, crawlOrderObservers)
	if err != nil {
		return err
	}
	crawledPageAdmission, err := openCrawledPageAdmission(ctx, js, cfg, registry)
	if err != nil {
		return err
	}
	visits := buildPageVisitConsumer(
		js,
		crawlState,
		cfg,
		crawledPageAdmission,
		pendingPageVisitObservers,
		pageFetchObservers,
		htmlPageReading,
		crawledPageObservers,
	)

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{
			{Name: "ops", Listener: listeners.Ops, Server: opsServerFor(registry)},
		},
		orders.Run,
		visits.Run,
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func opsServerFor(registry *prometheus.Registry) *http.Server {
	return &http.Server{
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
}

func ensureNATSState(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if err := ensureOrdersStream(ctx, js, cfg); err != nil {
		return err
	}
	return ensureCrawledPagesStream(ctx, js, cfg)
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

func openCrawlState(
	cfg ServiceConfig,
	registry prometheus.Registerer,
) (*vault.Vault, error) {
	engine, err := pebblevault.OpenEngine(
		cfg.CrawlStateDir,
		0,
		cfg.CrawlStateLimits,
		pebblemetrics.NewWriteStallMetrics(registry, metricNamespace),
		pebblemetrics.NewWriteConflictMetrics(registry, metricNamespace),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvCrawlStateDir, err)
	}
	pebblemetrics.NewConditionMetrics(registry, metricNamespace, engine)
	storage, err := vault.New(
		engine,
		vaultmetrics.NewTransactionMetrics(registry, metricNamespace),
		vaultmetrics.NewExpiryMetrics(registry, metricNamespace),
		wallclock.Clock{},
	)
	if err != nil {
		return nil, errors.Join(err, engine.Close())
	}
	vaultmetrics.NewCapacityMetrics(registry, metricNamespace, storage)
	vaultmetrics.NewBucketRecordMetrics(registry, metricNamespace, storage)
	return storage, nil
}

type crawlStateStores struct {
	frontier         *frontiersvault.Frontier
	crawls           *crawlsvault.Crawls
	takenPageVisits  *takenpagevisitsvault.TakenPageVisits
	pageVisitLimits  *pagevisitlimitsvault.PageVisitLimits
	pageVisitRecords pagevisit.PageVisitRecords
}

func openCrawlStateStores(
	ctx context.Context,
	storage *vault.Vault,
	cfg ServiceConfig,
	pageVisitRecordObserver pagevisitrecordsvault.PageVisitRecordObserver,
) (crawlStateStores, error) {
	frontier, err := frontiersvault.New(
		ctx,
		storage,
		cfg.PendingPageVisitLease(),
		wallclock.Clock{},
		frontierobserversapplog.FrontierLog{},
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	crawls, err := crawlsvault.New(ctx, storage, cfg.CrawlRetention())
	if err != nil {
		return crawlStateStores{}, err
	}
	takenPageVisits, err := takenpagevisitsvault.New(
		ctx, storage, cfg.TakenPageVisitRetention(),
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	pageVisitLimits, err := pagevisitlimitsvault.New(
		ctx, storage, cfg.PageVisitLimitRetention(), cfg.MaxPerURL(),
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	pageVisitRecords, err := pageVisitRecordsOf(
		ctx, storage, cfg, pageVisitRecordObserver,
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	return crawlStateStores{
		frontier:         frontier,
		crawls:           crawls,
		takenPageVisits:  takenPageVisits,
		pageVisitLimits:  pageVisitLimits,
		pageVisitRecords: pageVisitRecords,
	}, nil
}

func pageVisitRecordsOf(
	ctx context.Context,
	storage *vault.Vault,
	cfg ServiceConfig,
	observer pagevisitrecordsvault.PageVisitRecordObserver,
) (pagevisit.PageVisitRecords, error) {
	if !cfg.SuppressesRecrawl() {
		return norecords.PageVisitRecords{}, nil
	}
	return pagevisitrecordsvault.New(
		ctx, storage, wallclock.Clock{}, cfg.PageVisitRetention(), observer,
	)
}

func buildOrderConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	crawlState crawlStateStores,
	cfg ServiceConfig,
	observer orderintake.CrawlOrderObserver,
) (*orderintake.OrderConsumer, error) {
	consumer, err := ordersConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	return orderintake.NewOrderConsumer(
		consumer,
		crawlState.crawls,
		crawlState.frontier,
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
	js jetstream.JetStream,
	crawlState crawlStateStores,
	cfg ServiceConfig,
	crawledPageAdmission *consumerbacklog.Admission,
	pendingPageVisitObserver pagevisitintake.PendingPageVisitObserver,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
) *pagevisitintake.PageVisitConsumer {
	pageVisitor := buildPageVisitor(
		js,
		crawlState,
		cfg,
		pageFetchObserver,
		htmlPageReading,
		crawledPageObserver,
	)
	return pagevisitintake.NewPageVisitConsumer(
		crawlState.frontier,
		crawlState.takenPageVisits,
		pagevisitallowance.New(crawlState.pageVisitLimits, cfg.FetchRetryBounds()),
		crawlState.crawls,
		pageVisitor,
		pendingPageVisitObserver,
		crawledPageAdmission,
		cfg.FetchConcurrency,
	)
}

func buildPageVisitor(
	js jetstream.JetStream,
	crawlState crawlStateStores,
	cfg ServiceConfig,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
) pagevisit.PageVisitor {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	return pagevisit.New(
		observedfetch.New(fetch, wallclock.Clock{}, pageFetchObserver),
		dueaftergrace.New(wallclock.Clock{}, cfg.RecrawlGrace),
		crawlState.pageVisitRecords,
		htmlPageReading,
		crawledpagesjetstream.New(js, crawledPageObserver),
		crawlState.frontier,
	)
}

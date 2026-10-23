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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlfinishing"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawls"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/observedfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitlimits"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitrecords"
	crawledpageobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/jetstream/applog"
	crawledpageobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/jetstream/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
	crawlerapiobservershttpapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlerapiobservers/http/applog"
	crawlerapishttp "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlerapis/http"
	crawlobserverscrawlfinishingapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/crawlfinishing/applog"
	crawlobserverscrawlprogressapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/crawlprogress/applog"
	crawlobserversfrontierapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/frontier/applog"
	crawlobserversfrontierprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/frontier/prometheus"
	crawlobserverslinkdiscoveryapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/linkdiscovery/applog"
	crawlobserverslinkdiscoveryprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/linkdiscovery/prometheus"
	crawlobserversobservedfetchapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/observedfetch/applog"
	crawlobserversobservedfetchprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/observedfetch/prometheus"
	crawlobserversorderintakeapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/orderintake/applog"
	crawlobserversorderintakeprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/orderintake/prometheus"
	crawlobserverspagehtmlapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagehtml/applog"
	crawlobserverspagehtmlprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagehtml/prometheus"
	crawlobserverspagehtmlreadingapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagehtmlreading/applog"
	crawlobserverspagehtmlreadingprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagehtmlreading/prometheus"
	crawlobserverspagevisitapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagevisit/applog"
	crawlobserverspagevisitprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagevisit/prometheus"
	crawlobserverspagevisitintakeapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagevisitintake/applog"
	crawlobserverspagevisitintakeprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagevisitintake/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
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
	API net.Listener
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	crawlOrderObservers := orderintake.CrawlOrderObservers{
		crawlobserversorderintakeapplog.CrawlOrderLog{},
		crawlobserversorderintakeprometheus.New(registry),
	}
	pageFetchObservers := observedfetch.PageFetchObservers{
		crawlobserversobservedfetchapplog.PageFetchLog{},
		crawlobserversobservedfetchprometheus.New(registry, cfg.FetchDeadline),
	}
	htmlPageReading := pagehtmlreading.NewHTMLPageReading(
		pagehtml.NewHTMLParser(pagehtml.MediaTypeObservers{
			crawlobserverspagehtmlapplog.MediaTypeLog{},
			crawlobserverspagehtmlprometheus.New(registry),
		}),
		linkdiscovery.NewLinkDiscovery(linkdiscovery.LinkResolutionObservers{
			crawlobserverslinkdiscoveryapplog.LinkResolutionLog{},
			crawlobserverslinkdiscoveryprometheus.New(registry),
		}),
		pagehtmlreading.RefusalEnforcementObservers{
			crawlobserverspagehtmlreadingapplog.RefusalEnforcementLog{},
			crawlobserverspagehtmlreadingprometheus.New(registry),
		},
	)
	crawledPageObservers := crawledpagesjetstream.CrawledPagePublicationObservers{
		crawledpageobserversjetstreamapplog.CrawledPagePublicationLog{},
		crawledpageobserversjetstreamprometheus.New(registry),
	}
	pageVisitRecordObservers := pagevisit.PageVisitRecordObservers{
		crawlobserverspagevisitapplog.PageVisitRecordLog{},
		crawlobserverspagevisitprometheus.New(registry),
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
	crawlState, err := openCrawlStateStores(ctx, storage, cfg)
	if err != nil {
		return err
	}
	crawlobserversfrontierprometheus.New(registry, crawlState.frontier)
	progressCounter := crawlprogress.NewCounter(
		storage,
		crawlState.progressCounts,
		crawlobserverscrawlprogressapplog.CrawlProgressLog{},
	)
	pendingPageVisitObservers := pagevisitintake.PendingPageVisitObservers{
		crawlobserverspagevisitintakeapplog.PendingPageVisitLog{},
		crawlobserverspagevisitintakeprometheus.New(registry),
		progressCounter,
	}
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
		pageVisitRecordObservers,
		htmlPageReading,
		crawledPageObservers,
		progressCounter,
	)

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
	)
	err = servergroup.Run(
		ctx,
		opsShutdownLimit,
		serversFor(listeners, registry, crawlerapishttp.New(
			storage,
			crawlState.crawls,
			crawlState.frontier,
			crawlState.progressCounts,
			crawlState.frontier,
			crawlerapiobservershttpapplog.CrawlerAPILog{},
		)),
		orders.Run,
		visits.Run,
		crawlState.frontier.DrainQueues,
		crawlfinishing.NewFinisher(
			storage,
			crawlState.crawls,
			crawlState.frontier,
			wallclock.Clock{},
			crawlobserverscrawlfinishingapplog.CrawlFinishingLog{},
		).Run,
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func serversFor(
	listeners Listeners,
	registry *prometheus.Registry,
	crawlerAPI http.Handler,
) []servergroup.NamedServer {
	servers := make([]servergroup.NamedServer, 0, 2)
	servers = append(servers, servergroup.NamedServer{
		Name: "ops", Listener: listeners.Ops, Server: opsServerFor(registry),
	})
	if listeners.API == nil {
		return servers
	}
	return append(servers, servergroup.NamedServer{
		Name:     "api",
		Listener: listeners.API,
		Server:   &http.Server{Handler: crawlerAPI, ReadHeaderTimeout: opsReadHeaderLimit},
	})
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
	storage          *vault.Vault
	crawls           *crawls.Crawls
	frontier         *frontier.Frontier
	progressCounts   *crawlprogress.Counts
	pageVisitLimits  *pagevisitlimits.PageVisitLimits
	pageVisitRecords pagevisit.PageVisitRecords
}

func openCrawlStateStores(
	ctx context.Context,
	storage *vault.Vault,
	cfg ServiceConfig,
) (crawlStateStores, error) {
	wiring := &crawlStateWiring{}
	progressCounts, err := crawlprogress.Open(storage)
	if err != nil {
		return crawlStateStores{}, err
	}
	pageVisitFrontier, err := frontier.Open(
		ctx,
		storage,
		wiring,
		cfg.PendingPageVisitLease(),
		wallclock.Clock{},
		crawlobserversfrontierapplog.FrontierLog{},
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	wiring.frontier, wiring.progress = pageVisitFrontier, progressCounts
	keptCrawls, err := crawls.Open(
		ctx, storage, cfg.CrawlRetention(), wiring, wiring, wallclock.Clock{},
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	pageVisitLimits, err := pagevisitlimits.Open(
		ctx, storage, cfg.PageVisitLimitRetention(), cfg.MaxPerURL(),
	)
	if err != nil {
		return crawlStateStores{}, err
	}
	pageVisitRecords, err := pageVisitRecordsOf(ctx, storage, cfg)
	if err != nil {
		return crawlStateStores{}, err
	}
	return crawlStateStores{
		storage:          storage,
		crawls:           keptCrawls,
		frontier:         pageVisitFrontier,
		progressCounts:   progressCounts,
		pageVisitLimits:  pageVisitLimits,
		pageVisitRecords: pageVisitRecords,
	}, nil
}

func pageVisitRecordsOf(
	ctx context.Context,
	storage *vault.Vault,
	cfg ServiceConfig,
) (pagevisit.PageVisitRecords, error) {
	if !cfg.SuppressesRecrawl() {
		return pagevisitrecords.None{}, nil
	}
	return pagevisitrecords.Open(ctx, storage, wallclock.Clock{}, cfg.PageVisitRetention())
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
		crawlState.storage,
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
	pageVisitRecordObserver pagevisit.PageVisitRecordObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	linkObserver pagevisit.LinkAdmissionObserver,
) *pagevisitintake.PageVisitConsumer {
	pageVisitor := buildPageVisitor(
		js,
		crawlState,
		cfg,
		pageFetchObserver,
		pageVisitRecordObserver,
		htmlPageReading,
		crawledPageObserver,
		linkObserver,
	)
	return pagevisitintake.NewPageVisitConsumer(
		crawlState.storage,
		crawlState.frontier,
		pagevisitallowance.New(
			crawlState.storage,
			crawlState.pageVisitLimits,
			cfg.FetchRetryBounds(),
			cfg.MaxDeferral(),
		),
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
	pageVisitRecordObserver pagevisit.PageVisitRecordObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	linkObserver pagevisit.LinkAdmissionObserver,
) pagevisit.PageVisitor {
	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	return pagevisit.New(
		crawlState.storage,
		observedfetch.New(fetch, wallclock.Clock{}, pageFetchObserver),
		dueaftergrace.New(wallclock.Clock{}, cfg.RecrawlGrace),
		crawlState.pageVisitRecords,
		pageVisitRecordObserver,
		htmlPageReading,
		crawledpagesjetstream.New(js, crawledPageObserver),
		crawlState.frontier,
		linkObserver,
	)
}

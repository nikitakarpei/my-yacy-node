package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	acceptedordersredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/observedfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
	crawledpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/applog"
	crawledpageobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
	crawlorderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/applog"
	crawlorderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/prometheus"
	frontiermetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontiermetrics/prometheus"
	frontierobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontierobservers/applog"
	linkresolutionobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/applog"
	linkresolutionobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/prometheus"
	mediatypeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/applog"
	mediatypeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/prometheus"
	pagefetchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/applog"
	pagefetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/prometheus"
	pagevisitlimitsredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/redis"
	pagevisitrecordobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/applog"
	pagevisitrecordobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecordobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/norecords"
	pagevisitrecordsredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/redis"
	pendingpagevisitobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/applog"
	pendingpagevisitobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/prometheus"
	pendingpagevisitsredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisits/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	refusalenforcementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/applog"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
	robotsrulesorigin "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrules/origin"
	robotsrulesredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrules/redis"
	robotsrulesobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrulesobservers/applog"
	robotsrulesobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrulesobservers/prometheus"
	robotstxtfetchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotstxtfetchobservers/applog"
	robotstxtfetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotstxtfetchobservers/prometheus"
	takenpagevisitsredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/redis"
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
	pageVisitRecordObservers := pagevisitrecordsredis.PageVisitRecordObservers{
		pagevisitrecordobserversapplog.PageVisitRecordLog{},
		pagevisitrecordobserversprometheus.New(registry),
	}
	robotsTxtFetchObservers := robotsrulesorigin.RobotsTxtFetchObservers{
		robotstxtfetchobserversapplog.RobotsTxtFetchLog{},
		robotstxtfetchobserversprometheus.New(registry),
	}
	robotsRulesObservers := robotsrulesredis.RobotsRulesObservers{
		robotsrulesobserversapplog.RobotsRulesLog{},
		robotsrulesobserversprometheus.New(registry),
	}
	js, connection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := ensureNATSState(ctx, js, cfg); err != nil {
		return err
	}
	redisClient, err := openRedisClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = redisClient.Close() }()
	frontier := pendingpagevisitsredis.New(
		redisClient,
		cfg.PendingPageVisitLease(),
		wallclock.Clock{},
		frontierobserversapplog.FrontierLog{},
	)
	frontiermetricsprometheus.New(registry, frontier)
	orders, err := buildOrderConsumer(ctx, js, redisClient, frontier, cfg, crawlOrderObservers)
	if err != nil {
		return err
	}
	crawledPageAdmission, err := openCrawledPageAdmission(ctx, js, cfg, registry)
	if err != nil {
		return err
	}
	visits := buildPageVisitConsumer(
		js,
		redisClient,
		frontier,
		cfg,
		crawledPageAdmission,
		buildRobotsExclusion(redisClient, cfg, robotsTxtFetchObservers, robotsRulesObservers),
		pendingPageVisitObservers,
		pageFetchObservers,
		htmlPageReading,
		crawledPageObservers,
		pageVisitRecordObservers,
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

func openRedisClient(ctx context.Context, cfg ServiceConfig) (*redis.Client, error) {
	redisClient := redis.NewClient(cfg.CrawlStateRedis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("%s: %w", EnvCrawlStateRedisURL, err)
	}
	return redisClient, nil
}

func buildOrderConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	redisClient redis.UniversalClient,
	frontier *pendingpagevisitsredis.PendingPageVisits,
	cfg ServiceConfig,
	observer orderintake.CrawlOrderObserver,
) (*orderintake.OrderConsumer, error) {
	consumer, err := ordersConsumer(ctx, js, cfg)
	if err != nil {
		return nil, err
	}
	return orderintake.NewOrderConsumer(
		consumer,
		acceptedordersredis.New(redisClient, cfg.AcceptedOrderRetention()),
		frontier,
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
	redisClient redis.UniversalClient,
	frontier *pendingpagevisitsredis.PendingPageVisits,
	cfg ServiceConfig,
	crawledPageAdmission *consumerbacklog.Admission,
	robotsExclusion pagevisitintake.RobotsExclusion,
	pendingPageVisitObserver pagevisitintake.PendingPageVisitObserver,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver pagevisitrecordsredis.PageVisitRecordObserver,
) *pagevisitintake.PageVisitConsumer {
	pageVisitor := buildPageVisitor(
		js,
		redisClient,
		frontier,
		cfg,
		pageFetchObserver,
		htmlPageReading,
		crawledPageObserver,
		pageVisitRecordObserver,
	)
	return pagevisitintake.NewPageVisitConsumer(
		frontier,
		takenpagevisitsredis.New(redisClient, cfg.TakenPageVisitRetention()),
		robotsExclusion,
		pageVisitAllowances(redisClient, cfg),
		acceptedordersredis.New(redisClient, cfg.AcceptedOrderRetention()),
		pageVisitor,
		pendingPageVisitObserver,
		crawledPageAdmission,
		cfg.FetchConcurrency,
	)
}

func buildRobotsExclusion(
	redisClient redis.UniversalClient,
	cfg ServiceConfig,
	robotsTxtFetchObserver robotsrulesorigin.RobotsTxtFetchObserver,
	robotsRulesObserver robotsrulesredis.RobotsRulesObserver,
) *robotsexclusion.RobotsExclusion {
	robotsTxtFetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		robotsrulesorigin.RobotsTxtParseLimit,
		cfg.FetchDeadline,
	)
	return robotsexclusion.New(
		robotsrulesredis.New(
			redisClient,
			robotsrulesorigin.New(robotsTxtFetch, cfg.FetchDeadline, robotsTxtFetchObserver),
			robotsRulesObserver,
		),
		cfg.RobotsProductToken,
	)
}

func pageVisitAllowances(
	redisClient redis.UniversalClient,
	cfg ServiceConfig,
) *pagevisitallowance.Allowances {
	return pagevisitallowance.New(
		pagevisitlimitsredis.New(redisClient, cfg.PageVisitLimitRetention(), cfg.MaxPerURL()),
		cfg.FetchRetryBounds(),
	)
}

func buildPageVisitor(
	js jetstream.JetStream,
	redisClient redis.UniversalClient,
	frontier *pendingpagevisitsredis.PendingPageVisits,
	cfg ServiceConfig,
	pageFetchObserver observedfetch.PageFetchObserver,
	htmlPageReading pagevisit.HTMLPageReading,
	crawledPageObserver crawledpagesjetstream.CrawledPagePublicationObserver,
	pageVisitRecordObserver pagevisitrecordsredis.PageVisitRecordObserver,
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
		pageVisitRecordsOf(redisClient, cfg, pageVisitRecordObserver),
		htmlPageReading,
		crawledpagesjetstream.New(js, crawledPageObserver),
		frontier,
	)
}

func pageVisitRecordsOf(
	redisClient redis.UniversalClient,
	cfg ServiceConfig,
	observer pagevisitrecordsredis.PageVisitRecordObserver,
) pagevisit.PageVisitRecords {
	if !cfg.SuppressesRecrawl() {
		return norecords.PageVisitRecords{}
	}
	return pagevisitrecordsredis.New(
		redisClient, cfg.PageVisitRetention(), wallclock.Clock{}, observer,
	)
}

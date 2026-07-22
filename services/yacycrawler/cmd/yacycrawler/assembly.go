package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordertraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/documentextractors/archive"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/documentextractors/html"
	orderreceiversjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/orderreceivers/jetstream"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlpolicies/alwaysdue"
	redirectresolversjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolvers/jetstream"
)

const (
	fetchRetryLimit       = 3
	fetchRetryFloor       = 500 * time.Millisecond
	fetchRetryCeil        = 30 * time.Second
	publishRetryFloor     = 500 * time.Millisecond
	publishRetryCeil      = 30 * time.Second
	maxDeferPerURL        = 3
	containerMaxDepth     = 4
	containerMaxDocuments = 1024
	archiveMaxMembers     = 1024
	opsReadHeaderLimit    = 10 * time.Second
	opsShutdownLimit      = 15 * time.Second
	ordersAckWait         = 30 * time.Second
	msgServiceStarted     = "crawler started"
	msgServiceStopped     = "crawler stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	metrics *prometheus.CrawlMetrics,
) error {
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if err := ensureStreams(ctx, js, cfg); err != nil {
		return err
	}

	consumer, err := ordersConsumer(ctx, js, cfg)
	if err != nil {
		return err
	}
	receiver, err := orderreceiversjetstream.NewOrderReceiver(ctx, consumer)
	if err != nil {
		return fmt.Errorf("start order receiver: %w", err)
	}

	fetch := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)
	resolve, err := redirectRecorder(ctx, js)
	if err != nil {
		return err
	}
	absorption, err := buildAbsorption(js, cfg, resolve, metrics)
	if err != nil {
		return err
	}

	visitor := pagevisit.NewPageVisit(
		fetch,
		alwaysdue.AlwaysDue{},
		absorption,
		metrics,
		wallclock.Clock{},
	)
	traverser := ordertraversal.NewOrderTraverser(
		traversalConfig(cfg),
		visitor,
		metrics,
		wallclock.Clock{},
	)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(metrics.Handler()),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.OrdersSubject),
		slog.Int("fetchConcurrency", cfg.FetchConcurrency),
		slog.Any("representations", publishedRepresentations(cfg)),
	)

	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			return ordersettlement.Run(
				runCtx,
				receiver.Deliveries(),
				traverser,
				metrics,
				wallclock.Clock{},
				ordersAckWait/2,
			)
		},
	)
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func ensureStreams(ctx context.Context, js jetstream.JetStream, cfg ServiceConfig) error {
	if err := yacycrawlcontract.EnsureOrdersStream(ctx, js, cfg.OrdersStreamSpec()); err != nil {
		return fmt.Errorf("ensure orders stream: %w", err)
	}
	for _, stream := range cfg.PageStreams {
		if err := yacycrawlcontract.EnsureCrawledPageStream(
			ctx, js, stream.Representation, stream.Stream,
		); err != nil {
			return fmt.Errorf("ensure page %s stream: %w", stream.Representation, err)
		}
	}
	if err := yacycrawlcontract.EnsureRedirectResolutionBucket(
		ctx, js, cfg.RedirectResolutionBucketSpec(),
	); err != nil {
		return fmt.Errorf("ensure redirect resolution bucket: %w", err)
	}
	return nil
}

func redirectRecorder(
	ctx context.Context,
	js jetstream.JetStream,
) (pageabsorption.RedirectResolver, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return nil, fmt.Errorf("open redirect resolution bucket: %w", err)
	}
	return redirectresolversjetstream.New(bucket), nil
}

func publishedRepresentations(cfg ServiceConfig) []string {
	names := make([]string, 0, len(cfg.PageStreams))
	for _, stream := range cfg.PageStreams {
		if stream.Published {
			names = append(names, string(stream.Representation))
		}
	}
	return names
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

func buildPageFeeds(js jetstream.JetStream, cfg ServiceConfig) []pageabsorption.Feed {
	subjects := make(
		map[yacycrawlcontract.PageRepresentationKind]string,
		len(cfg.PageStreams),
	)
	for _, stream := range cfg.PageStreams {
		if stream.Published {
			subjects[stream.Representation] = stream.Stream.Subject
		}
	}
	feeds := make([]pageabsorption.Feed, 0, len(subjects))
	for _, preset := range pageFeedCatalog() {
		subject, published := subjects[preset.representation]
		if !published {
			continue
		}
		feeds = append(feeds, preset.build(js, subject))
	}
	return feeds
}

func buildAbsorption(
	js jetstream.JetStream,
	cfg ServiceConfig,
	resolve pageabsorption.RedirectResolver,
	observer pageabsorption.Progress,
) (*pageabsorption.Absorption, error) {
	feeds := buildPageFeeds(js, cfg)
	graph := contentformatgraph.New(pageDerivationCatalog())
	if err := graph.Validate(feedContentFormats(feeds)); err != nil {
		return nil, err
	}
	extract, err := buildExtractor(cfg)
	if err != nil {
		return nil, err
	}
	return pageabsorption.New(
		graph,
		extract,
		resolve,
		feeds,
		observer,
		wallclock.Clock{},
		pageabsorption.Config{
			PublishRetryFloor:   publishRetryFloor,
			PublishRetryCeiling: publishRetryCeil,
		},
	), nil
}

func feedContentFormats(feeds []pageabsorption.Feed) []contentformatgraph.Format {
	formats := make([]contentformatgraph.Format, 0, len(feeds))
	for _, feed := range feeds {
		formats = append(formats, feed.ContentFormat())
	}
	return formats
}

func buildExtractor(cfg ServiceConfig) (pageabsorption.DocumentExtractor, error) {
	allow := allowedMediaTypes(cfg.ContentTypes)
	router := contentextraction.New(containerMaxDepth, containerMaxDocuments)

	htmlExtractor := html.New()
	for _, mediaType := range htmlExtractor.MediaTypes() {
		if allow == nil || allow[mediaType] {
			router.RegisterExtractor(mediaType, htmlExtractor)
		}
	}

	archiveContainer := archive.New(archiveMaxMembers, cfg.MaxBodyBytes)
	for _, mediaType := range archiveContainer.MediaTypes() {
		if allow == nil || allow[mediaType] {
			router.RegisterContainer(mediaType, archiveContainer)
		}
	}

	if router.RegisteredMediaTypes() == 0 {
		return nil, fmt.Errorf("%s: leaves no content extractor active", EnvContentTypes)
	}
	return router, nil
}

func allowedMediaTypes(contentTypes []string) map[string]bool {
	if len(contentTypes) == 0 {
		return nil
	}
	allow := make(map[string]bool, len(contentTypes))
	for _, mediaType := range contentTypes {
		allow[mediaType] = true
	}
	return allow
}

func traversalConfig(cfg ServiceConfig) ordertraversal.Config {
	return ordertraversal.Config{
		RunPageBudget:       cfg.RunPageBudget,
		FrontierCapacity:    cfg.FrontierCap,
		FetchRetryLimit:     fetchRetryLimit,
		FetchRetryFloor:     fetchRetryFloor,
		FetchRetryCeiling:   fetchRetryCeil,
		PublishRetryFloor:   publishRetryFloor,
		PublishRetryCeiling: publishRetryCeil,
		MaxDeferralsPerURL:  maxDeferPerURL,
		FetchConcurrency:    cfg.FetchConcurrency,
	}
}

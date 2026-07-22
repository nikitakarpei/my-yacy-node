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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/archivemember"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlrun"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawltraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmlpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/httpfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolution"
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

func RunService(ctx context.Context, cfg ServiceConfig, metrics *crawlmetrics.CrawlMetrics) error {
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
	receiver, err := orderintake.NewOrderReceiver(ctx, consumer)
	if err != nil {
		return fmt.Errorf("start order receiver: %w", err)
	}

	fetch := httpfetch.New(
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
	feeds := buildPageFeeds(js, cfg)
	renderings, err := buildPageRenderings(feeds)
	if err != nil {
		return err
	}

	extract, err := buildExtractor(cfg)
	if err != nil {
		return err
	}

	crawler := crawltraversal.NewCrawler(
		traversalConfig(cfg),
		fetch,
		extract,
		crawltraversal.AlwaysDue{},
		resolve,
		feeds,
		renderings,
		metrics,
		crawltraversal.SystemClock{},
	)
	engine := crawlrun.NewEngine(metrics, crawler)

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
			return engine.Run(runCtx, receiver.Deliveries())
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
) (crawlcapability.RedirectResolution, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return nil, fmt.Errorf("open redirect resolution bucket: %w", err)
	}
	return redirectresolution.New(bucket), nil
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

func buildPageFeeds(js jetstream.JetStream, cfg ServiceConfig) []crawlcapability.PageFeed {
	subjects := make(
		map[yacycrawlcontract.PageRepresentationKind]string,
		len(cfg.PageStreams),
	)
	for _, stream := range cfg.PageStreams {
		if stream.Published {
			subjects[stream.Representation] = stream.Stream.Subject
		}
	}
	feeds := make([]crawlcapability.PageFeed, 0, len(subjects))
	for _, preset := range pageFeedCatalog() {
		subject, published := subjects[preset.representation]
		if !published {
			continue
		}
		feeds = append(feeds, preset.build(js, subject))
	}
	return feeds
}

func buildPageRenderings(
	feeds []crawlcapability.PageFeed,
) ([]crawlcapability.PageRendering, error) {
	derivations := make(map[crawlcapability.PageContentFormat][]crawlcapability.PageRendering)
	for _, rendering := range pageRenderingCatalog() {
		derivations[rendering.Format()] = append(derivations[rendering.Format()], rendering)
	}
	renderings := []crawlcapability.PageRendering{}
	selected := map[crawlcapability.PageContentFormat]bool{}
	var require func(crawlcapability.PageFeed, crawlcapability.PageContentFormat) error
	require = func(feed crawlcapability.PageFeed, format crawlcapability.PageContentFormat) error {
		if format == crawlcapability.PageContentFormatDocumentHTML || selected[format] {
			return nil
		}
		candidates, ok := derivations[format]
		if !ok {
			return fmt.Errorf(
				"page %s feed reads %s content, which no rendering produces",
				feed.Representation(), format,
			)
		}
		selected[format] = true
		for _, candidate := range candidates {
			if err := require(feed, candidate.SourceFormat()); err != nil {
				return err
			}
			renderings = append(renderings, candidate)
		}
		return nil
	}
	for _, feed := range feeds {
		if err := require(feed, feed.ContentFormat()); err != nil {
			return nil, err
		}
	}
	return renderings, nil
}

func buildExtractor(cfg ServiceConfig) (crawlcapability.DocumentExtraction, error) {
	allow := allowedMediaTypes(cfg.ContentTypes)
	router := contentextraction.New(containerMaxDepth, containerMaxDocuments)

	html := htmlpage.New()
	for _, mediaType := range html.MediaTypes() {
		if allow == nil || allow[mediaType] {
			router.RegisterExtractor(mediaType, html)
		}
	}

	archive := archivemember.New(archiveMaxMembers, cfg.MaxBodyBytes)
	for _, mediaType := range archive.MediaTypes() {
		if allow == nil || allow[mediaType] {
			router.RegisterContainer(mediaType, archive)
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

func traversalConfig(cfg ServiceConfig) crawltraversal.Config {
	return crawltraversal.Config{
		RunPageBudget:       cfg.RunPageBudget,
		FrontierCapacity:    cfg.FrontierCap,
		FetchRetryLimit:     fetchRetryLimit,
		FetchRetryFloor:     fetchRetryFloor,
		FetchRetryCeiling:   fetchRetryCeil,
		PublishRetryFloor:   publishRetryFloor,
		PublishRetryCeiling: publishRetryCeil,
		MaxDeferralsPerURL:  maxDeferPerURL,
		FetchConcurrency:    cfg.FetchConcurrency,
		OwnershipInterval:   ordersAckWait / 2,
	}
}

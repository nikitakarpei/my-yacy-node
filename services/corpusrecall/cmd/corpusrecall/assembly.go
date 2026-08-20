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

	crawlorderplacersjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/crawlorderplacers/jetstream"
	disposedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/disposedpages/jetstream"
	markdownjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown/jetstream"
	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/progressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	recallreceiversgrpc "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallreceivers/grpc"
	redirectresolversjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/redirectresolvers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
	opsServerName      = "ops"
	msgServiceStarted  = "corpusrecall started"
	msgServiceStopped  = "corpusrecall stopped"
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	crawlJetStream, crawlConnection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = crawlConnection.Close() }()
	pageMarkdownJetStream, pageMarkdownConnection, err := jetstreamconnect.Open(
		cfg.PageMarkdownNATSURL,
	)
	if err != nil {
		return err
	}
	defer func() { _ = pageMarkdownConnection.Close() }()
	corpora, err := corporaFrom(ctx, pageMarkdownJetStream, cfg)
	if err != nil {
		return err
	}
	registry := prometheus.NewRegistry()
	metrics := progressobserversprometheus.NewRecallMetrics(registry)
	recaller, err := newRecaller(ctx, crawlJetStream, cfg, corpora, metrics)
	if err != nil {
		return err
	}
	receiver, err := recallreceiversgrpc.NewRecallReceiver(recaller, corpora, cfg.ListenAddr)
	if err != nil {
		return err
	}
	announceServiceStarted(ctx, cfg)
	err = servergroup.Run(ctx, opsShutdownLimit, opsServersFor(cfg, registry), receiver.Serve)
	announceServiceStopped(ctx)
	return err
}

func corporaFrom(
	ctx context.Context,
	pageMarkdownJetStream jetstream.JetStream,
	cfg ServiceConfig,
) ([]recall.Corpus, error) {
	markdownStore, err := pageMarkdownJetStream.ObjectStore(ctx, pagemarkdownstore.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page markdown bucket: %w", err)
	}
	return []recall.Corpus{markdownjetstream.NewCorpus(markdownStore, cfg.MaxResponseBytes)}, nil
}

func newRecaller(
	ctx context.Context,
	crawlJetStream jetstream.JetStream,
	cfg ServiceConfig,
	corpora []recall.Corpus,
	metrics *progressobserversprometheus.RecallMetrics,
) (*recall.Recaller, error) {
	redirects, err := redirectResolutionsFrom(ctx, crawlJetStream)
	if err != nil {
		return nil, err
	}
	disposedPages, err := disposedPagesFrom(ctx, crawlJetStream)
	if err != nil {
		return nil, err
	}
	return recall.NewRecaller(
		crawlorderplacersjetstream.NewCrawlOrderPlacement(crawlJetStream, cfg.OrdersSubject),
		redirects,
		disposedPages,
		corpora,
		metrics,
		recallConfigFrom(cfg),
	)
}

func redirectResolutionsFrom(
	ctx context.Context,
	crawlJetStream jetstream.JetStream,
) (*redirectresolversjetstream.RedirectResolutions, error) {
	bucket, err := crawlJetStream.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return nil, fmt.Errorf("open redirect resolution bucket: %w", err)
	}
	return redirectresolversjetstream.NewRedirectResolutions(bucket), nil
}

func disposedPagesFrom(
	ctx context.Context,
	crawlJetStream jetstream.JetStream,
) (*disposedpagesjetstream.DisposedPages, error) {
	bucket, err := crawlJetStream.KeyValue(ctx, yacycrawlcontract.DisposedPagesBucketName)
	if err != nil {
		return nil, fmt.Errorf("open disposed pages bucket: %w", err)
	}
	return disposedpagesjetstream.NewDisposedPages(bucket), nil
}

func recallConfigFrom(cfg ServiceConfig) recall.Config {
	return recall.Config{
		RecallLimit:         cfg.RecallLimit,
		PollInterval:        cfg.PollInterval,
		MaxRequestsInFlight: cfg.MaxInFlight,
	}
}

func announceServiceStarted(ctx context.Context, cfg ServiceConfig) {
	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listen", cfg.ListenAddr),
		slog.String("ordersSubject", cfg.OrdersSubject),
		slog.Duration("recallLimit", cfg.RecallLimit),
	)
}

func opsServersFor(
	cfg ServiceConfig,
	registry *prometheus.Registry,
) []servergroup.NamedServer {
	return []servergroup.NamedServer{{
		Name: opsServerName,
		Server: &http.Server{
			Addr: cfg.OpsAddr,
			Handler: opsmetrics.NewMux(
				promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
			),
			ReadHeaderTimeout: opsReadHeaderLimit,
		},
	}}
}

func announceServiceStopped(ctx context.Context) {
	slog.InfoContext(ctx, msgServiceStopped)
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"

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
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	corpora, err := corporaFrom(ctx, js, cfg)
	if err != nil {
		return err
	}
	metrics := progressobserversprometheus.NewRecallMetrics()
	recaller, err := newRecaller(ctx, js, cfg, corpora, metrics)
	if err != nil {
		return err
	}
	receiver, err := recallreceiversgrpc.NewRecallReceiver(recaller, corpora, cfg.ListenAddr)
	if err != nil {
		return err
	}
	announceServiceStarted(ctx, cfg)
	err = servergroup.Run(ctx, opsShutdownLimit, opsServersFor(cfg, metrics), receiver.Serve)
	announceServiceStopped(ctx)
	return err
}

func corporaFrom(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
) ([]recall.Corpus, error) {
	markdownStore, err := js.ObjectStore(ctx, pagemarkdownstore.BucketName)
	if err != nil {
		return nil, fmt.Errorf("open page markdown bucket: %w", err)
	}
	return []recall.Corpus{markdownjetstream.NewCorpus(markdownStore, cfg.MaxResponseBytes)}, nil
}

func newRecaller(
	ctx context.Context,
	js jetstream.JetStream,
	cfg ServiceConfig,
	corpora []recall.Corpus,
	metrics *progressobserversprometheus.RecallMetrics,
) (*recall.Recaller, error) {
	redirects, err := redirectResolutionsFrom(ctx, js)
	if err != nil {
		return nil, err
	}
	disposedPages, err := disposedPagesFrom(ctx, js)
	if err != nil {
		return nil, err
	}
	return recall.NewRecaller(
		crawlorderplacersjetstream.NewCrawlOrderPlacement(js, cfg.OrdersSubject),
		redirects,
		disposedPages,
		corpora,
		metrics,
		recallConfigFrom(cfg),
	), nil
}

func redirectResolutionsFrom(
	ctx context.Context,
	js jetstream.JetStream,
) (*redirectresolversjetstream.RedirectResolutions, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return nil, fmt.Errorf("open redirect resolution bucket: %w", err)
	}
	return redirectresolversjetstream.NewRedirectResolutions(bucket), nil
}

func disposedPagesFrom(
	ctx context.Context,
	js jetstream.JetStream,
) (*disposedpagesjetstream.DisposedPages, error) {
	bucket, err := js.KeyValue(ctx, yacycrawlcontract.DisposedPagesBucketName)
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
	metrics *progressobserversprometheus.RecallMetrics,
) []servergroup.NamedServer {
	return []servergroup.NamedServer{{
		Name: opsServerName,
		Server: &http.Server{
			Addr:              cfg.OpsAddr,
			Handler:           opsmetrics.NewMux(metrics.Exposition()),
			ReadHeaderTimeout: opsReadHeaderLimit,
		},
	}}
}

func announceServiceStopped(ctx context.Context) {
	slog.InfoContext(ctx, msgServiceStopped)
}

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

	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownstoremetrics"
	pagemarkdowncorporajetstream "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
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

	consumer, err := crawledPageConsumerFor(ctx, crawlJetStream, cfg)
	if err != nil {
		return err
	}
	corpus, err := pagemarkdowncorporajetstream.OpenCorpus(ctx, pageMarkdownJetStream)
	if err != nil {
		return err
	}

	registry := prometheus.NewRegistry()
	metrics := markdownstoremetrics.New(registry)
	intake := markdownintake.NewPageMarkdownConsumer(consumer, corpus, metrics, cfg.Concurrency)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("subject", cfg.CrawledPageSubject),
		slog.String("bucket", pagemarkdownstore.BucketName),
		slog.Int("concurrency", cfg.Concurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run page markdown consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpusmarkdown stopped")
	return err
}

func crawledPageConsumerFor(
	ctx context.Context,
	crawlJetStream jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := crawlJetStream.Stream(
		ctx,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindMarkdown),
	)
	if err != nil {
		return nil, fmt.Errorf("lookup crawled page stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.CrawledPageDurable,
		FilterSubject: cfg.CrawledPageSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create crawled page consumer: %w", err)
	}
	return consumer, nil
}

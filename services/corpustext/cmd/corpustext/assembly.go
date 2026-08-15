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

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/indexmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
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
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	stream, err := js.Stream(
		ctx,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindText),
	)
	if err != nil {
		return fmt.Errorf("lookup crawled page stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.CrawledPageDurable,
		FilterSubject: cfg.CrawledPageSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.Concurrency,
	})
	if err != nil {
		return fmt.Errorf("create crawled page consumer: %w", err)
	}

	selection, err := selectSearchIndex(cfg, http.DefaultClient)
	if err != nil {
		return fmt.Errorf("select search index: %w", err)
	}
	if err := selection.schema.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap search index schema: %w", err)
	}
	registry := prometheus.NewRegistry()
	metrics := indexmetrics.New(registry)
	intake := pageintake.NewCrawledPageConsumer(consumer, selection.index, metrics, cfg.Concurrency)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpustext started",
		slog.String("subject", cfg.CrawledPageSubject),
		slog.String("engine", cfg.SearchIndexEngine),
		slog.String("indexPrefix", selection.prefix),
		slog.Int("concurrency", cfg.Concurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run crawled page consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpustext stopped")
	return err
}

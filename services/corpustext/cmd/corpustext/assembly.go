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
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	js, conn, err := jetstreamconnect.Open(cfg.ScrapeRequestNATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	consumer, err := scrapeRequestConsumerFor(ctx, js, cfg)
	if err != nil {
		return err
	}
	scraper, err := textScraperFor(cfg)
	if err != nil {
		return err
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
	intake := pageintake.NewScrapeRequestConsumer(
		consumer, scraper, selection.index, metrics, cfg.Concurrency,
	)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpustext started",
		slog.String("subject", cfg.ScrapeRequestSubject),
		slog.String("engine", cfg.SearchIndexEngine),
		slog.String("indexPrefix", selection.prefix),
		slog.Int("concurrency", cfg.Concurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run scrape request consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpustext stopped")
	return err
}

func scrapeRequestConsumerFor(
	ctx context.Context,
	scrapeRequestJetStream jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := scrapeRequestJetStream.Stream(
		ctx,
		scraperequestcontract.ScrapeRequestsStreamName,
	)
	if err != nil {
		return nil, fmt.Errorf("lookup scrape requests stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.ScrapeRequestDurable,
		FilterSubject: cfg.ScrapeRequestSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create scrape request consumer: %w", err)
	}
	return consumer, nil
}

func textScraperFor(cfg ServiceConfig) (*pagescrape.Scraper, error) {
	return pagescrape.New(
		pagefetchershttp.New(
			cfg.ProxyURL,
			cfg.ProxyDialMode,
			cfg.UserAgent,
			cfg.MaxBodyBytes,
			cfg.FetchDeadline,
		),
		documentextraction.FormatReadableText,
	)
}

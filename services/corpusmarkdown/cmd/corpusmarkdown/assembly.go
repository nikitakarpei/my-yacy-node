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
	markdownrecallreceiversgrpc "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/grpc"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownstoremetrics"
	pagemarkdowncorporajetstream "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/jetstream"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
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
	scrapeRequestJetStream, scrapeRequestConnection, err := jetstreamconnect.Open(
		cfg.ScrapeRequestNATSURL,
	)
	if err != nil {
		return err
	}
	defer func() { _ = scrapeRequestConnection.Close() }()
	pageMarkdownJetStream, pageMarkdownConnection, err := jetstreamconnect.Open(
		cfg.PageMarkdownNATSURL,
	)
	if err != nil {
		return err
	}
	defer func() { _ = pageMarkdownConnection.Close() }()

	consumer, err := scrapeRequestConsumerFor(ctx, scrapeRequestJetStream, cfg)
	if err != nil {
		return err
	}
	corpus, err := pagemarkdowncorporajetstream.OpenCorpus(ctx, pageMarkdownJetStream)
	if err != nil {
		return err
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		return err
	}
	fetcher := pagefetchershttp.New(
		cfg.ProxyURL,
		cfg.ProxyDialMode,
		cfg.UserAgent,
		cfg.MaxBodyBytes,
		cfg.FetchDeadline,
	)

	registry := prometheus.NewRegistry()
	metrics := markdownstoremetrics.New(registry)
	intake := markdownintake.NewScrapeRequestConsumer(markdownintake.Config{
		Source:                         consumer,
		Fetcher:                        fetcher,
		FormatDerivations:              formatDerivations,
		Corpus:                         corpus,
		Progress:                       metrics,
		ScrapeRequestIntakeConcurrency: cfg.ScrapeRequestIntakeConcurrency,
	})

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	receiver := markdownrecallreceiversgrpc.NewMarkdownRecallReceiver(corpus, cfg.ListenAddr)

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("listen", cfg.ListenAddr),
		slog.String("subject", cfg.ScrapeRequestSubject),
		slog.String("bucket", pagemarkdownstore.BucketName),
		slog.Int("scrapeRequestIntakeConcurrency", cfg.ScrapeRequestIntakeConcurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run scrape request consumer: %w", err)
			}
			return nil
		},
		receiver.Serve,
	)
	slog.InfoContext(ctx, "corpusmarkdown stopped")
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
		MaxAckPending: cfg.ScrapeRequestIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create scrape request consumer: %w", err)
	}
	return consumer, nil
}

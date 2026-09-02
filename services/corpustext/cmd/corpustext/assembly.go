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

	intakeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakeprogressobservers/applog"
	intakeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakeprogressobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second

	corpus = "corpustext"
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	js, conn, err := jetstreamconnect.Open(cfg.PageOfferNATSURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	consumer, err := pageOfferConsumerFor(ctx, js, cfg)
	if err != nil {
		return err
	}
	formatDerivations, err := pageformats.New()
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
	intake := pageintake.NewOfferedPageConsumer(pageintake.Config{
		Source:            consumer,
		FormatDerivations: formatDerivations,
		SearchIndex:       selection.index,
		IntakeReceipts:    intakereceiptsnats.NewIntakeReceipts(conn, corpus),
		IntakeProgress: pageintake.IntakeProgressObservers{
			intakeprogressobserversapplog.IntakeProgressLog{},
			intakeprogressobserversprometheus.New(registry),
		},
		PageOfferIntakeConcurrency: cfg.PageOfferIntakeConcurrency,
	})

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpustext started",
		slog.String("engine", cfg.SearchIndexEngine),
		slog.String("indexPrefix", selection.prefix),
		slog.Int("pageOfferIntakeConcurrency", cfg.PageOfferIntakeConcurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run offered page consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpustext stopped")
	return err
}

func pageOfferConsumerFor(
	ctx context.Context,
	pageOffers jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := pageOffers.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
	if err != nil {
		return nil, fmt.Errorf("lookup page offers stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.PageOfferDurable,
		FilterSubject: pagescrapecontract.OfferedPageSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.PageOfferIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create page offer consumer: %w", err)
	}
	return consumer, nil
}

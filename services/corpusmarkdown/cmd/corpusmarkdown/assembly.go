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
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	markdownrecallreceivershttp "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/http"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintakeobservers/prometheus"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit = 10 * time.Second
	shutdownLimit   = 15 * time.Second
)

type Listeners struct {
	Recall net.Listener
	Ops    net.Listener
}

func RunService(ctx context.Context, cfg ServiceConfig, listeners Listeners) error {
	pageOfferJetStream, pageOfferConnection, err := jetstreamconnect.Open(cfg.PageOfferNATSURL)
	if err != nil {
		return err
	}
	defer pageOfferConnection.Close()

	consumer, err := pageOfferConsumerFor(ctx, pageOfferJetStream, cfg)
	if err != nil {
		return err
	}
	markdownCorpus, err := pagemarkdowncorporas3.OpenCorpus(
		ctx, cfg.PageMarkdownObjectStore, cfg.PageMarkdownBucket, wallclock.Clock{},
	)
	if err != nil {
		return err
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		return err
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "corpusmarkdown_info",
			Help: "Corpus Markdown application identity.",
		}, func() float64 { return 1 }),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	intake := pageintake.NewOfferedPageConsumer(pageintake.Config{
		Source:            consumer,
		FormatDerivations: formatDerivations,
		Corpus:            markdownCorpus,
		IntakeReceipts: intakereceiptsnats.NewIntakeReceipts(
			pageOfferConnection, pagescrapecontract.CorpusMarkdown,
			intakereceiptsnats.IntakeReceiptPublicationObservers{
				intakereceiptpublicationobserversapplog.IntakeReceiptPublicationLog{},
				intakereceiptpublicationobserversprometheus.New(registry),
			},
		),
		PageIntakeObserver: pageintake.PageIntakeObservers{
			pageintakeobserversapplog.PageIntakeLog{},
			pageintakeobserversprometheus.New(registry),
		},
		PageOfferIntakeConcurrency: cfg.PageOfferIntakeConcurrency,
	})

	opsServer := &http.Server{
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: readHeaderLimit,
	}

	recall := markdownrecall.NewPageMarkdownRecall(markdownCorpus)
	recallServer := &http.Server{
		Handler:           markdownrecallreceivershttp.NewMux(recall),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("listen", listeners.Recall.Addr().String()),
		slog.String("bucket", cfg.PageMarkdownBucket),
		slog.Int("pageOfferIntakeConcurrency", cfg.PageOfferIntakeConcurrency),
	)
	err = servergroup.Run(ctx, shutdownLimit,
		[]servergroup.NamedServer{
			{Name: "ops", Listener: listeners.Ops, Server: opsServer},
			{Name: "recall", Listener: listeners.Recall, Server: recallServer},
		},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run offered page consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpusmarkdown stopped")
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

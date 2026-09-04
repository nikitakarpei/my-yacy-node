package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	intakeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakeprogressobservers/applog"
	intakeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakeprogressobservers/prometheus"
	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	markdownrecallreceiversgrpc "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/grpc"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintake"
	pagemarkdowncorporajetstream "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	pageOfferJetStream, pageOfferConnection, err := jetstreamconnect.Open(cfg.PageOfferNATSURL)
	if err != nil {
		return err
	}
	defer pageOfferConnection.Close()
	pageMarkdownJetStream, pageMarkdownConnection, err := jetstreamconnect.Open(
		cfg.PageMarkdownNATSURL,
	)
	if err != nil {
		return err
	}
	defer pageMarkdownConnection.Close()

	consumer, err := pageOfferConsumerFor(ctx, pageOfferJetStream, cfg)
	if err != nil {
		return err
	}
	markdownCorpus, err := pagemarkdowncorporajetstream.OpenCorpus(ctx, pageMarkdownJetStream)
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

	recall := markdownrecall.NewPageMarkdownRecall(markdownCorpus)
	receiver := markdownrecallreceiversgrpc.NewMarkdownRecallReceiver(recall, cfg.ListenAddr)

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("listen", cfg.ListenAddr),
		slog.String("bucket", pagemarkdownstore.BucketName),
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
		receiver.Serve,
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

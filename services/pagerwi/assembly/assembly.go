package pagerwiassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageintakeobservers/prometheus"
	pageofferopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageofferopeningobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwi"
	pagerwipublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwipublicationobservers/applog"
	pagerwipublishersjetstream "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwipublishers/jetstream"
	pagerwistreamsjetstream "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwistreams/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shutdownLimit       = 15 * time.Second
	metricNamespace     = "pagerwi"
	corpus              = "pagerwi"
	pageRWIBacklogLimit = 1000
)

type Dial = func(ctx context.Context, network, address string) (net.Conn, error)

type Listeners struct{}

func ListenersFor(context.Context, ServiceConfig) (Listeners, error) {
	return Listeners{}, nil
}

func (Listeners) Close() error {
	return nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	_ Listeners,
	dial Dial,
	registry *prometheus.Registry,
	health *servicehealth.Health,
) error {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(cfg.PageRWIPartitionExponent)
	if err != nil {
		return fmt.Errorf("%s: %w", EnvPageRWIPartitionExponent, err)
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		return err
	}
	pageRWIBroker, pageRWIConnection, err := jetstreamconnect.OpenObserved(
		cfg.PageRWINATSURL,
		dial,
		health,
	)
	if err != nil {
		return err
	}
	defer pageRWIConnection.Close()
	pageRWIs, err := attachPageRWIsStream(ctx, pageRWIBroker, cfg)
	if err != nil {
		return err
	}
	pageOfferOpening := streamopening.Start(
		ctx,
		pageOfferOpenerFor(cfg, dial, health),
		wallclock.Clock{},
		pageofferopeningobserversapplog.PageOfferOpeningLog{},
	)
	defer pageOfferOpening.Close()

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: metricNamespace + "_info",
		Help: "pagerwi application identity.",
	}, func() float64 { return 1 }))
	intakeConfig := pageintake.OfferedPageConsumerConfig{
		FormatDerivations: formatDerivations,
		PageIndexBuilder:  pagerwi.NewIndexBuilder(partitions),
		PageRWIs: pagerwipublishersjetstream.NewPageRWIPublisher(
			pageRWIBroker, pagerwipublicationobserversapplog.PageRWIPublicationLog{},
		),
		PageRWIBacklog: consumerbacklog.New(
			pageRWIs, pageRWIBacklogLimit, wallclock.Clock{}, registry, metricNamespace,
		),
		PageIntakeObserver: pageintake.PageIntakeObservers{
			pageintakeobserversapplog.PageIntakeLog{},
			pageintakeobserversprometheus.New(registry),
		},
		PageOfferIntakeConcurrency: cfg.PageOfferIntakeConcurrency,
	}
	intakeReceiptPublicationObservers := intakereceiptsnats.IntakeReceiptPublicationObservers{
		intakereceiptpublicationobserversapplog.IntakeReceiptPublicationLog{},
		intakereceiptpublicationobserversprometheus.New(registry),
	}

	slog.InfoContext(ctx, "pagerwi started",
		slog.String("durable", cfg.PageOfferDurable),
		slog.Int("pageOfferIntakeConcurrency", cfg.PageOfferIntakeConcurrency),
		slog.Uint64("partitionExponent", uint64(cfg.PageRWIPartitionExponent)),
	)
	err = servergroup.Run(ctx, shutdownLimit, nil,
		func(runCtx context.Context) error {
			return takeInPageOffers(
				runCtx, pageOfferOpening, intakeConfig, intakeReceiptPublicationObservers, health,
			)
		},
	)
	slog.InfoContext(ctx, "pagerwi stopped")
	return err
}

func attachPageRWIsStream(
	ctx context.Context,
	broker jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Stream, error) {
	if cfg.ProvisionsStreams {
		return pagerwistreamsjetstream.CreatePageRWIsStream(
			ctx,
			broker,
			pagerwistreamsjetstream.PageRWIsStreamLimits{MaxBytes: cfg.PageRWIMaxBytes},
		)
	}
	return pagerwistreamsjetstream.OpenPageRWIsStream(ctx, broker)
}

func takeInPageOffers(
	ctx context.Context,
	pageOfferOpening *streamopening.Opening[pageOffers],
	intakeConfig pageintake.OfferedPageConsumerConfig,
	intakeReceiptPublicationObservers intakereceiptsnats.IntakeReceiptPublicationObservers,
	health *servicehealth.Health,
) error {
	offers, opened := pageOfferOpening.Opened(ctx)
	if !opened {
		return nil
	}
	defer offers.Close()
	health.MarkStarted()
	intakeConfig.OfferedPageSource = offers.consumer
	intakeConfig.IntakeReceipts = intakereceiptsnats.NewIntakeReceipts(
		offers.connection, corpus, intakeReceiptPublicationObservers,
	)
	if err := pageintake.NewOfferedPageConsumer(intakeConfig).Run(ctx); err != nil {
		return fmt.Errorf("run offered page consumer: %w", err)
	}
	return nil
}

type pageOffers struct {
	consumer   jetstream.Consumer
	connection *nats.Conn
}

func (offers pageOffers) Close() {
	offers.connection.Close()
}

func pageOfferOpenerFor(
	cfg ServiceConfig,
	dial Dial,
	health *servicehealth.Health,
) func(ctx context.Context) (pageOffers, error) {
	return func(ctx context.Context) (pageOffers, error) {
		js, connection, err := jetstreamconnect.OpenObserved(cfg.PageOfferNATSURL, dial, health)
		if err != nil {
			return pageOffers{}, err
		}
		consumer, err := attachPageOfferConsumer(ctx, js, cfg)
		if err != nil {
			connection.Close()
			return pageOffers{}, err
		}
		return pageOffers{consumer: consumer, connection: connection}, nil
	}
}

func attachPageOfferConsumer(
	ctx context.Context,
	jetStream jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := jetStream.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
	if err != nil {
		return nil, fmt.Errorf("open page offers stream: %w", err)
	}
	if cfg.ProvisionsStreams {
		return createPageOfferConsumer(ctx, stream, cfg)
	}
	return openPageOfferConsumer(ctx, stream, cfg.PageOfferDurable)
}

func createPageOfferConsumer(
	ctx context.Context,
	stream jetstream.Stream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
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

func openPageOfferConsumer(
	ctx context.Context,
	stream jetstream.Stream,
	pageOfferDurable string,
) (jetstream.Consumer, error) {
	consumer, err := stream.Consumer(ctx, pageOfferDurable)
	if err != nil {
		return nil, fmt.Errorf("open page offer consumer: %w", err)
	}
	return consumer, nil
}

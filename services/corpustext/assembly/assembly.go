package corpustextassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintakeobservers/prometheus"
	pageofferopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageofferopeningobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	textsearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearchobservers/applog"
	textsearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearchobservers/prometheus"
	textsearchreceivershttp "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearchreceivers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	serverReadHeaderLimit = 10 * time.Second
	serverShutdownLimit   = 15 * time.Second

	corpus = "corpustext"
)

type Listeners struct {
	Search net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	search, err := servergroup.Listen(ctx, cfg.SearchAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Search: search}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	pageOfferOpening := streamopening.Start(
		ctx,
		pageOfferOpenerFor(cfg),
		wallclock.Clock{},
		pageofferopeningobserversapplog.PageOfferOpeningLog{},
	)
	defer pageOfferOpening.Close()
	formatDerivations, err := pageformats.New()
	if err != nil {
		return err
	}
	selection, err := selectSearchIndex(cfg, http.DefaultClient)
	if err != nil {
		return fmt.Errorf("select search index: %w", err)
	}
	if err := selection.open(ctx); err != nil {
		return fmt.Errorf("open search index: %w", err)
	}
	defer selection.close(ctx)
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "corpustext_info",
		Help: "Corpus text application identity.",
	}, func() float64 { return 1 }))
	textSearch := textsearch.New(selection.matcher, textsearch.SearchObservers{
		textsearchobserversapplog.TextSearchLog{},
		textsearchobserversprometheus.New(registry),
	})
	intakeConfig := pageintake.Config{
		FormatDerivations: formatDerivations,
		SearchIndex:       selection.index,
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

	searchServer := &http.Server{
		Handler:           textsearchreceivershttp.NewMux(textSearch),
		ReadHeaderTimeout: serverReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpustext started",
		slog.String("engine", cfg.SearchIndexEngine),
		slog.String("indexPrefix", selection.prefix),
		slog.Int("pageOfferIntakeConcurrency", cfg.PageOfferIntakeConcurrency),
	)
	err = servergroup.Run(ctx, serverShutdownLimit,
		[]servergroup.NamedServer{
			{Name: "search", Listener: listeners.Search, Server: searchServer},
		},
		func(runCtx context.Context) error {
			return takeInPageOffers(
				runCtx, pageOfferOpening, intakeConfig, intakeReceiptPublicationObservers,
			)
		},
	)
	slog.InfoContext(ctx, "corpustext stopped")
	return err
}

func takeInPageOffers(
	ctx context.Context,
	pageOfferOpening *streamopening.Opening[pageOffers],
	intakeConfig pageintake.Config,
	intakeReceiptPublicationObservers intakereceiptsnats.IntakeReceiptPublicationObservers,
) error {
	offers, opened := pageOfferOpening.Opened(ctx)
	if !opened {
		return nil
	}
	defer offers.Close()
	intakeConfig.Source = offers.consumer
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

func pageOfferOpenerFor(cfg ServiceConfig) func(ctx context.Context) (pageOffers, error) {
	return func(ctx context.Context) (pageOffers, error) {
		js, connection, err := jetstreamconnect.Open(cfg.PageOfferNATSURL)
		if err != nil {
			return pageOffers{}, err
		}
		consumer, err := pageOfferConsumerFor(ctx, js, cfg)
		if err != nil {
			connection.Close()
			return pageOffers{}, err
		}
		return pageOffers{consumer: consumer, connection: connection}, nil
	}
}

func pageOfferConsumerFor(
	ctx context.Context,
	jetStream jetstream.JetStream,
	cfg ServiceConfig,
) (jetstream.Consumer, error) {
	stream, err := jetStream.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
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

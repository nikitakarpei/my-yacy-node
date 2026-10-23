package corpusmarkdownassembly

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

	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	markdownrecallreceivershttp "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/http"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintakeobservers/prometheus"
	pagemarkdowncorporafilesystem "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/filesystem"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
	pageofferopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageofferopeningobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit = 10 * time.Second
	shutdownLimit   = 15 * time.Second
)

type pageMarkdownCorpus interface {
	pageintake.PageMarkdownCorpus
	markdownrecall.PageMarkdownCorpus
}

type Listeners struct {
	Recall net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	recall, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Recall: recall}, nil
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
	markdownCorpus, err := openPageMarkdownCorpus(ctx, cfg)
	if err != nil {
		return err
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		return err
	}

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "corpusmarkdown_info",
		Help: "Corpus Markdown application identity.",
	}, func() float64 { return 1 }))
	intakeConfig := pageintake.Config{
		FormatDerivations: formatDerivations,
		Corpus:            markdownCorpus,
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

	recall := markdownrecall.NewPageMarkdownRecall(markdownCorpus)
	recallServer := &http.Server{
		Handler:           markdownrecallreceivershttp.NewMux(recall),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("listen", listeners.Recall.Addr().String()),
		slog.String("pageMarkdownLocation", pageMarkdownLocationOf(cfg)),
		slog.Int("pageOfferIntakeConcurrency", cfg.PageOfferIntakeConcurrency),
	)
	err = servergroup.Run(ctx, shutdownLimit,
		[]servergroup.NamedServer{
			{Name: "recall", Listener: listeners.Recall, Server: recallServer},
		},
		func(runCtx context.Context) error {
			return takeInPageOffers(
				runCtx, pageOfferOpening, intakeConfig, intakeReceiptPublicationObservers,
			)
		},
	)
	slog.InfoContext(ctx, "corpusmarkdown stopped")
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
		offers.connection, pagescrapecontract.CorpusMarkdown, intakeReceiptPublicationObservers,
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

func openPageMarkdownCorpus(ctx context.Context, cfg ServiceConfig) (pageMarkdownCorpus, error) {
	if cfg.PageMarkdownObjectStore == nil {
		return pagemarkdowncorporafilesystem.OpenCorpus(
			cfg.PageMarkdownDirectory,
			wallclock.Clock{},
		)
	}
	return pagemarkdowncorporas3.OpenCorpus(
		ctx, *cfg.PageMarkdownObjectStore, cfg.PageMarkdownBucket, wallclock.Clock{},
	)
}

func pageMarkdownLocationOf(cfg ServiceConfig) string {
	if cfg.PageMarkdownObjectStore == nil {
		return cfg.PageMarkdownDirectory
	}
	return cfg.PageMarkdownObjectStore.Endpoint + "/" + cfg.PageMarkdownBucket
}

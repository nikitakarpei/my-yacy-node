package yacynodeassembly

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageofferbroker"
	pageofferopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageofferopeningobservers/applog"
)

const corpus = "yacynode"

type pageOfferIntake struct {
	brokerOpening                     *streamopening.Opening[*pageofferbroker.PageOfferBroker]
	consumerConfig                    pageintake.OfferedPageConsumerConfig
	intakeReceiptPublicationObservers intakereceiptsnats.IntakeReceiptPublicationObservers
}

func openPageOfferIntake(
	ctx context.Context,
	config nodeconfiguration.PageOfferIntakeConfig,
	pages pageadmission.PageReceiver,
	registry prometheus.Registerer,
) (*pageOfferIntake, error) {
	formatDerivations, err := pageformats.New()
	if err != nil {
		return nil, err
	}

	return &pageOfferIntake{
		brokerOpening: streamopening.Start(
			ctx,
			pageOfferBrokerOpenerFor(config),
			wallclock.Clock{},
			pageofferopeningobserversapplog.PageOfferOpeningLog{},
		),
		consumerConfig: pageintake.OfferedPageConsumerConfig{
			FormatDerivations: formatDerivations,
			PageReceiver:      pages,
			PageIntakeObserver: pageintake.PageIntakeObservers{
				pageintakeobserversapplog.PageIntakeLog{},
				pageintakeobserversprometheus.New(registry),
			},
			PageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
		},
		intakeReceiptPublicationObservers: intakereceiptsnats.IntakeReceiptPublicationObservers{
			intakereceiptpublicationobserversapplog.IntakeReceiptPublicationLog{},
			intakereceiptpublicationobserversprometheus.New(registry),
		},
	}, nil
}

func pageOfferBrokerOpenerFor(
	config nodeconfiguration.PageOfferIntakeConfig,
) func(ctx context.Context) (*pageofferbroker.PageOfferBroker, error) {
	return func(ctx context.Context) (*pageofferbroker.PageOfferBroker, error) {
		broker, err := pageofferbroker.Open(ctx, pageofferbroker.Config{
			PageOfferNATSURL:           config.PageOfferNATSURL,
			PageOfferDurable:           config.PageOfferDurable,
			PageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
		})
		if err != nil {
			return nil, fmt.Errorf("open page offer broker: %w", err)
		}
		return broker, nil
	}
}

func (i *pageOfferIntake) Run(ctx context.Context) error {
	broker, opened := i.brokerOpening.Opened(ctx)
	if !opened {
		return nil
	}
	defer broker.Close()
	consumerConfig := i.consumerConfig
	consumerConfig.OfferedPageSource = broker.OfferedPages
	consumerConfig.IntakeReceipts = intakereceiptsnats.NewIntakeReceipts(
		broker.Connection, corpus, i.intakeReceiptPublicationObservers,
	)
	return pageintake.NewOfferedPageConsumer(consumerConfig).Run(ctx)
}

func (i *pageOfferIntake) Close() {
	i.brokerOpening.Close()
}

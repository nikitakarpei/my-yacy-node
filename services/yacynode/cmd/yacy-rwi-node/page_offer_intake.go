package main

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	intakeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakeprogressobservers/applog"
	intakeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakeprogressobservers/prometheus"
	intakereceiptpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceiptpublicationobservers/applog"
	intakereceiptpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceiptpublicationobservers/prometheus"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageofferbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const corpus = "yacynode"

type pageOfferIntake struct {
	broker   *pageofferbroker.PageOfferBroker
	consumer *pageintake.OfferedPageConsumer
}

func openPageOfferIntake(
	ctx context.Context,
	config nodeconfiguration.PageOfferIntakeConfig,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
	registry prometheus.Registerer,
) (*pageOfferIntake, error) {
	broker, err := pageofferbroker.Open(ctx, pageofferbroker.Config{
		PageOfferNATSURL:           config.PageOfferNATSURL,
		PageOfferDurable:           config.PageOfferDurable,
		PageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("open page offer broker: %w", err)
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		broker.Close()

		return nil, err
	}

	return &pageOfferIntake{
		broker: broker,
		consumer: pageintake.NewOfferedPageConsumer(
			pageintake.OfferedPageConsumerConfig{
				OfferedPageSource: broker.OfferedPages,
				FormatDerivations: formatDerivations,
				URLReceiver:       urls,
				PostingReceiver:   postings,
				IntakeReceipts: intakereceiptsnats.NewIntakeReceipts(
					broker.Connection, corpus,
					intakereceiptsnats.IntakeReceiptPublicationObservers{
						intakereceiptpublicationobserversapplog.IntakeReceiptPublicationLog{},
						intakereceiptpublicationobserversprometheus.New(registry),
					},
				),
				IntakeProgress: pageintake.IntakeProgressObservers{
					intakeprogressobserversapplog.IntakeProgressLog{},
					intakeprogressobserversprometheus.New(registry),
				},
				PageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
			}),
	}, nil
}

func (i *pageOfferIntake) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *pageOfferIntake) Close() {
	i.broker.Close()
}

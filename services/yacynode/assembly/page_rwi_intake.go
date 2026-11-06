package yacynodeassembly

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwibroker"
	pagerwiopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwiopeningobservers/applog"
)

type pageRWIIntake struct {
	brokerOpening  *streamopening.Opening[*pagerwibroker.PageRWIBroker]
	consumerConfig pageintake.PageRWIConsumerConfig
}

func openPageRWIIntake(
	ctx context.Context,
	config nodeconfiguration.PageRWIIntakeConfig,
	partitions yacymodel.DHTRingPartitions,
	pages pageadmission.PageReceiver,
	registry prometheus.Registerer,
) *pageRWIIntake {
	return &pageRWIIntake{
		brokerOpening: streamopening.Start(
			ctx,
			pageRWIBrokerOpenerFor(config),
			wallclock.Clock{},
			pagerwiopeningobserversapplog.PageRWIOpeningLog{},
		),
		consumerConfig: pageintake.PageRWIConsumerConfig{
			Partitions:   partitions,
			PageReceiver: pages,
			PageIntakeObserver: pageintake.PageIntakeObservers{
				pageintakeobserversapplog.PageIntakeLog{},
				pageintakeobserversprometheus.New(registry),
			},
			PageRWIIntakeConcurrency: config.PageRWIIntakeConcurrency,
		},
	}
}

func pageRWIBrokerOpenerFor(
	config nodeconfiguration.PageRWIIntakeConfig,
) func(ctx context.Context) (*pagerwibroker.PageRWIBroker, error) {
	return func(ctx context.Context) (*pagerwibroker.PageRWIBroker, error) {
		broker, err := pagerwibroker.Open(ctx, pagerwibroker.Config{
			PageRWINATSURL:           config.PageRWINATSURL,
			PageRWIDurable:           config.PageRWIDurable,
			PageRWIIntakeConcurrency: config.PageRWIIntakeConcurrency,
		})
		if err != nil {
			return nil, fmt.Errorf("open rwi broker: %w", err)
		}
		return broker, nil
	}
}

func (i *pageRWIIntake) Run(ctx context.Context) error {
	broker, opened := i.brokerOpening.Opened(ctx)
	if !opened {
		return nil
	}
	defer broker.Close()
	consumerConfig := i.consumerConfig
	consumerConfig.PageRWISource = broker.PageRWIs
	return pageintake.NewPageRWIConsumer(consumerConfig).Run(ctx)
}

func (i *pageRWIIntake) Close() {
	i.brokerOpening.Close()
}

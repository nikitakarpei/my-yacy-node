package yacynodeassembly

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/streamopening"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/applog"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/prometheus"
	pagerwiopeningobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwiopeningobservers/applog"
)

type pageRWIIntake struct {
	streamOpening  *streamopening.Opening[pageRWIs]
	consumerConfig pageintake.PageRWIConsumerConfig
}

func openPageRWIIntake(
	ctx context.Context,
	config nodeconfiguration.PageRWIIntakeConfig,
	dial Dial,
	partitions yacymodel.DHTRingPartitions,
	pages pageadmission.PageReceiver,
	registry prometheus.Registerer,
) *pageRWIIntake {
	return &pageRWIIntake{
		streamOpening: streamopening.Start(
			ctx,
			pageRWIOpenerFor(config, dial),
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

func (i *pageRWIIntake) Run(ctx context.Context) error {
	opened, isOpen := i.streamOpening.Opened(ctx)
	if !isOpen {
		return nil
	}
	defer opened.Close()
	consumerConfig := i.consumerConfig
	consumerConfig.PageRWISource = opened.consumer
	return pageintake.NewPageRWIConsumer(consumerConfig).Run(ctx)
}

func (i *pageRWIIntake) Close() {
	i.streamOpening.Close()
}

type pageRWIs struct {
	consumer   jetstream.Consumer
	connection *nats.Conn
}

func (rwis pageRWIs) Close() {
	rwis.connection.Close()
}

func pageRWIOpenerFor(
	config nodeconfiguration.PageRWIIntakeConfig,
	dial Dial,
) func(ctx context.Context) (pageRWIs, error) {
	return func(ctx context.Context) (pageRWIs, error) {
		js, connection, err := jetstreamconnect.Open(config.PageRWINATSURL, dial)
		if err != nil {
			return pageRWIs{}, err
		}
		consumer, err := pageRWIConsumerFor(ctx, js, config)
		if err != nil {
			connection.Close()
			return pageRWIs{}, err
		}
		return pageRWIs{consumer: consumer, connection: connection}, nil
	}
}

func pageRWIConsumerFor(
	ctx context.Context,
	js jetstream.JetStream,
	config nodeconfiguration.PageRWIIntakeConfig,
) (jetstream.Consumer, error) {
	stream, err := js.Stream(ctx, pagerwicontract.PageRWIsStreamName)
	if err != nil {
		return nil, fmt.Errorf("open rwi stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       config.PageRWIDurable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: pagerwicontract.PageRWISubject,
		MaxAckPending: config.PageRWIIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create rwi consumer: %w", err)
	}
	return consumer, nil
}

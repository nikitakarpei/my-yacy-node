// Package pageofferbroker is the node's NATS JetStream edge to the scrape service. It is the
// only place that speaks the broker protocol: it opens the durable consumer the node reads
// offered pages through, and holds the connection the node sends its intake receipts on.
// Open wires the connection; Close releases it.
package pageofferbroker

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	PageOfferNATSURL           string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
}

type PageOfferBroker struct {
	Connection   *nats.Conn
	OfferedPages jetstream.Consumer
}

func Open(ctx context.Context, cfg Config) (*PageOfferBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.PageOfferNATSURL)
	if err != nil {
		return nil, err
	}

	offeredPages, err := offeredPageConsumerFor(ctx, js, cfg)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &PageOfferBroker{Connection: conn, OfferedPages: offeredPages}, nil
}

func offeredPageConsumerFor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg Config,
) (jetstream.Consumer, error) {
	stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
	if err != nil {
		return nil, fmt.Errorf("open page offers stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.PageOfferDurable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: pagescrapecontract.OfferedPageSubject,
		MaxAckPending: cfg.PageOfferIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create page offer consumer: %w", err)
	}

	return consumer, nil
}

func (b *PageOfferBroker) Close() {
	b.Connection.Close()
}

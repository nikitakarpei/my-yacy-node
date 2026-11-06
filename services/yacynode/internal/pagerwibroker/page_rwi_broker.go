// Package pagerwibroker is the node's NATS JetStream edge to the pagerwi service. It is the
// only place that speaks the broker protocol: it opens the durable consumer the node reads the
// RWIs of pages through. Open wires the connection; Close releases it.
package pagerwibroker

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	PageRWINATSURL           string
	PageRWIDurable           string
	PageRWIIntakeConcurrency int
}

type PageRWIBroker struct {
	connection *nats.Conn
	PageRWIs   jetstream.Consumer
}

func Open(ctx context.Context, cfg Config) (*PageRWIBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.PageRWINATSURL)
	if err != nil {
		return nil, err
	}

	pageRWIs, err := pageRWIConsumerFor(ctx, js, cfg)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &PageRWIBroker{connection: conn, PageRWIs: pageRWIs}, nil
}

func pageRWIConsumerFor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg Config,
) (jetstream.Consumer, error) {
	stream, err := js.Stream(ctx, pagerwicontract.PageRWIsStreamName)
	if err != nil {
		return nil, fmt.Errorf("open rwi stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.PageRWIDurable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: pagerwicontract.PageRWISubject,
		MaxAckPending: cfg.PageRWIIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create rwi consumer: %w", err)
	}

	return consumer, nil
}

func (b *PageRWIBroker) Close() {
	b.connection.Close()
}

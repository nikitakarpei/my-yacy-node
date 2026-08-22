// Package crawlbroker is the node's NATS JetStream edge to the crawl fleet. It is the
// only place that speaks the broker protocol: it opens the durable consumer the node
// reads scrape requests through. Open wires the connection; Close releases it.
package crawlbroker

import (
	"context"
	"fmt"
	"io"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	ScrapeRequestNATSURL string
	ScrapeRequestSubject string
	ScrapeRequestDurable string
	Concurrency          int
}

type CrawlBroker struct {
	conn           io.Closer
	ScrapeRequests jetstream.Consumer
}

func Open(ctx context.Context, cfg Config) (*CrawlBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.ScrapeRequestNATSURL)
	if err != nil {
		return nil, err
	}

	scrapeRequests, err := scrapeRequestConsumerFor(ctx, js, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &CrawlBroker{conn: conn, ScrapeRequests: scrapeRequests}, nil
}

func scrapeRequestConsumerFor(
	ctx context.Context,
	js jetstream.JetStream,
	cfg Config,
) (jetstream.Consumer, error) {
	stream, err := js.Stream(ctx, scraperequestcontract.ScrapeRequestsStreamName)
	if err != nil {
		return nil, fmt.Errorf("open scrape requests stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.ScrapeRequestDurable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: cfg.ScrapeRequestSubject,
		MaxAckPending: cfg.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("create scrape request consumer: %w", err)
	}

	return consumer, nil
}

func (b *CrawlBroker) Close() {
	_ = b.conn.Close()
}

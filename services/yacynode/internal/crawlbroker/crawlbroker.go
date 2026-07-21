// Package crawlbroker is the node's NATS JetStream edge to the crawl fleet. It is
// the only place that speaks the broker protocol: it publishes crawl orders and
// receives ingest batches, exposing them as the plain ports the inner packages
// consume. Open wires the connection; Close releases it. The orders stream
// belongs to yacycrawler; until yacycrawler has created it, publishing an order
// fails.
package crawlbroker

import (
	"context"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	NATSURL       string
	OrdersSubject string
	IngestSubject string
	IngestDurable string
}

type CrawlBroker struct {
	conn   io.Closer
	Orders *OrderPublisher
	Ingest *IngestReceiver
}

func Open(ctx context.Context, cfg Config) (*CrawlBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, err
	}

	ingest, err := newIngestReceiver(ctx, js, cfg.IngestDurable, cfg.IngestSubject)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &CrawlBroker{
		conn:   conn,
		Orders: newOrderPublisher(js, cfg.OrdersSubject),
		Ingest: ingest,
	}, nil
}

func (b *CrawlBroker) Close() {
	_ = b.conn.Close()
}

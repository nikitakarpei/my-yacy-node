// Package crawlbroker is the node's NATS JetStream edge to the crawl fleet. It is
// the only place that speaks the broker protocol: it receives ingest batches,
// exposing them as the plain port the inner packages consume. Open wires the
// connection; Close releases it.
package crawlbroker

import (
	"context"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	NATSURL       string
	IngestSubject string
	IngestDurable string
}

type CrawlBroker struct {
	conn   io.Closer
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
		Ingest: ingest,
	}, nil
}

func (b *CrawlBroker) Close() {
	_ = b.conn.Close()
}

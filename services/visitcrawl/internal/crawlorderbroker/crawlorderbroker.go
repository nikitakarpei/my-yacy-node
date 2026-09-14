// Package crawlorderbroker is the visit intake's NATS JetStream edge to the
// crawl fleet. It is the only place that speaks the broker protocol: it opens
// the connection and exposes PublishingCrawlOrderPlacer, which publishes each
// crawl order to the orders subject and reports the publication to its
// observer. Open wires the connection; Close releases it. The orders stream
// belongs to yacycrawler; until yacycrawler has created it, publishing fails.
package crawlorderbroker

import (
	"context"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	NATSURL            string
	CrawlOrdersSubject string
}

type CrawlOrderBroker struct {
	conn        *nats.Conn
	OrderPlacer *PublishingCrawlOrderPlacer
}

func Open(
	_ context.Context,
	cfg Config,
	observer CrawlOrderPublicationObserver,
) (*CrawlOrderBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, err
	}

	return &CrawlOrderBroker{
		conn:        conn,
		OrderPlacer: newPublishingCrawlOrderPlacer(js, cfg.CrawlOrdersSubject, observer),
	}, nil
}

func (b *CrawlOrderBroker) Close() {
	b.conn.Close()
}

// Package crawlorderbroker is the visit intake's NATS JetStream edge to the
// crawl fleet. It is the only place that speaks the broker protocol: it opens
// the connection and exposes OrderPlacement as the plain port the visit intake
// places orders through. Open wires the connection; Close releases it. The
// orders stream belongs to yacycrawler; until yacycrawler has created it,
// placing an order fails.
package crawlorderbroker

import (
	"context"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Config struct {
	NATSURL       string
	OrdersSubject string
}

type CrawlOrderBroker struct {
	conn   io.Closer
	Orders *OrderPlacement
}

func Open(_ context.Context, cfg Config) (*CrawlOrderBroker, error) {
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, err
	}

	return &CrawlOrderBroker{
		conn:   conn,
		Orders: newOrderPlacement(js, cfg.OrdersSubject),
	}, nil
}

func (b *CrawlOrderBroker) Close() {
	_ = b.conn.Close()
}

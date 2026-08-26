// Package jetstream publishes a scrape request for every capture the command selects. The
// scrape requests stream belongs to the stack that reads it; until that stream exists,
// publishing fails.
package jetstream

import (
	"context"
	"fmt"
	"io"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Publisher struct {
	stream jetstream.JetStream
	conn   io.Closer
}

func Open(natsURL string) (*Publisher, error) {
	stream, conn, err := jetstreamconnect.Open(natsURL)
	if err != nil {
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error {
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{CanonicalURL: canonicalURL},
	)
	if err != nil {
		return fmt.Errorf("marshal scrape request %s: %w", canonicalURL, err)
	}
	if _, err := p.stream.Publish(
		ctx,
		scraperequestcontract.ScrapeRequestSubject,
		data,
	); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", canonicalURL, err)
	}
	return nil
}

func (p *Publisher) Close() {
	_ = p.conn.Close()
}

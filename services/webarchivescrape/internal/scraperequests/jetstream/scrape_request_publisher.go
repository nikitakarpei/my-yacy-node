// Package jetstream publishes a scrape request for every capture the command selects. The
// scrape requests stream belongs to the stack that reads it; until that stream exists,
// publishing fails.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type Publisher struct {
	stream jetstream.JetStream
	conn   *nats.Conn
}

func Open(natsURL string) (*Publisher, error) {
	stream, conn, err := jetstreamconnect.Open(natsURL)
	if err != nil {
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) Publish(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	replayURL canonicalurl.CanonicalURL,
) error {
	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL, FetchURL: replayURL},
	)
	if err != nil {
		return fmt.Errorf("marshal scrape request %s: %w", pageURL, err)
	}
	if _, err := p.stream.Publish(
		ctx,
		pagescrapecontract.ScrapeRequestSubject,
		data,
	); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", pageURL, err)
	}
	return nil
}

func (p *Publisher) Close() {
	p.conn.Close()
}
